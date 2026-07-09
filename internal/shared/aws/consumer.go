package aws

import (
	"context"

	"commerce/internal/shared/configs"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
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
