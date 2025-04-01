package main

import (
	"os"
	"path/filepath"
	"time"

	"github.com/astaxie/beego/logs"
)

var forceStatus bool // force status

func init() {
	go forceScanTask()
}

func ForceScanSet(status bool) {
	forceStatus = status
}

func ForceScanGet() bool {
	return forceStatus
}

func driveScan(s *SQLiteDB, cfg Config, drive string) error {
	startup := time.Now()

	err := filepath.Walk(drive, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			if cfg.CheckFolder(path) {
				logs.Info("skip folder %s", path)
				return filepath.SkipDir
			}
			s.Write(FileInfo{
				Name:    filepath.Base(path),
				IsDir:   1,
				Path:    path,
				Ext:     "",
				Drive:   drive,
				ModTime: info.ModTime(),
				Size:    0,
			})
		} else {
			s.Write(FileInfo{
				Name:    filepath.Base(path),
				IsDir:   0,
				Path:    path,
				Ext:     filepath.Ext(path),
				Drive:   drive,
				ModTime: info.ModTime(),
				Size:    info.Size(),
			})
		}

		temp := time.Now()

		if temp.Sub(startup).Milliseconds() > 500 {
			WorkingUpdate(path)
			startup = temp
		}

		return nil
	})
	return err
}

func driveFullScan(s *SQLiteDB, cfg Config) {
	for _, drive := range cfg.SearchDrives {
		if !drive.Enable {
			continue
		}
		logs.Info("full drive scan %s start", drive.Name)
		err := driveScan(s, cfg, drive.Name)
		if err != nil {
			logs.Error("full drive scan %s error: %s", drive.Name, err.Error())
		} else {
			logs.Info("full drive scan %s end", drive.Name)
		}
	}
}

func forceScanTask() {
	for {
		time.Sleep(time.Second)

		if forceStatus && srv != nil {
			srv.sql.Reset()
			logs.Info("force scan start")
			driveFullScan(srv.sql, configCache)
			logs.Info("force scan end")
			WorkingUpdate("")
			forceStatus = false
			forceScan.SetEnabled(true)
		}
	}
}
