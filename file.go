package main

import (
	"fmt"
	"os"
	"path/filepath"
)

var DEFAULT_HOME string
var APPLICATION_NAME = "GoMcpFileServer"

func RunlogDirGet() string {
	dir := fmt.Sprintf("%s\\runlog", DEFAULT_HOME)
	_, err := os.Stat(dir)
	if err != nil {
		os.MkdirAll(dir, 0644)
	}
	return dir
}

func ConfigDirGet() string {
	dir := fmt.Sprintf("%s\\config", DEFAULT_HOME)
	_, err := os.Stat(dir)
	if err != nil {
		os.MkdirAll(dir, 0644)
	}
	return dir
}

func appDataDir() string {
	datadir := os.Getenv("APPDATA")
	if datadir == "" {
		datadir = os.Getenv("CD")
	}
	if datadir == "" {
		datadir = ".\\"
	} else {
		datadir = filepath.Join(datadir, APPLICATION_NAME)
	}
	return datadir
}

func appDataDirInit() {
	dir := appDataDir()
	_, err := os.Stat(dir)
	if err != nil {
		os.MkdirAll(dir, 0644)
	}
	DEFAULT_HOME = dir
}

func FileInit() {
	appDataDirInit()
}
