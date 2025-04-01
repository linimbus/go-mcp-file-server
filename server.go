package main

import "github.com/astaxie/beego/logs"

type Server struct {
	sql  *SQLiteDB
	mcp  *MCPServer
	file *FileEvent
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

	return &Server{
		sql: sql, mcp: mcp, file: file,
	}, nil
}

func (s *Server) Shutdown() {
	s.file.Close()
	s.mcp.Shutdown()
	s.sql.Close()
}
