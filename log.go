package main

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/astaxie/beego/logs"
)

type logconfig struct {
	Filename string `json:"filename"`
	Level    int    `json:"level"`
	MaxLines int    `json:"maxlines"`
	MaxSize  int    `json:"maxsize"`
	Daily    bool   `json:"daily"`
	MaxDays  int    `json:"maxdays"`
	Color    bool   `json:"color"`
}

var logCfg = logconfig{
	Filename: os.Args[0],
	Level:    logs.LevelInformational,
	Daily:    true,
	MaxSize:  10 * 1024 * 1024,
	MaxLines: 100 * 1024,
	MaxDays:  7,
	Color:    false,
}

func LogInit() {
	logCfg.Filename = filepath.Join(RunlogDirGet(), "runlog.log")
	value, err := json.Marshal(&logCfg)
	if err != nil {
		return
	}
	err = logs.SetLogger(logs.AdapterFile, string(value))
	if err != nil {
		return
	}
	logs.EnableFuncCallDepth(true)
	logs.SetLogFuncCallDepth(3)
}
