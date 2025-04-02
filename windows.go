package main

import (
	"fmt"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/astaxie/beego/logs"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

var mainWindow *walk.MainWindow
var mainServer *Server

var WILDCARDS_HELP = `
The GLOB operator in SQLite uses wildcards for pattern matching.
Here are all the special symbols it supports and their functions:

Basic wildcards: 

* (asterisk)
	Matches zero or more of any character
	Example: '*.txt' matches all strings ending with .txt

? (question mark)
	Matches any single character
	Example: 'file?.txt' matches file1.txt, fileA.txt, etc.

Character set matching: 

[...] (character set)
	Matches any single character in the square brackets
	Example: '[abc]*' matches a string starting with a, b, or c

[^...] or [!...] (negated character set)
	Matches any single character not in the square brackets
	Example: '[^0-9]*' matches a string that does not start with a number

[a-z] (character range)
	Matches any single character in the specified range
	Example: '[A-Za-z]*' Matches a string that starts with a letter

Special considerations: 

Case sensitivity: GLOB is case sensitive by default.
'A*' matches "Apple" but not "apple"

`

func init() {
	go func() {
		for {
			if mainWindow != nil && mainWindow.Visible() {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		NotifyInit()
	}()
}

var autoStartupCheck, forceScan *walk.Action

func MenuBarInit() []MenuItem {
	return []MenuItem{
		Menu{
			Text: "Setting",
			Items: []MenuItem{
				Action{
					Text: "Index Setting",
					OnTriggered: func() {
						SearchSettingDialog(mainWindow)
					},
				},
				Action{
					AssignTo: &forceScan,
					Text:     "Index Rebuild",
					Enabled:  true,
					OnTriggered: func() {
						go func() {
							if mainServer != nil {
								forceScan.SetEnabled(false)
								mainServer.RebuidIndex()
								forceScan.SetEnabled(true)
							}
						}()
					},
				},
				Action{
					Text: "MCP Server Setting",
					OnTriggered: func() {
						McpServerConfigDialog(mainWindow)
					},
				},
				Action{
					Text:     "Auto Startup",
					AssignTo: &autoStartupCheck,
					Checked:  ConfigGet().AutoStartup,
					OnTriggered: func() {
						flag := !autoStartupCheck.Checked()
						err := AutoStartupSave(flag)
						if err != nil {
							ErrorBoxAction(mainWindow, err.Error())
						} else {
							autoStartupCheck.SetChecked(flag)
						}
					},
				},
				Separator{},
				Action{
					Text: "Exit",
					OnTriggered: func() {
						CloseWindows()
					},
				},
			},
		},
		Menu{
			Text: "View",
			Items: []MenuItem{
				Action{
					Text: "Mini Windows",
					OnTriggered: func() {
						NotifyAction()
					},
				},
			},
		},
		Menu{
			Text: "Help",
			Items: []MenuItem{
				Action{
					Text: "Wildcards",
					OnTriggered: func() {
						InfoAction(mainWindow, WILDCARDS_HELP)
					},
				},
				Action{
					Text: "Runlog",
					OnTriggered: func() {
						OpenBrowserWeb(RunlogDirGet())
					},
				},
				Separator{},
				Action{
					Text: "About",
					OnTriggered: func() {
						AboutAction()
					},
				},
			},
		},
	}
}

type ViewItem struct {
	Name    string
	Path    string
	Size    string
	ModTime string

	checked bool
}

type QueryTable struct {
	sync.RWMutex

	walk.TableModelBase
	walk.SorterBase
	sortColumn int
	sortOrder  walk.SortOrder

	items []*ViewItem
}

func (n *QueryTable) RowCount() int {
	return len(n.items)
}

func (n *QueryTable) Value(row, col int) interface{} {
	item := n.items[row]
	switch col {
	case 0:
		return item.Name
	case 1:
		return item.Path
	case 2:
		return item.Size
	case 3:
		return item.ModTime
	}
	panic("unexpected col")
}

func (n *QueryTable) Checked(row int) bool {
	return n.items[row].checked
}

func (n *QueryTable) SetChecked(row int, checked bool) error {
	n.items[row].checked = checked
	return nil
}

func (m *QueryTable) Sort(col int, order walk.SortOrder) error {
	m.sortColumn, m.sortOrder = col, order
	sort.SliceStable(m.items, func(i, j int) bool {
		a, b := m.items[i], m.items[j]
		c := func(ls bool) bool {
			if m.sortOrder == walk.SortAscending {
				return ls
			}
			return !ls
		}
		switch m.sortColumn {
		case 0:
			return c(a.Name < b.Name)
		case 1:
			return c(a.Path < b.Path)
		case 2:
			return c(a.Size < b.Size)
		case 3:
			return c(a.ModTime < b.ModTime)
		}
		panic("unreachable")
	})
	return m.SorterBase.Sort(col, order)
}

func (m *QueryTable) QuerySearch(keyword string) error {
	m.Lock()
	defer m.Unlock()

	if mainServer == nil {
		return fmt.Errorf("sqlite db init failed")
	}

	fileInfos, err := mainServer.sql.Query(keyword, 1024)
	if err != nil {
		return err
	}

	m.items = make([]*ViewItem, 0)
	for _, file := range fileInfos {
		m.items = append(m.items, &ViewItem{
			Name:    file.Name,
			Path:    file.Path,
			Size:    ByteView(file.Size),
			ModTime: file.ModTime.Format(time.DateTime),
		})
	}

	m.PublishRowsReset()
	m.Sort(m.sortColumn, m.sortOrder)

	return nil
}

func (m *QueryTable) OpenFile(index int) {
	m.RLock()
	defer m.RUnlock()

	if index < len(m.items) {
		OpenBrowserWeb(m.items[index].Path)
	}
}

var queryTableView *walk.TableView
var queryTableData *QueryTable
var searchText *walk.LineEdit

func init() {
	queryTableData = &QueryTable{
		items: make([]*ViewItem, 0),
	}
}

func ServerRestart(config Config) {
	var err error

	if mainServer != nil {
		mainServer.Shutdown()
	}

	mainServer, err = NewServer(config)
	if err != nil {
		StatusUpdate(err.Error())
	}
}

func MainWindows() {
	defer func() {
		if err := recover(); err != nil {
			logs.Error("recover error %v", err)
		}
	}()

	ServerRestart(ConfigGet())

	CapSignal(CloseWindows)

	cnt, err := MainWindow{
		Title:          APPLICATION_NAME + " " + APPLICATION_VERSION,
		Icon:           ICON_Main,
		AssignTo:       &mainWindow,
		MinSize:        Size{Width: 900, Height: 500},
		Size:           Size{Width: 900, Height: 500},
		Layout:         VBox{Margins: Margins{Top: 5, Bottom: 5, Left: 5, Right: 5}},
		Font:           DefaultFont(),
		MenuItems:      MenuBarInit(),
		StatusBarItems: StatusBarInit(),
		Children: []Widget{
			Composite{
				Layout: HBox{MarginsZero: true},
				Children: []Widget{
					LineEdit{
						AssignTo: &searchText,
						OnEditingFinished: func() {
							queryTableData.QuerySearch(searchText.Text())
						},
						OnTextChanged: func() {
							queryTableData.QuerySearch(searchText.Text())
						},
					},
				},
			},
			TableView{
				AssignTo:         &queryTableView,
				AlternatingRowBG: false,
				ColumnsOrderable: true,
				CheckBoxes:       false,
				Columns: []TableViewColumn{
					{Title: "Name", Width: 180},
					{Title: "FilePath", Width: 380},
					{Title: "Size", Width: 120},
					{Title: "ModTime", Width: 160},
				},
				Model: queryTableData,
				OnItemActivated: func() {
					index := queryTableView.CurrentIndex()
					queryTableData.OpenFile(index)
				},
			},
		},
	}.Run()

	if err != nil {
		logs.Error("main windows failed %s", err.Error())
	} else {
		logs.Info("main windows exit %d", cnt)
	}

	CloseWindows()
}

func CloseWindows() {
	if mainServer != nil {
		mainServer.Shutdown()
		mainServer = nil
	}
	if mainWindow != nil {
		mainWindow.Close()
		mainWindow = nil
	}
	NotifyExit()
	os.Exit(0)
}
