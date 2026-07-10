package aws

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sqs"

	aws_sdk "github.com/aws/aws-sdk-go-v2/aws"
)

type VisibilityExtender struct {
	client *sqs.Client
	url    string
}

func NewVisibilityExtender(client *sqs.Client, url string) *VisibilityExtender {
	return &VisibilityExtender{
		client: client,
		url:    url,
	}
}

func (v *VisibilityExtender) ExtendVisibility(
	ctx context.Context,
	receiptHandle string,
	timeout int32) error {
	input := &sqs.ChangeMessageVisibilityInput{
		QueueUrl:          aws_sdk.String(v.url),
		ReceiptHandle:     aws_sdk.String(receiptHandle),
		VisibilityTimeout: timeout,
	}

	_, err := v.client.ChangeMessageVisibility(ctx, input)
	if err != nil {
		slog.Info("exception occured when extending visibility timeout", "error", err)
		return fmt.Errorf("extending visibility timeout: %w", err)
	}

	return nil
}

// StartVisibilityHeartbeat continuously refreshes visibility for long tasks
// On each tick it sets the message's visibility timeout to visibilityTimeout
// seconds from that moment. Stops when the context is cancelled.
func (v *VisibilityExtender) StartVisibilityHeartbeat(
	ctx context.Context,
	receiptHandle string,
	interval time.Duration,
	visibilityTimeout int32,
) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := v.ExtendVisibility(ctx, receiptHandle, visibilityTimeout); err != nil {
				slog.Info("Failed to extend visibility", "error", err)
				return
			}
		}
	}
}
