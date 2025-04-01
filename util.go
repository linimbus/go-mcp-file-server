package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/astaxie/beego/logs"
	"github.com/lxn/walk"
	"github.com/lxn/win"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"

	. "github.com/lxn/walk/declarative"
)

var APPLICATION_VERSION = "0.1.0"

func SaveToFile(name string, body []byte) error {
	return os.WriteFile(name, body, 0664)
}

func CapSignal(proc func()) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-signalChan
		proc()
		logs.Error("recv signcal %s, ready to exit", sig.String())
		os.Exit(-1)
	}()
}

func TimeStampGet(tm time.Time) string {
	return tm.Format("2006-01-02 15:04:05")
}

func ByteView(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%dB", size)
	} else if size < (1024 * 1024) {
		return fmt.Sprintf("%.1fKB", float64(size)/float64(1024))
	} else if size < (1024 * 1024 * 1024) {
		return fmt.Sprintf("%.1fMB", float64(size)/float64(1024*1024))
	} else if size < (1024 * 1024 * 1024 * 1024) {
		return fmt.Sprintf("%.1fGB", float64(size)/float64(1024*1024*1024))
	} else {
		return fmt.Sprintf("%.1fTB", float64(size)/float64(1024*1024*1024*1024))
	}
}

func DefaultFont() Font {
	return Font{Family: "Segoe UI", PointSize: 9, Bold: false}
}

func InterfaceGet(iface *net.Interface) ([]net.IP, error) {
	addrs, err := iface.Addrs()
	if err != nil {
		return nil, err
	}
	ips := make([]net.IP, 0)
	for _, v := range addrs {
		ipone, _, err := net.ParseCIDR(v.String())
		if err != nil {
			continue
		}
		if len(ipone) > 0 {
			ips = append(ips, ipone)
		}
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("interface not any address")
	}
	return ips, nil
}

func InterfaceOptions() []string {
	output := []string{"0.0.0.0", "::"}
	ifaces, err := net.Interfaces()
	if err != nil {
		logs.Error("interface query failed, %s", err.Error())
		return output
	}
	for _, v := range ifaces {
		if v.Flags&net.FlagUp == 0 {
			continue
		}
		address, err := InterfaceGet(&v)
		if err != nil {
			logs.Warning("interface get failed, %s", err.Error())
			continue
		}
		for _, addr := range address {
			output = append(output, addr.String())
		}
	}
	return output
}

func CopyClipboard() (string, error) {
	text, err := walk.Clipboard().Text()
	if err != nil {
		logs.Error(err.Error())
		return "", fmt.Errorf("can not find the any clipboard")
	}
	return text, nil
}

func PasteClipboard(input string) error {
	err := walk.Clipboard().SetText(input)
	if err != nil {
		logs.Error(err.Error())
	}
	return err
}

func RegistryStartupSet(appName string, appPath string) error {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("can not open registry: %v", err)
	}
	defer key.Close()

	err = key.SetStringValue(appName, appPath)
	if err != nil {
		return fmt.Errorf("can not set %s the windows registry: %v", appPath, err)
	}
	return nil
}

func RegistryStartupDel(appName string) error {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("can not open registry: %v", err)
	}
	defer key.Close()

	err = key.DeleteValue(appName)
	if err != nil {
		return fmt.Errorf("can not del %s the windows registry: %v", appName, err)
	}
	return nil
}

func GetPathAttributes(filePath string) (uint32, error) {
	path, err := windows.UTF16PtrFromString(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to convert file path to UTF-16: %v", err)
	}
	attrs, err := windows.GetFileAttributes(path)
	if err != nil {
		return 0, fmt.Errorf("failed to get file attributes: %v", err)
	}
	return attrs, nil
}

func GetDriveNames() []string {
	var driveNames []string
	for i := 'A'; i <= 'Z'; i++ {
		drive := fmt.Sprintf("%c:\\", i)
		if _, err := os.Stat(drive); err == nil {
			driveNames = append(driveNames, drive)
		}
	}
	return driveNames
}

func SelectFolder() (string, error) {
	dlgDir := new(walk.FileDialog)
	dlgDir.FilePath = ""
	dlgDir.Flags = win.OFN_EXPLORER
	dlgDir.Title = "Please select a folder as filter directory"

	exist, err := dlgDir.ShowBrowseFolder(mainWindow)
	if err != nil {
		logs.Error(err.Error())
		return "", fmt.Errorf("can not find the any folder")
	}
	if exist {
		logs.Info("select %s as search directory", dlgDir.FilePath)
		return dlgDir.FilePath, nil
	}
	return "", nil
}
