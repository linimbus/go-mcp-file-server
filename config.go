package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/astaxie/beego/logs"
	"golang.org/x/sys/windows"
)

type DriveConfig struct {
	Name   string `json:"name"`   // drive name
	Enable bool   `json:"enable"` // drive enable
}

type Config struct {
	McpListen string `json:"mcp_listen"` // mcp server listen address
	McpPort   int    `json:"mcp_port"`   // mcp server listen port
	McpEnable bool   `json:"map_enable"` // mcp server enable

	SearchDrives []DriveConfig `json:"search_drives"`        // drive name list
	FilterRegexp []string      `json:"filter_regexp"`        // filter regex list
	FilterFolder []string      `json:"filter_folder"`        // filter folder list
	FilterHide   bool          `json:"filter_hide_folder"`   // filter hide
	FilterSystem bool          `json:"filter_system_folder"` // filter system folder
	FileNotify   bool          `json:"file_notify_enable"`   // filesystem change event notify

	AutoHide    bool `json:"auto_hide_windows"`   // auto hide
	AutoStartup bool `json:"auto_startup_system"` // auto startup

	CacheLength uint32 `json:"cache_length"`
}

func (c *Config) CheckFolder(path string) bool {
	if len(path) < 4 { // ignore the root path like "C:\"
		return false
	}

	if strings.Contains(path, "$Recycle.Bin") {
		return true
	}

	if strings.Contains(path, DEFAULT_HOME) {
		return true
	}

	for _, v := range c.FilterFolder {
		if strings.Contains(path, v) {
			return true
		}
	}

	if c.FilterHide || c.FilterSystem {
		attr, err := GetPathAttributes(path)
		if err != nil {
			// logs.Warning("get path attributes failed, %s", err.Error()) // too much
			return false
		}

		if c.FilterHide && attr&windows.FILE_ATTRIBUTE_HIDDEN != 0 {
			return true
		}

		if c.FilterSystem && attr&windows.FILE_ATTRIBUTE_SYSTEM != 0 {
			return true
		}
	}

	if c.FilterHide && strings.Contains(path, "\\.") {
		// logs.Info("skip hidden folder %s", path) // too much
		return true
	}

	return false
}

var configCache = Config{
	McpListen:    "0.0.0.0",
	McpPort:      8888,
	McpEnable:    true,
	SearchDrives: []DriveConfig{},
	FilterFolder: []string{"C:\\Windows", "C:\\Program Files", "C:\\Program Files (x86)", "C:\\ProgramData"},
	FilterRegexp: []string{},
	FilterHide:   true,
	FilterSystem: true,
	AutoHide:     false,
	AutoStartup:  false,
	CacheLength:  1024 * 1024,
}

func init() {
	drives := GetDriveNames()
	for _, drive := range drives {
		configCache.SearchDrives = append(configCache.SearchDrives, DriveConfig{
			Name:   drive,
			Enable: true,
		})
	}
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
	for _, v := range configCache.SearchDrives {
		if v.Name == name && v.Enable {
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
