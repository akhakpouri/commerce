package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"commerce/internal/shared/configs"

	aws_sdk "github.com/aws/aws-sdk-go-v2/aws"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

type Consumer struct {
	client   *sqs.Client
	url      string
	handler  Handler
	max      int32
	timeout  int32
	waitTime int32
	count    int
}

type Handler func(ctx context.Context, message *Message) error

func NewConsumer(client *sqs.Client, cfg configs.ConsumerConfig, handler Handler) *Consumer {
	cfg = *cfg.Validate()
	return &Consumer{
		client:   client,
		url:      cfg.Url,
		handler:  handler,
		max:      cfg.Max,
		timeout:  cfg.Timeout,
		waitTime: cfg.WaitTime,
		count:    cfg.Count,
	}
}

func (c *Consumer) Start(ctx context.Context) error {
	channels := make(chan *Message, c.count*2)

	var wg sync.WaitGroup
	for i := 0; i < c.count; i++ {
		wg.Add(i)
		go func(workerId int) {
			defer wg.Done()
			c.worker(ctx, workerId, channels)
		}(i)
	}
	// Start polling loop
	go func() {
		c.poll(ctx, channels)
		close(channels)
	}()

	// Wait for workers to finish
	wg.Wait()
	return ctx.Err()
}

func (c *Consumer) poll(ctx context.Context, msgChan chan<- *Message) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			messages, err := c.recive(ctx)
			if err != nil {
				slog.Info("error receiving messages", "error", err)
				time.Sleep(time.Second) // Back off on error
				continue
			}

			for _, msg := range messages {
				select {
				case msgChan <- msg:
				case <-ctx.Done():
					return
				}
			}
		}
	}
}

func (c *Consumer) worker(ctx context.Context, _ int, msgChan <-chan *Message) {
	for msg := range msgChan {
		select {
		case <-ctx.Done():
			return
		default:
			c.process(ctx, msg)
		}
	}
}

func (c *Consumer) process(ctx context.Context, msg *Message) {
	processCtx, cancel := context.WithTimeout(ctx, time.Duration(c.timeout-5)*time.Second)
	defer cancel()

	// Call the handler
	err := c.handler(processCtx, msg)
	if err != nil {
		slog.Info("Error processing message", "id", msg.Id, "error", err)
		// Message will become visible again after visibility timeout
		return
	}

	// Delete successfully processed message
	if err := c.delete(ctx, msg.ReceiptHandle); err != nil {
		slog.Info("Error deleting message", "id", msg.Id, "error", err)
	}
}

// receiveMessages fetches messages from SQS using long polling
func (c *Consumer) recive(ctx context.Context) ([]*Message, error) {
	input := &sqs.ReceiveMessageInput{
		QueueUrl:            aws_sdk.String(c.url),
		MaxNumberOfMessages: c.max,
		VisibilityTimeout:   c.timeout,
		WaitTimeSeconds:     c.waitTime,
		// Request all available message attributes
		MessageAttributeNames: []string{"All"},
		// Request system attributes like ApproximateReceiveCount
		AttributeNames: []types.QueueAttributeName{
			types.QueueAttributeNameAll,
		},
	}

	result, err := c.client.ReceiveMessage(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("receiving messages: %w", err)
	}

	messages := make([]*Message, len(result.Messages))
	for i, sqsMsg := range result.Messages {
		var msg Message
		if err := json.Unmarshal([]byte(*sqsMsg.Body), &msg); err != nil {
			slog.Info("Error unmarshaling message", "id", *sqsMsg.MessageId, "error", err)
			continue
		}

		msg.ReceiptHandle = *sqsMsg.ReceiptHandle
		msg.Attributes = sqsMsg.Attributes
		messages[i] = &msg
	}

	return messages, nil
}

func (c *Consumer) delete(ctx context.Context, receiptHandle string) error {
	input := &sqs.DeleteMessageInput{
		QueueUrl:      aws_sdk.String(c.url),
		ReceiptHandle: aws_sdk.String(receiptHandle),
	}

	_, err := c.client.DeleteMessage(ctx, input)
	if err != nil {
		slog.Error("exception occured when deleting message", "error", err)
		return fmt.Errorf("deleting message: %w", err)
	}

	return nil
}
