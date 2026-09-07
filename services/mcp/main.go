package main

import (
	"context"
	"log"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const version = "0.7.0"

func main() {
	api, err := newAPIClient(os.Getenv("DEVRELOS_API_URL"), os.Getenv("DEVRELOS_API_TOKEN"))
	if err != nil {
		log.Fatalf("MCP configuration: %v", err)
	}

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "devrelos",
		Version: version,
	}, nil)
	registerTools(server, api)

	// Keep protocol traffic exclusively on stdout. The standard logger writes to stderr.
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatalf("DevRelOS MCP server: %v", err)
	}
}
