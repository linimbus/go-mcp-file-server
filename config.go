package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/astaxie/beego/logs"
)

type Config struct {
	McpListen string // mcp server listen address
	McpPort   int    // mcp server listen port
	McpEnable bool   // mcp server enable

	DriveName  []string // drive name list
	FilterName []string // filter name list
	FilterHide bool     // filter hide
	FileNotify bool     // sub filesystem notify

	AutoHide    bool // auto hide
	AutoStartup bool // auto startup
}

var configCache = Config{
	McpListen: "0.0.0.0",
	McpPort:   8888,
	McpEnable: true,

	DriveName:  []string{"C:\\", "D:\\", "E:\\"},
	FilterName: []string{"C:\\Windows", "C:\\Program Files", "C:\\Program Files (x86)", "C:\\ProgramData"},
	FilterHide: true,

	AutoHide:    false,
	AutoStartup: false,
}

var configFilePath string

func configSyncToFile() error {
	value, err := json.MarshalIndent(configCache, "\t", " ")
	if err != nil {
		logs.Error("json marshal config fail, %s", err.Error())
		return err
	}
	return os.WriteFile(configFilePath, value, 0664)
}

func ConfigDriveExist(name string) bool {
	for _, v := range configCache.DriveName {
		if v == name {
			return true
		}
	}
	return false
}

func ConfigGet() Config {
	return configCache
}

func ConfigSet(config Config) error {
	configCache = config
	err := configSyncToFile()
	if err != nil {
		logs.Error("sync config to file fail, %s", err.Error())
		return err
	}
	return nil
}

func AutoStartupSave(value bool) error {
	if value {
		execPath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("call executable failed, %s", err.Error())
		}
		err = RegistryStartupSet(APPLICATION_NAME, execPath)
		if err != nil {
			return err
		}
	} else {
		err := RegistryStartupDel(APPLICATION_NAME)
		if err != nil {
			return err
		}
	}
	configCache.AutoStartup = value
	return configSyncToFile()
}

func AUtoHideSave(value bool) error {
	configCache.AutoHide = value
	return configSyncToFile()
}

func ConfigInit() {
	var err error
	var value []byte

	configFilePath = filepath.Join(ConfigDirGet(), "config.json")

	defer func() {
		if err != nil {
			err = configSyncToFile()
			if err != nil {
				logs.Error("config sync to file fail, %s", err.Error())
			}
		}
	}()

	_, err = os.Stat(configFilePath)
	if err != nil {
		logs.Info("config file not exist, create a new one")
		return
	}

	value, err = os.ReadFile(configFilePath)
	if err != nil {
		logs.Error("read config file from app data dir fail, %s", err.Error())
		return
	}

	err = json.Unmarshal(value, &configCache)
	if err != nil {
		logs.Error("json unmarshal config fail, %s", err.Error())
		return
	}
}
