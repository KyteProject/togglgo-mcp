package main

import (
	"context"
	"errors"
	"log/slog"
	"os"

	"github.com/kyteproject/togglgo-mcp/app"

	"github.com/mark3labs/mcp-go/server"
)

func main() {
	// Setup structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	apiToken := os.Getenv("TOGGL_API_TOKEN")
	if apiToken == "" {
		logger.Error("missing API token", slog.Any("error", app.ErrNoAPIToken))
		os.Exit(1)
	}

	togglClient := app.NewTogglClient(apiToken, app.WithLogger(logger))

	s := server.NewMCPServer("toggl-mcp", "1.0.0")

	if err := app.SetupTools(s, togglClient); err != nil {
		logger.Error("failed to setup tools", slog.Any("error", err))
		os.Exit(1)
	}

	logger.Info("starting Toggl MCP server")
	if err := server.ServeStdio(s); err != nil {
		if !errors.Is(err, context.Canceled) {
			logger.Error("server error", slog.Any("error", err))
			os.Exit(1)
		}
	}

	logger.Info("server stopped gracefully")
}
