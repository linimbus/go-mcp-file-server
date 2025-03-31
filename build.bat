del *.exe~ *.exe
rsrc -manifest exe.manifest -ico main.ico
go-bindata -o icon_files.go main.ico main.png status_ok.ico status_bad.ico setting.ico
go build -buildvcs=false -ldflags="-H windowsgui -w -s" -o GoMcpFileServer.exe
