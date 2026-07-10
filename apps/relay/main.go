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
	"time"
)

func main() {
	slog.Info("Starting the relay.")
	config := configs.NewConfig()
	db, err := config.Database.Connect()
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		panic("Failed to connect to the database")
	}
	daemon := daemon.NewDaemon(db, 5*time.Second)

	// Setup Graceful Shutdown Context
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := daemon.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			slog.ErrorContext(ctx, "daemon exited unexpectedly", "error", err)
		}
	}()

	<-ctx.Done()
	slog.Info("Shutting down cleanly.")
	time.Sleep(1 * time.Second)
}
