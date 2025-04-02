package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/astaxie/beego/logs"
	_ "github.com/mattn/go-sqlite3" // 引入 SQLite 驱动
)

var DATABASE_FILE = "sqlite3.db"
var NOTIFY_CACHE_LENGTH = 100000

type FileInfo struct {
	Name  string // 文件名
	IsDir int    // 是否目录: 0 表示文件，1 表示目录
	Path  string // 文件路径
	Ext   string // 扩展名

	Drive   string    // 磁盘符号
	ModTime time.Time // 修改时间
	Size    int64     // 文件大小
}

func (f *FileInfo) ToHeader() []string {
	return []string{
		"filename", "is directory", "filepath", "file extension name", "drive name", "file modification time", "file size",
	}
}

func (f *FileInfo) ToList() []string {
	isDir := "false"
	if f.IsDir > 0 {
		isDir = "true"
	}
	return []string{
		f.Name, isDir, f.Path, f.Ext, f.Drive,
		TimeStampGet(f.ModTime), ByteView(f.Size),
	}
}

func NewFileInfo(filePath string) (*FileInfo, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}
	isDir := 0
	if fileInfo.IsDir() {
		isDir = 1
	}
	return &FileInfo{
		Name:    filepath.Base(filePath),
		IsDir:   isDir,
		Path:    filePath,
		Ext:     filepath.Ext(filePath),
		Drive:   filePath[:3],
		ModTime: fileInfo.ModTime(),
		Size:    fileInfo.Size(),
	}, nil
}

type FileNotify struct {
	Event uint32
	File  FileInfo
}

var TABLE_CREATE_SQL = `
CREATE TABLE IF NOT EXISTS file_info (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL,
	is_dir INTEGER NOT NULL,
	path TEXT NOT NULL UNIQUE,
	ext TEXT NOT NULL,
	drive TEXT NOT NULL,
	mod_time TEXT NOT NULL,
	size INTEGER NOT NULL
);`

var TABLE_INDEX_SQL = `
CREATE INDEX IF NOT EXISTS idx_file_info_name ON file_info (name, ext);
`

var TABLE_INSERT_SQL = `
INSERT INTO file_info (name, is_dir, path, ext, drive, mod_time, size)
VALUES (?, ?, ?, ?, ?, ?, ?)`

var TABLE_UPDATE_SQL = `
UPDATE file_info
SET name = ?, is_dir = ?, ext = ?, drive = ?, mod_time = ?, size = ?
WHERE path = ?`

var TABLE_DELETE_SQL = `
DELETE FROM file_info WHERE path = ?`

var TABLE_QUERY_LIKE_SQL = `
SELECT name, is_dir, path, ext, drive, mod_time, size FROM file_info
WHERE name LIKE ?
LIMIT ?`

var TABLE_QUERY_GLOB_SQL = `
SELECT name, is_dir, path, ext, drive, mod_time, size FROM file_info
WHERE name GLOB ?
LIMIT ?`

type SQLiteDB struct {
	sync.WaitGroup
	sync.RWMutex

	db     *sql.DB
	notify chan interface{}
}

func NewSQLiteDB() (*SQLiteDB, error) {
	db, err := sql.Open("sqlite3", filepath.Join(ConfigDirGet(), DATABASE_FILE))
	if err != nil {
		return nil, fmt.Errorf("open sqlite3 database failed, %s", err.Error())
	}
	_, err = db.Exec(TABLE_CREATE_SQL)
	if err != nil {
		return nil, fmt.Errorf("create table failed, %s", err.Error())
	}
	_, err = db.Exec(TABLE_INDEX_SQL)
	if err != nil {
		return nil, fmt.Errorf("create index failed, %s", err.Error())
	}

	s := &SQLiteDB{db: db, notify: make(chan interface{}, NOTIFY_CACHE_LENGTH)}
	s.Add(1)
	go recvNotifyTask(s)
	return s, nil
}

func (s *SQLiteDB) Reset() error {
	s.Lock()
	defer s.Unlock()

	_, err := s.db.Exec("DROP INDEX idx_file_info_name;")
	if err != nil {
		return fmt.Errorf("drop index failed, %s", err.Error())
	}
	_, err = s.db.Exec("DROP TABLE IF EXISTS file_info;")
	if err != nil {
		return fmt.Errorf("drop table failed, %s", err.Error())
	}
	_, err = s.db.Exec(TABLE_CREATE_SQL)
	if err != nil {
		return fmt.Errorf("create table failed, %s", err.Error())
	}
	_, err = s.db.Exec(TABLE_INDEX_SQL)
	if err != nil {
		return fmt.Errorf("create index failed, %s", err.Error())
	}
	logs.Info("sql reset database success")
	return nil
}

func recvNotifyTask(s *SQLiteDB) {
	defer s.Done()

	logs.Info("sql recvice notify task startup")

	for {
		msg, ok := <-s.notify
		if !ok {
			break
		}
		if _, ok := msg.(struct{}); ok {
			break
		}

		s.Lock()

		if fileNotify, ok := msg.(*FileNotify); ok {
			logs.Info("recive file event: %d path: %s", fileNotify.Event, fileNotify.File.Path)

			switch fileNotify.Event {
			case FILE_ADD:
				{
					_, err := s.db.Exec(TABLE_INSERT_SQL,
						fileNotify.File.Name, fileNotify.File.IsDir,
						fileNotify.File.Path, fileNotify.File.Ext,
						fileNotify.File.Drive,
						fileNotify.File.ModTime.Format(time.RFC3339),
						fileNotify.File.Size)
					if err != nil {
						logs.Warning("insert sql failed, %s", err.Error())
					}
				}
			case FILE_MODIFIED:
				{
					_, err := s.db.Exec(TABLE_UPDATE_SQL,
						fileNotify.File.Name, fileNotify.File.IsDir,
						fileNotify.File.Ext, fileNotify.File.Drive,
						fileNotify.File.ModTime.Format(time.RFC3339),
						fileNotify.File.Size, fileNotify.File.Path)
					if err != nil {
						logs.Warning("insert sql failed, %s", err.Error())
					}
				}
			case FILE_REMOVE:
				{
					_, err := s.db.Exec(TABLE_DELETE_SQL, fileNotify.File.Path)
					if err != nil {
						logs.Warning("delete sql failed, %s", err.Error())
					}
				}

			case FILE_RENAME_OLD:
				{
					_, err := s.db.Exec(TABLE_DELETE_SQL, fileNotify.File.Path)
					if err != nil {
						logs.Warning("delete sql failed, %s", err.Error())
					}
				}
			case FILE_RENAME_NEW:
				{
					_, err := s.db.Exec(TABLE_INSERT_SQL,
						fileNotify.File.Name, fileNotify.File.IsDir,
						fileNotify.File.Path, fileNotify.File.Ext,
						fileNotify.File.Drive,
						fileNotify.File.ModTime.Format(time.RFC3339),
						fileNotify.File.Size)
					if err != nil {
						logs.Warning("insert sql failed, %s", err.Error())
					}
				}
			}
		}

		s.Unlock()
	}

	logs.Info("sql recvice notify task shutdown")
}

func (s *SQLiteDB) Close() {
	s.notify <- struct{}{}
	s.Wait()

	s.Lock()
	defer s.Unlock()

	err := s.db.Close()
	if err != nil {
		logs.Warning("sql close faileld, %s", err.Error())
	}

	close(s.notify)
	s.notify = nil

	logs.Info("sql closed")
}

func (s *SQLiteDB) Notify() chan interface{} {
	return s.notify
}

func (s *SQLiteDB) Write(file FileInfo) {
	s.Lock()
	defer s.Unlock()

	_, err := s.db.Exec(TABLE_INSERT_SQL,
		file.Name, file.IsDir,
		file.Path, file.Ext,
		file.Drive,
		file.ModTime.Format(time.RFC3339),
		file.Size)
	if err != nil {
		logs.Warning("insert sql %v failed, %s", file, err.Error())
	}
}

func (s *SQLiteDB) Count() (int, error) {
	var rowCount int
	err := s.db.QueryRow("SELECT COUNT(*) AS row_count FROM file_info").Scan(&rowCount)
	if err != nil {
		logs.Error("query row count failed, %s", err.Error())
		return -1, err
	}
	return rowCount, nil
}

func (s *SQLiteDB) Query(keyword string, limit int) ([]FileInfo, error) {
	s.RLock()
	defer s.RUnlock()

	var rows *sql.Rows
	var err error

	if IsGlobChar(keyword) {
		rows, err = s.db.Query(TABLE_QUERY_GLOB_SQL, keyword, limit)
	} else {
		rows, err = s.db.Query(TABLE_QUERY_LIKE_SQL, "%"+keyword+"%", limit)
	}

	if err != nil {
		logs.Warning("query sql failed, %s", err.Error())
		return nil, err
	}

	defer rows.Close()

	output := make([]FileInfo, 0)

	for rows.Next() {
		var name, path, ext, drive, modTimeStr string
		var isDir int
		var size int64

		err := rows.Scan(&name, &isDir, &path, &ext, &drive, &modTimeStr, &size)
		if err != nil {
			logs.Warning("find error during scan row: %v", err)
		} else {
			modTime, err := time.Parse(time.RFC3339, modTimeStr)
			if err != nil {
				logs.Warning("parse mod time %s failed, %s", modTimeStr, err.Error())
			}
			output = append(output, FileInfo{Name: name, IsDir: isDir, Path: path, Ext: ext, Drive: drive, ModTime: modTime, Size: size})
		}
	}
	if err := rows.Err(); err != nil {
		logs.Warning("find error during iteration: %v", err)
	}

	return output, nil
}

func IsGlobChar(keyword string) bool {
	for _, v := range keyword {
		if v == '*' || v == '?' || v == '[' || v == ']' {
			return true
		}
	}
	return false
}
