package queue

import (
	"context"

	manager "commerce/internal/shared/managers/aws"
)

type RelayManagerI interface {
	GetOrCreateQueue(url string)
}

type RelayManager struct {
	queue        string
	queueManager manager.QueueManager
}

// GetOrCreateQueue implements [QueueManagerI].
func (q *RelayManager) GetOrCreateQueue(url string) {
	panic("unimplemented")
}

func (q *RelayManager) getQueue(ctx context.Context, url string) {
}

func NewRelayManager(queue string, queueMnager *manager.QueueManager) RelayManagerI {
	return &RelayManager{
		queue:        queue,
		queueManager: *queueMnager,
	}
}
