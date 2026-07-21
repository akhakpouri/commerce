package managers

import (
	"commerce/internal/shared/aws"
	"commerce/internal/shared/configs"
	"context"
	"log"
	"time"

	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

// Not wired into RelayManager: the relay is producer-only per ADR-018.
// This is scaffolding for a future, separate consumer app.
type ConsumerManagerI interface {
	CreateConsumer(config configs.ConsumerConfig, queueUrl string) *aws.Consumer
}

type ConsumerManager struct {
	client *sqs.Client
}

// GetConsumer implements [ConsumerManagerI].
func (c *ConsumerManager) CreateConsumer(config configs.ConsumerConfig,
	queueUrl string) *aws.Consumer {
	return aws.NewConsumer(c.client, config, messageHandler)
}

func messageHandler(ctx context.Context, msg *aws.Message) error {
	slog.Info("Processing message: %s, type: %s", msg.Id, msg.Type)

	// Simulate processing based on message type
	switch msg.Type {
	case "order.placed":
		return processOrder(ctx, msg)
	case "user.registered":
		return processUserRegistration(ctx, msg)
	default:
		log.Printf("Unknown message type: %s", msg.Type)
		return nil // Acknowledge unknown messages to prevent queue buildup
	}
}

func processOrder(ctx context.Context, msg *aws.Message) error {
	// Simulate order processing
	time.Sleep(100 * time.Millisecond)
	slog.Info("Processed order", "payload", msg.Payload)
	return nil
}

func processUserRegistration(ctx context.Context, msg *aws.Message) error {
	// Simulate user registration processing
	time.Sleep(50 * time.Millisecond)
	slog.Info("Processed user registration", "payload", msg.Payload)
	return nil
}

// healthHandler returns queue health status
func healthHandler(monitor *aws.QueueMonitor, queueURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if err := monitor.HealthCheck(ctx, queueURL); err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}

		err := json.NewEncoder(w).Encode(map[string]string{
			"status": "healthy",
		})
		if err != nil {
			panic(err)
		}
	}
}

func NewConsumerManager(client *sqs.Client) ConsumerManagerI {
	return &ConsumerManager{
		client: client,
	}
}
