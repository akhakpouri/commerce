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

	"commerce/internal/shared/aws"
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

	sqsClient, err := aws.NewSqsClient(ctx, &config.Aws)
	if err != nil {
		slog.Error("failed to create SQS client", "error", err)
		panic("Failed to create SQS client")
	}
	daemon := daemon.NewDaemon(db, sqsClient, 5*time.Second)

	go func() {
		if err := daemon.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			slog.ErrorContext(ctx, "daemon exited unexpectedly", "error", err)
		}
	}()

	<-ctx.Done()
	slog.Info("Shutting down cleanly.")
	time.Sleep(1 * time.Second)
}
