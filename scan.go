package main

import (
	"os"
	"path/filepath"
	"time"

	"github.com/astaxie/beego/logs"
)

func driveScan(s *SQLiteDB, cfg Config, drive string, shutdown *bool) error {
	startup := time.Now()

	err := filepath.Walk(drive, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if *shutdown {
			logs.Info("full drive scan cancel")
			return filepath.SkipAll
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

		if temp.Sub(startup).Milliseconds() > 200 {
			WorkingUpdate(path)
			startup = temp
		}

		return nil
	})
	return err
}

func DriveFullScan(s *SQLiteDB, cfg Config, shutdown *bool) {
	for _, drive := range cfg.SearchDrives {
		if !drive.Enable && !*shutdown {
			continue
		}

		logs.Info("full drive scan %s start", drive.Name)
		err := driveScan(s, cfg, drive.Name, shutdown)
		if err != nil {
			logs.Error("full drive scan %s error: %s", drive.Name, err.Error())
		} else {
			logs.Info("full drive scan %s end", drive.Name)
		}
	}
}
