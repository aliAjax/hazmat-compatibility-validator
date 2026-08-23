package main

import (
	"context"
	"flag"
	platform "github.com/enterprise-labs/hazmat-compatibility-validator/internal/platform/application"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	configPath := flag.String("config", "configs/config.yaml", "configuration path")
	flag.Parse()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	config, err := platform.Load(*configPath)
	if err != nil {
		logger.Error("load config", "error", err)
		os.Exit(1)
	}
	runtime, err := platform.NewRuntime(config, logger)
	if err != nil {
		logger.Error("initialize runtime", "error", err)
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := runtime.Run(ctx); err != nil {
		logger.Error("runtime stopped", "error", err)
		os.Exit(1)
	}
}
