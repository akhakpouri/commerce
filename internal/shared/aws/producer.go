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

// RawMessage is a pre-marshalled SQS send: Body is the wire payload exactly
// as it will appear in the message body (the caller owns marshalling - e.g.
// an events.Envelope), Id correlates batch entries to their SendBatch
// result, and Attributes become SQS message attributes verbatim.
type RawMessage struct {
	Id         string
	Body       string
	Attributes map[string]string
}

// SendFIFOMessage sends a message to a FIFO queue with ordering guarantees
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

func (p *Producer) SendBatch(
	ctx context.Context,
	messages []*RawMessage) (*BatchResult, error) {
	if len(messages) == 0 {
		return nil, fmt.Errorf("messages are empty")
	} else if len(messages) > 10 {
		return nil, fmt.Errorf("batch size exceeds maximum of ten (10) messages")
	}

	entries := make([]types.SendMessageBatchRequestEntry, len(messages))

	for i, msg := range messages {
		if msg.Id == "" {
			msg.Id = uuid.New().String()
		}

		var attrs map[string]types.MessageAttributeValue
		if len(msg.Attributes) > 0 {
			attrs = make(map[string]types.MessageAttributeValue, len(msg.Attributes))
			for key, value := range msg.Attributes {
				attrs[key] = types.MessageAttributeValue{
					DataType:    aws_sdk.String("String"),
					StringValue: aws_sdk.String(value),
				}
			}
		}

		entries[i] = types.SendMessageBatchRequestEntry{
			Id:                aws_sdk.String(msg.Id),
			MessageBody:       aws_sdk.String(msg.Body),
			MessageAttributes: attrs,
		}
	}

	input := &sqs.SendMessageBatchInput{
		QueueUrl: aws_sdk.String(p.url),
		Entries:  entries,
	}

	result, err := p.client.SendMessageBatch(ctx, input)
	if err != nil {
		slog.Error("exception occured during send batch message", "error", err)
		return nil, fmt.Errorf("send batch message: %w", err)
	}

	successIds := make([]string, 0, len(result.Successful))
	failed := make([]BatchError, 0, len(result.Failed))

	for _, success := range result.Successful {
		successIds = append(successIds, *success.Id)
	}

	for _, f := range result.Failed {
		failed = append(failed, BatchError{
			Id:      *f.Id,
			Code:    *f.Code,
			Message: *f.Message,
		})
	}

	batchResult := &BatchResult{
		SuccessfulIds: successIds,
		Failed:        failed,
	}

	return batchResult, nil
}
