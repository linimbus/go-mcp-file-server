package main

import (
	"github.com/lxn/walk"
)

func ErrorBoxAction(form walk.Form, message string) {
	walk.MsgBox(form, message, "Error", walk.MsgBoxIconError|walk.MsgBoxOK)
}

func InfoBoxAction(form walk.Form, message string) {
	walk.MsgBox(form, message, "Info", walk.MsgBoxIconInformation|walk.MsgBoxOK)
}

func ConfirmBoxAction(form walk.Form, message string) {
	walk.MsgBox(form, message, "Confirm", walk.MsgBoxIconQuestion|walk.MsgBoxOKCancel)
}
