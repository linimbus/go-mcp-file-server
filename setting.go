package main

import (
	"sort"
	"sync"

	"github.com/astaxie/beego/logs"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

type DriveItem struct {
	Name    string
	checked bool
}

type DriveTable struct {
	sync.RWMutex

	walk.TableModelBase
	walk.SorterBase
	sortColumn int
	sortOrder  walk.SortOrder

	items []*DriveItem
}

func (n *DriveTable) RowCount() int {
	return len(n.items)
}

func (n *DriveTable) Value(row, col int) interface{} {
	item := n.items[row]
	switch col {
	case 0:
		return item.Name
	}
	panic("unexpected col")
}

func (n *DriveTable) Checked(row int) bool {
	return n.items[row].checked
}

func (n *DriveTable) SetChecked(row int, checked bool) error {
	n.items[row].checked = checked
	return nil
}

func (n *DriveTable) ItemRemove() {
	n.Lock()
	defer n.Unlock()

	item := make([]*DriveItem, 0)

	for _, v := range n.items {
		if v.checked {
			continue
		}
		item = append(item, v)
	}

	n.items = item
	n.PublishRowsReset()
	n.Sort(n.sortColumn, n.sortOrder)
}

func (n *DriveTable) ItemsAdd(path string) {
	n.RLock()
	defer n.RUnlock()

	for _, v := range n.items {
		if v.Name == path {
			return
		}
	}
	n.items = append(n.items, &DriveItem{
		Name: path, checked: false,
	})

	n.PublishRowsReset()
	n.Sort(n.sortColumn, n.sortOrder)
}

func (m *DriveTable) ItemsChecked() []DriveConfig {
	m.Lock()
	defer m.Unlock()

	items := make([]DriveConfig, 0)
	for _, v := range m.items {
		items = append(items, DriveConfig{Name: v.Name, Enable: v.checked})
	}
	return items
}

func (m *DriveTable) ItemsAll() []string {
	m.Lock()
	defer m.Unlock()

	items := make([]string, 0)
	for _, v := range m.items {
		items = append(items, v.Name)
	}
	return items
}

func (m *DriveTable) Sort(col int, order walk.SortOrder) error {
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
		}
		panic("unreachable")
	})
	return m.SorterBase.Sort(col, order)
}

func SearchSettingDialog(from walk.Form) {
	var dlg *walk.Dialog
	var acceptPB, cancelPB *walk.PushButton

	var ignoreFolderCB, ignoreSystem, monitorCB *walk.CheckBox
	var driveTableView, filterListView *walk.TableView

	driveTable := &DriveTable{
		items: make([]*DriveItem, 0),
	}

	filterListTable := &DriveTable{
		items: make([]*DriveItem, 0),
	}

	config := ConfigGet()

	driveLists := GetDriveNames()
	for _, v := range driveLists {
		driveTable.items = append(driveTable.items, &DriveItem{
			Name:    v,
			checked: ConfigDriveExist(v),
		})
	}

	for _, v := range config.FilterFolder {
		filterListTable.items = append(filterListTable.items, &DriveItem{
			Name:    v,
			checked: false,
		})
	}

	_, err := Dialog{
		AssignTo:      &dlg,
		Title:         "Search Setting",
		MinSize:       Size{Width: 350, Height: 400},
		Icon:          ICON_Setting,
		Font:          DefaultFont(),
		DefaultButton: &acceptPB,
		CancelButton:  &cancelPB,
		Layout:        VBox{},
		Children: []Widget{

			Composite{
				Layout: Grid{Columns: 2, MarginsZero: true},
				Children: []Widget{
					Label{
						Text: "Search Drive List",
					},
					HSpacer{},

					TableView{
						AssignTo:         &driveTableView,
						AlternatingRowBG: false,
						ColumnsOrderable: true,
						CheckBoxes:       true,
						Columns: []TableViewColumn{
							{Title: "#", Width: 200},
						},
						Model: driveTable,
					},
					Composite{
						Alignment: AlignHNearVCenter,
						Layout:    VBox{MarginsZero: true},
						Children: []Widget{
							CheckBox{
								Alignment:          AlignHNearVCenter,
								AssignTo:           &ignoreFolderCB,
								Text:               "Ignore Hide Folder",
								Checked:            config.FilterHide,
								RightToLeftReading: true,
								OnCheckedChanged: func() {
									config.FilterHide = ignoreFolderCB.Checked()
								},
							},
							CheckBox{
								Alignment:          AlignHNearVCenter,
								AssignTo:           &ignoreSystem,
								Text:               "Ignore System Folder",
								Checked:            config.FilterHide,
								RightToLeftReading: true,
								OnCheckedChanged: func() {
									config.FilterSystem = ignoreSystem.Checked()
								},
							},
							CheckBox{
								Alignment:          AlignHNearVCenter,
								AssignTo:           &monitorCB,
								Text:               "Monitor File Change",
								Checked:            config.FileNotify,
								RightToLeftReading: true,
								OnCheckedChanged: func() {
									config.FileNotify = monitorCB.Checked()
								},
							},
						},
					},

					Label{
						Text: "Ignore Folder List",
					},
					HSpacer{},

					TableView{
						AssignTo:         &filterListView,
						AlternatingRowBG: false,
						ColumnsOrderable: true,
						CheckBoxes:       true,
						Columns: []TableViewColumn{
							{Title: "#", Width: 200},
						},
						Model: filterListTable,
					},
					Composite{
						Alignment: AlignHNearVCenter,
						Layout:    VBox{MarginsZero: true},
						Children: []Widget{
							PushButton{
								Text: "Add",
								OnClicked: func() {
									path, err := SelectFolder()
									if err != nil {
										ErrorBoxAction(dlg, "Select folder failed, "+err.Error())
										return
									}
									if path == "" {
										return
									}
									filterListTable.ItemsAdd(path)
								},
							},
							PushButton{
								Text: "Remove",
								OnClicked: func() {
									filterListTable.ItemRemove()
								},
							},
						},
					},
				},
			},
			VSpacer{},
			Composite{
				Layout: HBox{MarginsZero: true},
				Children: []Widget{

					PushButton{
						AssignTo: &acceptPB,
						Text:     "Accept",
						OnClicked: func() {
							config.SearchDrives = driveTable.ItemsChecked()
							if len(config.SearchDrives) == 0 {
								ErrorBoxAction(dlg, "Drive list is empty")
								return
							}
							config.FilterFolder = filterListTable.ItemsAll()
							if err := ConfigSet(config); err != nil {
								ErrorBoxAction(dlg, "Save config failed, "+err.Error())
								return
							}
							ServerRestart(config)
							dlg.Accept()
							logs.Info("SearchSettingDialog accept")
						},
					},
					PushButton{
						AssignTo: &cancelPB,
						Text:     "Cancel",
						OnClicked: func() {
							dlg.Cancel()
							logs.Info("SearchSettingDialog cancel")
						},
					},
				},
			},
		},
	}.Run(from)

	if err != nil {
		logs.Error("SearchSettingDialog: %s", err.Error())
	}
}

func McpServerConfigDialog(from walk.Form) {
	var dlg *walk.Dialog
	var acceptPB, cancelPB *walk.PushButton
	var listenBox *walk.ComboBox
	var portNum *walk.NumberEdit
	var enableCB *walk.CheckBox

	interfaces := InterfaceOptions()
	config := ConfigGet()

	InterfaceGet := func() int {
		for i, addr := range interfaces {
			if addr == config.McpListen {
				return i
			}
		}
		return 0
	}

	_, err := Dialog{
		AssignTo:      &dlg,
		Title:         "MCP Server Setting",
		MinSize:       Size{Width: 400, Height: 200},
		Size:          Size{Width: 400, Height: 200},
		Icon:          ICON_Setting,
		Font:          DefaultFont(),
		DefaultButton: &acceptPB,
		CancelButton:  &cancelPB,
		Layout:        VBox{},
		Children: []Widget{
			Composite{
				Layout: Grid{Columns: 2, Spacing: 10},
				Children: []Widget{
					Label{
						Text: "Listen Address",
					},
					ComboBox{
						AssignTo:     &listenBox,
						CurrentIndex: InterfaceGet(),
						Model:        interfaces,
						OnCurrentIndexChanged: func() {
							config.McpListen = listenBox.Text()
						},
						OnBoundsChanged: func() {
							listenBox.SetCurrentIndex(InterfaceGet())
						},
					},
					Label{
						Text: "Listen Port",
					},
					NumberEdit{
						AssignTo:    &portNum,
						Value:       float64(config.McpPort),
						ToolTipText: "0~65535",
						MaxValue:    65535,
						MinValue:    0,
						OnValueChanged: func() {
							config.McpPort = int(portNum.Value())
						},
					},
					HSpacer{},
					CheckBox{
						AssignTo: &enableCB,
						Text:     "Auto Enable",
						Checked:  config.McpEnable,
						OnCheckedChanged: func() {
							config.McpEnable = enableCB.Checked()
						},
					},
				},
			},
			VSpacer{},
			Composite{
				Layout: HBox{MarginsZero: true},
				Children: []Widget{

					PushButton{
						AssignTo: &acceptPB,
						Text:     "Accept",
						OnClicked: func() {
							if err := ConfigSet(config); err != nil {
								ErrorBoxAction(dlg, "Save config failed, "+err.Error())
								return
							}
							ServerRestart(config)
							dlg.Accept()
							logs.Info("McpServerConfigDialog accept")
						},
					},
					PushButton{
						AssignTo: &cancelPB,
						Text:     "Cancel",
						OnClicked: func() {
							dlg.Cancel()
							logs.Info("McpServerConfigDialog cancel")
						},
					},
				},
			},
		},
	}.Run(from)

	if err != nil {
		logs.Error("McpServerConfigDialog: %s", err.Error())
	}
}
