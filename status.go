package main

import (
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

var statusBar, workingBar *walk.StatusBarItem
var statusText string

func statusIcon(status string) *walk.Icon {
	if status != "" {
		return ICON_Status_bad
	}
	return ICON_Status_ok
}

func StatusUpdate(status string) {
	if statusBar != nil {
		statusBar.SetText(status)
		statusBar.SetIcon(statusIcon(status))
	}
	statusText = status
}

func WorkingUpdate(status string) {
	if workingBar != nil {
		workingBar.SetText(status)
	}
}

func StatusBarInit() []StatusBarItem {
	return []StatusBarItem{
		{
			AssignTo: &statusBar,
			Icon:     statusIcon(statusText),
			Width:    200,
			Text:     statusText,
		},
		{
			AssignTo: &workingBar,
			Width:    400,
			Text:     "",
		},
	}
}
