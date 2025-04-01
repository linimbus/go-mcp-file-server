package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/astaxie/beego/logs"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type MCPServer struct {
	sync.WaitGroup

	server *server.MCPServer
	sse    *server.SSEServer
	sql    *SQLiteDB
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
		mcp.WithDescription("Execute a file search operation. Make sure you have a filename or keyword before executing the query."),
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
		mcp.WithDescription("Execute a file open operation. Make sure you have a filename or keyword before executing the open."),
		mcp.WithString("filename",
			mcp.Required(),
			mcp.Description("The filename as a keyword to open"),
		),
	)

	mcpServer.AddTool(queryTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {

		logs.Info("mcp server start query tools")

		limit, ok := request.Params.Arguments["limit"].(int)
		if !ok || limit <= 0 {
			limit = 100
		}

		filename, ok := request.Params.Arguments["filename"].(string)
		if !ok || len(filename) == 0 {
			return nil, fmt.Errorf("filename is empty")
		}

		logs.Info("mcp server start query filename: %s, limit: %d", filename, limit)

		fileInfos, err := s.Query(filename, limit)
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

func (s *MCPServer) Startup(addr string, port int) error {
	var address string
	if strings.Contains(addr, ":") {
		address = fmt.Sprintf("[%s]:%d", addr, port)
	} else {
		address = fmt.Sprintf("%s:%d", addr, port)
	}

	s.sse = server.NewSSEServer(s.server)

	go func() {
		defer s.Done()

		err := s.sse.Start(address)
		if err != nil {
			logs.Error("mcp server attach listen fail, %s", err.Error())
		}

		logs.Info("mcp server shutdown")
	}()

	return nil
}

func (s *MCPServer) Shutdown() {
	logs.Info("mcp server ready to shutdown")
	context, cencel := context.WithTimeout(context.Background(), 3*time.Second)
	err := s.sse.Shutdown(context)
	cencel()
	if err != nil {
		logs.Error("mcp server shutdown failed, %s", err.Error())
	}
	s.Wait()
}
