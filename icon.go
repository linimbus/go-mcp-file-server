package main

import (
	"path/filepath"

	"github.com/astaxie/beego/logs"
	"github.com/lxn/walk"
)

func IconLoadFromBox(filename string, size walk.Size) *walk.Icon {
	body, err := Asset(filename)
	if err != nil {
		logs.Error("asset file %s failed, %s", filename, err.Error())
		return walk.IconApplication()
	}
	filename = filepath.Join(ConfigDirGet(), filename)
	err = SaveToFile(filename, body)
	if err != nil {
		logs.Error("save to file %s failed, %s", filename, err.Error())
		return walk.IconApplication()
	}
	icon, err := walk.NewIconFromFileWithSize(filename, size)
	if err != nil {
		logs.Error("walk new icon from file %s failed, %s", filename, err.Error())
		return walk.IconApplication()
	}
	return icon
}

var ICON_Main *walk.Icon
var ICON_Status_ok *walk.Icon
var ICON_Status_bad *walk.Icon
var ICON_Setting *walk.Icon

var ICON_Max_Size = walk.Size{
	Width: 64, Height: 64,
}

var ICON_Min_Size = walk.Size{
	Width: 16, Height: 16,
}

func IconInit() {
	ICON_Main = IconLoadFromBox("main.ico", ICON_Max_Size)
	ICON_Status_ok = IconLoadFromBox("status_ok.ico", ICON_Min_Size)
	ICON_Status_bad = IconLoadFromBox("status_bad.ico", ICON_Min_Size)
	ICON_Setting = IconLoadFromBox("setting.ico", ICON_Max_Size)
}
