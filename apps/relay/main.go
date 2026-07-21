package main

import (
	"commerce/relay/configs"
	daemon "commerce/relay/worker"
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	slog.Info("Starting the relay.")
	config := configs.NewConfig()
	db, err := config.Database.Connect()
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		panic("Failed to connect to the database")
	}

	// Setup Graceful Shutdown Context
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	relay, err := daemon.NewDaemon(ctx, db, &config.Aws)
	if err != nil {
		slog.Error("failed to initialize relay", "error", err)
		panic("Failed to initialize the relay")
	}

	// Start blocks until ctx is canceled, so shutdown is synchronous - no
	// goroutine, no arbitrary sleep racing an in-flight publish.
	if err := relay.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
		slog.Error("relay stopped unexpectedly", "error", err)
		os.Exit(1)
	}

	slog.Info("Shutting down cleanly.")
}
