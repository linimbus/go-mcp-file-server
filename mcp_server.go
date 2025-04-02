package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/astaxie/beego/logs"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type MCPServer struct {
	sync.WaitGroup

	server     *server.MCPServer
	sse        *server.SSEServer
	httpserver *http.Server
	sql        *SQLiteDB
}

func ResultToCSV(files []FileInfo) (string, error) {
	var csvBuf strings.Builder
	writer := csv.NewWriter(&csvBuf)

	if err := writer.Write(files[0].ToHeader()); err != nil {
		return "", fmt.Errorf("failed to write headers: %v", err)
	}

	for _, item := range files {
		if err := writer.Write(item.ToList()); err != nil {
			return "", fmt.Errorf("failed to write row: %v", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", fmt.Errorf("error flushing CSV writer: %v", err)
	}

	return csvBuf.String(), nil
}

func NewMCPServer(s *SQLiteDB) *MCPServer {
	mcpServer := server.NewMCPServer(
		APPLICATION_NAME,
		APPLICATION_VERSION,
	)

	queryTool := mcp.NewTool(
		"file_query",
		mcp.WithDescription("Execute a file search operation. "+
			" Make sure you have a filename or keyword before executing the query. "+
			" Make sure you have knowledge of SQLite3 query rules, "+
			" please use the `LIKE` or `GLOB` query rules."),
		mcp.WithString("filename",
			mcp.Required(),
			mcp.Description("The filename as a keyword to query"),
		),
		mcp.WithNumber("limit",
			mcp.DefaultNumber(100),
			mcp.Description("The maximum number of results to return"),
		),
	)

	openTool := mcp.NewTool(
		"file_open",
		mcp.WithDescription("Execute a file open operation."+
			" Make sure you have a filename or keyword before executing the open."),
		mcp.WithString("filename",
			mcp.Required(),
			mcp.Description("The filename as a keyword to open"),
		),
	)

	mcpServer.AddTool(queryTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		defer func() {
			if err := recover(); err != nil {
				logs.Error("serve http panic: %v", err)
			}
		}()

		logs.Info("mcp server start query tools")

		limit, ok := request.Params.Arguments["limit"].(float64)
		if !ok || limit <= 0.0 {
			limit = 100.0
		}

		filename, ok := request.Params.Arguments["filename"].(string)
		if !ok || len(filename) == 0 {
			return nil, fmt.Errorf("filename is empty")
		}

		logs.Info("mcp server start query filename: %s, limit: %d", filename, int(limit))

		fileInfos, err := s.Query(filename, int(limit))
		if err != nil {
			logs.Error("mcp server query failed, %s", err.Error())

			return nil, fmt.Errorf("sql query failed, %s", err.Error())
		}

		if len(fileInfos) == 0 {
			logs.Error("mcp server no files found")
			return nil, fmt.Errorf("no files found")
		}

		logs.Info("mcp server start query filename: %s, number: %d", filename, len(fileInfos))

		csvText, err := ResultToCSV(fileInfos)
		if err != nil {
			return nil, fmt.Errorf("covert to csv failed, %s", err.Error())
		}

		return mcp.NewToolResultText(csvText), nil
	})

	mcpServer.AddTool(openTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		defer func() {
			if err := recover(); err != nil {
				logs.Error("serve http panic: %v", err)
			}
		}()

		err := OpenBrowserWeb(request.Params.Arguments["filename"].(string))
		if err != nil {
			return nil, err
		}
		return mcp.NewToolResultText("ok"), nil
	})

	return &MCPServer{
		server: mcpServer,
		sql:    s,
	}
}

func (s *MCPServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.sse.ServeHTTP(w, r)
}

func (s *MCPServer) Startup(addr string, port int) error {
	var address string
	if strings.Contains(addr, ":") {
		address = fmt.Sprintf("[%s]:%d", addr, port)
	} else {
		address = fmt.Sprintf("%s:%d", addr, port)
	}

	listen, err := net.Listen("tcp", address)
	if err != nil {
		logs.Error("http file server listen %s address fail", address)
		return err
	}

	s.httpserver = &http.Server{
		Addr:    address,
		Handler: s,
	}

	s.sse = server.NewSSEServer(s.server, server.WithHTTPServer(s.httpserver))

	logs.Info("http file server listening on %s", address)

	s.Add(1)

	go func() {
		defer s.Done()
		err = s.httpserver.Serve(listen)
		if err != nil {
			logs.Warning("httpserver serv failed, %s", err.Error())
		}
	}()

	return nil
}

func (s *MCPServer) Shutdown() {
	logs.Info("mcp server ready to shutdown")
	context, cencel := context.WithTimeout(context.Background(), 5*time.Second)
	err := s.sse.Shutdown(context)
	cencel()
	if err != nil {
		logs.Warning("mcp server shutdown failed, %s", err.Error())
	}
	s.Wait()
}
