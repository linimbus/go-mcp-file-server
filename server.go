package main

import (
	"github.com/astaxie/beego/logs"
)

type Server struct {
	shutdown bool

	config Config
	sql    *SQLiteDB
	mcp    *MCPServer
	file   *FileEvent
}

func NewServer(config Config) (*Server, error) {
	sql, err := NewSQLiteDB()
	if err != nil {
		logs.Error("sqlite db init failed %s", err.Error())
		return nil, err
	}
	mcp := NewMCPServer(sql)

	err = mcp.Startup(config.McpListen, config.McpPort)
	if err != nil {
		logs.Error("mcp server startup failed, %s", err.Error())
		return nil, err
	}

	file, err := NewFileEvent(sql, config)
	if err != nil {
		logs.Error("file event startup failed, %s", err.Error())
	}

	logs.Info("server init success")

	return &Server{
		sql: sql, mcp: mcp, file: file,
		config: config,
	}, nil
}

func (s *Server) Shutdown() {
	s.shutdown = true
	s.file.Close()
	s.mcp.Shutdown()
	s.sql.Close()
	logs.Info("server done")
}

func (s *Server) RebuidIndex() {
	err := s.sql.Reset()
	if err != nil {
		logs.Error("sql index reset failed, %s", err.Error())
		return
	}
	logs.Info("force scan start")
	DriveFullScan(s.sql, s.config, &s.shutdown)
	logs.Info("force scan end")
	WorkingUpdate("")
}
