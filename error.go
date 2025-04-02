package main

import (
	"github.com/astaxie/beego/logs"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

func InfoAction(form walk.Form, message string) {
	var ok *walk.PushButton
	var info *walk.Dialog
	var err error

	_, err = Dialog{
		AssignTo:      &info,
		Title:         "Help Info",
		Icon:          walk.IconInformation(),
		MinSize:       Size{Width: 450, Height: 500},
		DefaultButton: &ok,
		Layout:        VBox{},
		Children: []Widget{
			Composite{
				Layout: HBox{},
				Children: []Widget{
					TextLabel{
						Text: message,
					},
				},
			},
			Composite{
				Layout: HBox{MarginsZero: true},
				Children: []Widget{
					HSpacer{},
					PushButton{
						Text:      "OK",
						OnClicked: func() { info.Cancel() },
					},
					HSpacer{},
				},
			},
		},
	}.Run(mainWindow)

	if err != nil {
		logs.Error("info action %s", err.Error())
	}
}

func ErrorBoxAction(form walk.Form, message string) {
	walk.MsgBox(form, "Error", message, walk.MsgBoxIconError|walk.MsgBoxOK)
}

func InfoBoxAction(form walk.Form, message string) {
	walk.MsgBox(form, "Info", message, walk.MsgBoxIconInformation|walk.MsgBoxOK)
}

func ConfirmBoxAction(form walk.Form, message string) {
	walk.MsgBox(form, "Confirm", message, walk.MsgBoxIconQuestion|walk.MsgBoxOKCancel)
}
