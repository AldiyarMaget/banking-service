package domain

import (
	"context"
	"errors"
	"time"
)

var ErrAlreadyProcessed = errors.New("request already processed")

type Account struct {
	ID        string
	UserID    string
	Balance   int64 // In cents/kopecks to avoid floating point issues
	Currency  string
	CreatedAt time.Time
}

type OutboxEvent struct {
	ID            string
	AggregateType string
	AggregateID   string
	EventType     string
	Payload       []byte
	CreatedAt     time.Time
	ProcessedAt   *time.Time
}

// TransactionManager executes operations within a database transaction, passing tx via context
type TransactionManager interface {
	DoInTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// AccountRepository defines data access methods for accounts
type AccountRepository interface {
	CreateAccount(ctx context.Context, account *Account) error
}

// OutboxRepository defines data access methods for the Transactional Outbox
type OutboxRepository interface {
	CreateEvent(ctx context.Context, event *OutboxEvent) error
	FetchUnprocessedEvents(ctx context.Context, limit int) ([]*OutboxEvent, error)
	MarkProcessed(ctx context.Context, id string) error
}

// IdempotencyRepository manages duplicate request checks
type IdempotencyRepository interface {
	MarkProcessing(ctx context.Context, key string) (bool, error)
}

// AccountUseCase defines the business logic scenarios
type AccountUseCase interface {
	CreateAccount(ctx context.Context, idempotencyKey, id, userID, currency string) (*Account, error)
}
