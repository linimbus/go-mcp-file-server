package main

import (
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

var statusBar *walk.StatusBarItem

func StatusUpdate(status string) {
	if statusBar != nil {
		statusBar.SetText(status)
	}

}

func StatusBarInit() []StatusBarItem {
	return []StatusBarItem{
		{
			AssignTo: &statusBar,
			Icon:     ICON_Status_bad,
			Width:    300,
			OnClicked: func() {
				// PasteClipboard(statusBar.Text())
			},
		},
	}
}
