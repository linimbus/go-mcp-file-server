package main

import (
	"fmt"
	"sync"
	"syscall"
	"unsafe"

	"github.com/astaxie/beego/logs"
	"golang.org/x/sys/windows"
)

const (
	FILE_LIST_DIRECTORY        = 0x0001
	FILE_SHARE_READ            = 0x00000001
	FILE_SHARE_WRITE           = 0x00000002
	FILE_SHARE_DELETE          = 0x00000004
	OPEN_EXISTING              = 3
	FILE_FLAG_BACKUP_SEMANTICS = 0x02000000
)

type FILE_NOTIFY_INFORMATION struct {
	NextEntryOffset uint32
	Action          uint32
	FileNameLength  uint32
	FileName        [1]uint16 // Dynamic array, actual length is determined by FileNameLength
}

const (
	FILE_ADD uint32 = iota + 1
	FILE_REMOVE
	FILE_MODIFIED
	FILE_RENAME_OLD
	FILE_RENAME_NEW
	FILE_EVENT_MAX
)

type FileEvent struct {
	sync.WaitGroup

	shutdown bool
	config   Config
	sql      *SQLiteDB
	handles  []windows.Handle
}

func WindowCreateFile(driveName string) (windows.Handle, error) {
	handle, err := windows.CreateFile(
		windows.StringToUTF16Ptr(driveName),
		FILE_LIST_DIRECTORY,
		FILE_SHARE_READ|FILE_SHARE_WRITE|FILE_SHARE_DELETE,
		nil,
		OPEN_EXISTING,
		FILE_FLAG_BACKUP_SEMANTICS,
		0,
	)
	if err != nil {
		logs.Warning("call windows.CreateFile %s failed, %s", driveName, err.Error())
		return 0, fmt.Errorf("notify event directory %s failed, %s", driveName, err.Error())
	}
	return handle, nil
}

func ReadDirectoryChanges(handle windows.Handle, buffer []byte) (uint32, error) {
	var bytesReturned uint32
	err := windows.ReadDirectoryChanges(
		handle,
		&buffer[0],
		uint32(len(buffer)),
		true,
		windows.FILE_NOTIFY_CHANGE_FILE_NAME|
			windows.FILE_NOTIFY_CHANGE_DIR_NAME|
			windows.FILE_NOTIFY_CHANGE_ATTRIBUTES|
			windows.FILE_NOTIFY_CHANGE_SIZE|
			windows.FILE_NOTIFY_CHANGE_LAST_WRITE|
			windows.FILE_NOTIFY_CHANGE_SECURITY,
		&bytesReturned,
		nil,
		0,
	)
	if err != nil {
		logs.Warning("call windows.ReadDirectoryChanges failed, %s", err.Error())
	}
	return bytesReturned, err
}

func NewFileEvent(s *SQLiteDB, config Config) (*FileEvent, error) {
	e := &FileEvent{handles: make([]windows.Handle, 0), sql: s, config: config}

	for _, v := range config.SearchDrives {
		if !v.Enable {
			continue
		}
		handle, err := WindowCreateFile(v.Name)
		if err != nil {
			return nil, err
		}
		e.handles = append(e.handles, handle)
	}

	e.Add(len(e.handles))
	for _, handle := range e.handles {
		go e.listenDriveTask(handle, config.CacheLength)
	}

	return e, nil
}

func (e *FileEvent) Close() {
	e.shutdown = true
	for _, handle := range e.handles {
		windows.CloseHandle(handle)
	}
	e.Wait()
}

func (e *FileEvent) parseEvents(data []byte) {
	var offset uint32 = 0
	for {
		event := (*FILE_NOTIFY_INFORMATION)(unsafe.Pointer(&data[offset]))

		filePath := syscall.UTF16ToString((*[1 << 20]uint16)(unsafe.Pointer(&event.FileName))[:event.FileNameLength/2])

		if event.Action < FILE_EVENT_MAX {
			switch event.Action {
			case FILE_ADD:
				fallthrough
			case FILE_MODIFIED:
				fallthrough
			case FILE_RENAME_NEW:
				{
					fileInfo, err := NewFileInfo(filePath)
					if err != nil {
						logs.Warning("load %s file info failed, %s", filePath, err.Error())
					} else {
						e.sql.Notify() <- FileNotify{Event: event.Action, File: *fileInfo}
					}
				}
			case FILE_REMOVE:
				fallthrough
			case FILE_RENAME_OLD:
				{
					e.sql.Notify() <- FileNotify{Event: event.Action, File: FileInfo{
						Path: filePath,
					}}
				}
			}
		}
		// Check if there are more events
		if event.NextEntryOffset == 0 {
			break
		}
		offset += event.NextEntryOffset
	}
}

func (e *FileEvent) listenDriveTask(handle windows.Handle, cacheLength uint32) {
	defer e.Done()

	logs.Info("listen drive file change task startup")

	buffer := make([]byte, cacheLength)

	for {
		if e.shutdown {
			break
		}

		bytesReturned, err := ReadDirectoryChanges(handle, buffer)
		if err == nil {
			e.parseEvents(buffer[:bytesReturned])
		}
	}

	logs.Info("listen drive file change task done")
}
