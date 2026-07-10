package aws

import "time"

type Message struct {
	Id            string            `json:"id"`
	Type          string            `json:"type"`
	Payload       string            `json:"payload"`
	Timestamp     time.Time         `json:"timestamp"`
	ReceiptHandle string            `json:"-"`
	Attributes    map[string]string `json:"-"`
}
