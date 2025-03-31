package main

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type MCPServer struct {
	server *server.MCPServer
}

func NewMCPServer() *MCPServer {
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

		limit := request.Params.Arguments["limit"].(int)
		if limit <= 0 {
			limit = 100
		}

		fileName := request.Params.Arguments["filename"].(string)
		if len(fileName) == 0 {
			return nil, fmt.Errorf("filename is empty")
		}

		// result, err := HandleExec(request.Params.Arguments["filename"].(string), StatementTypeDelete)
		// if err != nil {
		// 	return nil, err
		// }

		return mcp.NewToolResultText(""), nil
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
	}
}

func (s *MCPServer) ServeSSE(addr string) *server.SSEServer {
	return server.NewSSEServer(s.server,
		server.WithBaseURL(fmt.Sprintf("http://%s", addr)),
	)
}
