package aws

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	aws_sdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

// the purpose of this manager is to handle queue operation
type QueueManager struct {
	client *sqs.Client
}

func NewQueueManager(client *sqs.Client) *QueueManager {
	return &QueueManager{
		client: client,
	}
}

// purpose of this function is to create a new standard SQS queue with specific settings.
func (q *QueueManager) CreateStandardQueue(
	ctx context.Context,
	name string,
	timeout int,
	retention int,
	waitTime ...int) (string, error) {
	wt := calcWaitTime(waitTime...)
	input := &sqs.CreateQueueInput{
		Attributes: map[string]string{
			"VisibilityTimeout": strconv.Itoa(timeout),
			// Time in seconds to retain messages (default 4 days, max 14 days)
			"MessageRetentionPeriod":        strconv.Itoa(retention),
			"ReceiveMessageWaitTimeSeconds": strconv.Itoa(wt),
		},
	}

	result, err := q.client.CreateQueue(ctx, input)
	if err != nil {
		slog.Error("exception occured when creating standard queue", "error", err)
		return "", fmt.Errorf("exception occured when creating standard queue. %w", err)
	}
	return *result.QueueUrl, nil
}

func (q *QueueManager) CreateFIFOQueue(
	ctx context.Context,
	name string,
	contentDedup bool,
	timeout int,
	waitTime ...int) (string, error) {
	wt := calcWaitTime(waitTime...)
	attributes := map[string]string{
		"FifoQueue":                     "true",
		"VisibilityTimeout":             strconv.Itoa(timeout),
		"ReceiveMessageWaitTimeSeconds": strconv.Itoa(wt),
	}

	fifoName := name + ".fifo"

	// Content-based deduplication uses message body hash
	if contentDedup {
		attributes["ContentBasedDeduplication"] = "true"
	}

	input := &sqs.CreateQueueInput{
		QueueName:  aws_sdk.String(fifoName),
		Attributes: attributes,
	}

	result, err := q.client.CreateQueue(ctx, input)
	if err != nil {
		slog.Error("exception occured when creating FIFO queue", "error", err)
		return "", fmt.Errorf("exception occured when creating FIFO queue. %w", err)
	}
	return *result.QueueUrl, nil
}

func (q *QueueManager) DeleteQueue(ctx context.Context, queueURL string) error {
	input := &sqs.DeleteQueueInput{
		QueueUrl: aws_sdk.String(queueURL),
	}

	_, err := q.client.DeleteQueue(ctx, input)
	if err != nil {
		slog.Error("deleting queue", "error", err)
		return fmt.Errorf("deleting queue: %w", err)
	}

	return nil
}

func (q *QueueManager) GetUrl(ctx context.Context, name string) (string, error) {
	input := sqs.GetQueueUrlInput{
		QueueName: aws_sdk.String(name),
	}
	result, err := q.client.GetQueueUrl(ctx, &input)
	if err != nil {
		slog.Error("exception occured when creating FIFO queue", "error", err)
		return "", fmt.Errorf("exception occured when creating FIFO queue. %w", err)
	}
	return *result.QueueUrl, nil
}

func (q *QueueManager) PurgeQueue(ctx context.Context, queueURL string) error {
	input := &sqs.PurgeQueueInput{
		QueueUrl: aws_sdk.String(queueURL),
	}

	_, err := q.client.PurgeQueue(ctx, input)
	if err != nil {
		slog.Error("purging queue", "error", err)
		return fmt.Errorf("purging queue: %w", err)
	}

	return nil
}

// ConfigureDeadLetterQueue sets up a DLQ for the main queue
func (q *QueueManager) SetDlq(
	ctx context.Context,
	mainQueueUrl string,
	arn string,
	maxReceiveCount int) error {
	// Create the redrive policy JSON
	redrivePolicy := fmt.Sprintf(
		`{"deadLetterTargetArn":"%s","maxReceiveCount":"%d"}`,
		arn,
		maxReceiveCount,
	)

	input := &sqs.SetQueueAttributesInput{
		QueueUrl: aws_sdk.String(mainQueueUrl),
		Attributes: map[string]string{
			"RedrivePolicy": redrivePolicy,
		},
	}

	_, err := q.client.SetQueueAttributes(ctx, input)

	if err != nil {
		slog.Error("configuring dead letter queue.", "error", err)
		return fmt.Errorf("configuring dead letter queue: %w", err)
	}

	return nil
}

// GetQueueARN retrieves the ARN for a queue given its URL
func (q *QueueManager) GetQueueARN(ctx context.Context, queueURL string) (string, error) {
	input := &sqs.GetQueueAttributesInput{
		QueueUrl:       aws_sdk.String(queueURL),
		AttributeNames: []types.QueueAttributeName{types.QueueAttributeNameQueueArn},
	}

	result, err := q.client.GetQueueAttributes(ctx, input)
	if err != nil {
		return "", fmt.Errorf("getting queue ARN: %w", err)
	}

	return result.Attributes["QueueArn"], nil
}

func calcWaitTime(waitTime ...int) int {
	wt := 20
	if len(waitTime) > 0 {
		wt = waitTime[0]
	}
	return wt
}
