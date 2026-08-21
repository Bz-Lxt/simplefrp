package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"

	"simplefrp/internal/config"
	"simplefrp/internal/logger"
	"simplefrp/internal/server"
)

func main() {
	cfg, err := config.LoadServer()
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
	log := logger.New(cfg.LogLevel, os.Stdout)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := server.New(cfg, log).Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		log.Error("server exit", "err", err)
		os.Exit(1)
	}
}
