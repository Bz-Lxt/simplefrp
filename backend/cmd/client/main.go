package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"simplefrp/internal/client"
	"simplefrp/internal/config"
	"simplefrp/internal/logger"
)

func main() {
	cfg, err := config.LoadClient()
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
	log := logger.New(cfg.LogLevel, os.Stdout)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	agent := client.New(cfg, log)
	if err := agent.Run(ctx); err != nil && err != context.Canceled {
		log.Error("client exit", "err", err)
		os.Exit(1)
	}
}
