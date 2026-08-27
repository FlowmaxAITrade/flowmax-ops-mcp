package main

import (
	"fmt"
	"log"
	"os"

	"github.com/FlowmaxAITrade/flowmax-ops-mcp/internal/config"
	mcpserver "github.com/FlowmaxAITrade/flowmax-ops-mcp/internal/mcp"
	"github.com/FlowmaxAITrade/flowmax-ops-mcp/internal/version"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v":
			fmt.Println(version.String())
			return
		}
	}

	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("config error: %v", err)
	}

	s := mcpserver.NewServer(cfg.OpsBEBaseURL, cfg.OpsAPIKey)
	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
