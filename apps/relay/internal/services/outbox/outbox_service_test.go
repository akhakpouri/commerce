package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"commerce/internal/shared/events"
	manager "commerce/internal/shared/managers/transaction"
	"commerce/internal/shared/models"
	orderrepo "commerce/internal/shared/repositories/order"
	repo "commerce/internal/shared/repositories/outbox"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"gorm.io/datatypes"
)

// fakeManager mirrors the pattern in apps/api/internal/services/order:
// instead of opening a real transaction, Execute runs the callback with fake
// repositories, so the service's transactional calls land on the mock.
type fakeManager struct {
	repos manager.RepositoriesI
}

func (f *fakeManager) Execute(fn func(r manager.RepositoriesI) error) error {
	return fn(f.repos)
}

type fakeRepositories struct {
	outbox repo.OutboxRepositoryI
}

// Order() is part of manager.RepositoriesI (shared with apps/api) but
// ProcessBatch never calls it - returning nil is safe.
func (f *fakeRepositories) Order() orderrepo.OrderRepositoryI { return nil }
func (f *fakeRepositories) Outbox() repo.OutboxRepositoryI    { return f.outbox }

// fakePublisher is a hand-written fake for the Publisher port - per the
// design spec, this service is tested against a fake, not a mock of SQS.
type fakePublisher struct {
	envelopes []events.Envelope
	result    PublishResult
	err       error
	calls     int
}

func (f *fakePublisher) Publish(_ context.Context, envelopes []events.Envelope) (PublishResult, error) {
	f.calls++
	f.envelopes = envelopes
	return f.result, f.err
}

func newMockRepo(t *testing.T) *MockOutboxRepositoryI {
	t.Helper()
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	return NewMockOutboxRepositoryI(ctrl)
}

func setup(mockRepo *MockOutboxRepositoryI, publisher *fakePublisher, maxAttempts int) OutboxServiceI {
	txm := &fakeManager{repos: &fakeRepositories{outbox: mockRepo}}
	return NewOutboxService(txm, publisher, maxAttempts)
}

func TestProcessBatch_EmptyClaim(t *testing.T) {
	mockRepo := newMockRepo(t)
	publisher := &fakePublisher{}
	svc := setup(mockRepo, publisher, 10)

	mockRepo.EXPECT().GetNextBatch(10, 10).Return(nil, nil)

	err := svc.ProcessBatch(context.Background(), 10)

	assert.NoError(t, err)
	assert.Equal(t, 0, publisher.calls)
}

func TestProcessBatch_AllAccepted(t *testing.T) {
	mockRepo := newMockRepo(t)
	eventId := uuid.New()
	createdAt := time.Date(2026, 8, 1, 9, 0, 0, 0, time.UTC)
	row := &models.Outbox{
		Base:          models.Base{Id: 1, CreatedDate: createdAt},
		EventId:       eventId,
		EventType:     "order.placed",
		AggregateId:   42,
		AggregateType: "Order",
		Payload:       datatypes.JSON(`{"total_amount":125.5}`),
		Version:       1,
	}
	publisher := &fakePublisher{result: PublishResult{PublishedIds: []uuid.UUID{eventId}}}
	svc := setup(mockRepo, publisher, 10)

	mockRepo.EXPECT().GetNextBatch(10, 10).Return([]*models.Outbox{row}, nil)
	mockRepo.EXPECT().MarkPublished([]uint{1}).Return(nil)

	err := svc.ProcessBatch(context.Background(), 10)

	assert.NoError(t, err)
	assert.Equal(t, 1, publisher.calls)
	assert.Equal(t, []events.Envelope{{
		EventId:    eventId,
		EventType:  "order.placed",
		OccurredAt: createdAt,
		Aggregate:  events.Aggregate{Type: "Order", Id: 42},
		Version:    1,
		Payload:    json.RawMessage(`{"total_amount":125.5}`),
	}}, publisher.envelopes)
}

func TestProcessBatch_PartialRejection(t *testing.T) {
	mockRepo := newMockRepo(t)
	acceptedId := uuid.New()
	rejectedId := uuid.New()
	rows := []*models.Outbox{
		{Base: models.Base{Id: 1}, EventId: acceptedId, EventType: "order.placed", AggregateType: "Order", Payload: datatypes.JSON(`{}`)},
		{Base: models.Base{Id: 2}, EventId: rejectedId, EventType: "order.placed", AggregateType: "Order", Payload: datatypes.JSON(`{}`)},
	}
	publisher := &fakePublisher{result: PublishResult{
		PublishedIds: []uuid.UUID{acceptedId},
		FailedIds:    []uuid.UUID{rejectedId},
	}}
	svc := setup(mockRepo, publisher, 10)

	mockRepo.EXPECT().GetNextBatch(10, 10).Return(rows, nil)
	mockRepo.EXPECT().MarkPublished([]uint{1}).Return(nil)
	mockRepo.EXPECT().IncrementAttempts([]uint{2}).Return(nil)

	err := svc.ProcessBatch(context.Background(), 10)

	assert.NoError(t, err)
}

func TestProcessBatch_WholeCallError(t *testing.T) {
	mockRepo := newMockRepo(t)
	row := &models.Outbox{Base: models.Base{Id: 1}, EventId: uuid.New(), AggregateType: "Order", Payload: datatypes.JSON(`{}`)}
	publisher := &fakePublisher{err: errors.New("sqs unavailable")}
	svc := setup(mockRepo, publisher, 10)

	mockRepo.EXPECT().GetNextBatch(10, 10).Return([]*models.Outbox{row}, nil)
	// Deliberately no MarkPublished/IncrementAttempts expectation - gomock
	// fails the test if either is called, proving nothing gets marked on a
	// whole-call error.

	err := svc.ProcessBatch(context.Background(), 10)

	assert.Error(t, err)
}
