package main

import (
	"fmt"
	"log"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// 定义常量和结构体
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
	FileName        [1]uint16 // 动态数组，实际长度由 FileNameLength 决定
}

const (
	FILE_ADD uint32 = iota + 1
	FILE_REMOVE
	FILE_MODIFIED
	FILE_RENAME_OLD
	FILE_RENAME_NEW
)

// 定义事件类型
var actionMap = map[uint32]string{
	1: "Added",
	2: "Removed",
	3: "Modified",
	4: "RenamedOldName",
	5: "RenamedNewName",
}

// func main() {
// 	// 监听的目标目录
// 	dir := `C:\`
// 	listenDirectory(dir)
// }

func listenDirectory(dir string) {
	// 打开目录
	handle, err := windows.CreateFile(
		windows.StringToUTF16Ptr(dir),
		FILE_LIST_DIRECTORY,
		FILE_SHARE_READ|FILE_SHARE_WRITE|FILE_SHARE_DELETE,
		nil,
		OPEN_EXISTING,
		FILE_FLAG_BACKUP_SEMANTICS,
		0,
	)
	if err != nil {
		log.Fatalf("Failed to open directory: %v", err)
	}
	defer windows.CloseHandle(handle)

	// 创建缓冲区
	buffer := make([]byte, 1024*1024) // 1MB 缓冲区

	for {
		var bytesReturned uint32
		err := windows.ReadDirectoryChanges(
			handle,
			&buffer[0],
			uint32(len(buffer)),
			true, // 是否递归监听子目录
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
			log.Fatalf("ReadDirectoryChanges failed: %v", err)
		}

		// 解析事件
		parseEvents(buffer[:bytesReturned])
	}
}

func parseEvents(data []byte) {
	var offset uint32 = 0
	for {
		event := (*FILE_NOTIFY_INFORMATION)(unsafe.Pointer(&data[offset]))

		// 提取文件名
		fileName := syscall.UTF16ToString((*[1 << 20]uint16)(unsafe.Pointer(&event.FileName))[:event.FileNameLength/2])

		// 获取事件类型
		action := actionMap[event.Action]
		fmt.Printf("Event: %s, File: %s\n", action, fileName)

		// 检查是否还有更多事件
		if event.NextEntryOffset == 0 {
			break
		}
		offset += event.NextEntryOffset
	}
}
