package managers

import (
	"context"
	"log/slog"
	"time"

	outbox_service "commerce/relay/internal/services/outbox"
)

const (
	DefaultInterval  = 5 * time.Second
	DefaultBatchSize = 10
)

// RelayManagerI is the relay's whole public surface: start the poll loop and
// run until ctx is canceled. Queue infrastructure is provisioned by iac-matrix,
// not the relay, so there is no create/ensure step here (see decisions.md).
type RelayManagerI interface {
	Start(ctx context.Context) error
}

type RelayManager struct {
	outboxService outbox_service.OutboxServiceI
	interval      time.Duration
	batchSize     int
}

func NewRelayManager(outboxService outbox_service.OutboxServiceI, interval time.Duration, batchSize int) RelayManagerI {
	return &RelayManager{
		outboxService: outboxService,
		interval:      interval,
		batchSize:     batchSize,
	}
}

// Start implements [RelayManagerI]. It polls the outbox for unpublished
// events on a fixed interval until ctx is canceled, then returns ctx.Err().
func (r *RelayManager) Start(ctx context.Context) error {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := r.outboxService.ProcessBatch(ctx, r.batchSize); err != nil {
				slog.Error("exception occurred processing outbox batch", "error", err)
			}
		}
	}
}
