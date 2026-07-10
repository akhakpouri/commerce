package aws

import (
	"context"
	"fmt"
	"strconv"

	aws_sdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

type QueueMonitor struct {
	client *sqs.Client
}

// NewQueueMonitor creates a new QueueMonitor
func NewQueueMonitor(client *sqs.Client) *QueueMonitor {
	return &QueueMonitor{client: client}
}

// GetQueueStats retrieves current queue statistics
func (m *QueueMonitor) GetQueueStats(ctx context.Context, queueURL string) (*QueueStats, error) {
	input := &sqs.GetQueueAttributesInput{
		QueueUrl: aws_sdk.String(queueURL),
		AttributeNames: []types.QueueAttributeName{
			types.QueueAttributeNameApproximateNumberOfMessages,
			types.QueueAttributeNameApproximateNumberOfMessagesNotVisible,
			types.QueueAttributeNameApproximateNumberOfMessagesDelayed,
		},
	}

	result, err := m.client.GetQueueAttributes(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("getting queue attributes: %w", err)
	}

	stats := &QueueStats{}

	if val, ok := result.Attributes["ApproximateNumberOfMessages"]; ok {
		stats.ApproximateMessages, _ = strconv.ParseInt(val, 10, 64)
	}

	if val, ok := result.Attributes["ApproximateNumberOfMessagesNotVisible"]; ok {
		stats.ApproximateMessagesNotVisible, _ = strconv.ParseInt(val, 10, 64)
	}

	if val, ok := result.Attributes["ApproximateNumberOfMessagesDelayed"]; ok {
		stats.ApproximateMessagesDelayed, _ = strconv.ParseInt(val, 10, 64)
	}

	return stats, nil
}

// HealthCheck verifies the queue is accessible and operational
func (m *QueueMonitor) HealthCheck(ctx context.Context, queueURL string) error {
	_, err := m.GetQueueStats(ctx, queueURL)
	if err != nil {
		return fmt.Errorf("queue health check failed: %w", err)
	}
	return nil
}
