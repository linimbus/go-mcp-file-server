package main

import (
	"github.com/lxn/walk"
)

func ErrorBoxAction(form walk.Form, message string) {
	walk.MsgBox(form, "Error", message, walk.MsgBoxIconError|walk.MsgBoxOK)
}

func InfoBoxAction(form walk.Form, message string) {
	walk.MsgBox(form, "Info", message, walk.MsgBoxIconInformation|walk.MsgBoxOK)
}

func ConfirmBoxAction(form walk.Form, message string) {
	walk.MsgBox(form, "Confirm", message, walk.MsgBoxIconQuestion|walk.MsgBoxOKCancel)
}
