package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	aws_sdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/google/uuid"
)

// purpose of the producer is to send messages to the queue
type Producer struct {
	client *sqs.Client
	url    string
}

func NewProducer(client *sqs.Client, url string) *Producer {
	return &Producer{
		client: client,
		url:    url,
	}
}

// Send a single message to the queue
func (p *Producer) Send(
	ctx context.Context,
	message *Message,
	delay int) (string, error) {
	if message.Id == "" {
		message.Id = uuid.New().String()
	}
	message.Timestamp = time.Now().UTC()

	body, err := json.Marshal(message)
	if err != nil {
		slog.Error("exception occured when marshalling message", "error", err)
		return "", fmt.Errorf("exception occured when marshalling message %w", err)
	}

	input := &sqs.SendMessageInput{
		QueueUrl:     aws_sdk.String(p.url),
		MessageBody:  aws_sdk.String(string(body)),
		DelaySeconds: int32(delay),
		MessageAttributes: map[string]types.MessageAttributeValue{
			"MessageType": {
				DataType:    aws_sdk.String("string"),
				StringValue: aws_sdk.String(message.Type),
			},
			"CorrelationId": {
				DataType:    aws_sdk.String("String"),
				StringValue: aws_sdk.String(message.Id),
			},
		},
	}
	result, err := p.client.SendMessage(ctx, input)
	if err != nil {
		slog.Error("exception occured when sending message", "error", err)
		return "", fmt.Errorf("sending message: %w", err)
	}

	return *result.MessageId, nil
}

func (p *Producer) SendFIFOMessage(
	ctx context.Context,
	message *Message,
	messageGroupId string,
	deduplicationId string,
) (string, error) {
	if message.Id == "" {
		message.Id = uuid.New().String()
	}
	message.Timestamp = time.Now().UTC()

	body, err := json.Marshal(message)
	if err != nil {
		return "", fmt.Errorf("marshaling message: %w", err)
	}

	// Use message ID as deduplication ID if not provided
	if deduplicationId == "" {
		deduplicationId = message.Id
	}

	input := &sqs.SendMessageInput{
		QueueUrl:               aws_sdk.String(p.url),
		MessageBody:            aws_sdk.String(string(body)),
		MessageGroupId:         aws_sdk.String(messageGroupId),
		MessageDeduplicationId: aws_sdk.String(deduplicationId),
	}

	result, err := p.client.SendMessage(ctx, input)
	if err != nil {
		return "", fmt.Errorf("sending FIFO message: %w", err)
	}

	return *result.MessageId, nil
}
