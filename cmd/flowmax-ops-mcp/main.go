package main

import (
	"log"

	"github.com/FlowmaxAITrade/flowmax-ops-mcp/internal/config"
	mcpserver "github.com/FlowmaxAITrade/flowmax-ops-mcp/internal/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("config error: %v", err)
	}

	s := mcpserver.NewServer(cfg.OpsBEBaseURL, cfg.OpsAPIKey)
	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
