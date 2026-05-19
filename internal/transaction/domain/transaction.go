package domain

import (
	"context"
	"errors"
	"time"
)

var ErrAlreadyProcessed = errors.New("request already processed")

type TransactionRecord struct {
	ID                   string
	SourceAccountID      string
	DestinationAccountID string
	Amount               int64
	Currency             string
	Status               string
	CreatedAt            time.Time
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

type TransactionManager interface {
	DoInTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type TransactionRepository interface {
	CreateTransaction(ctx context.Context, record *TransactionRecord) error
}

type OutboxRepository interface {
	CreateEvent(ctx context.Context, event *OutboxEvent) error
	FetchUnprocessedEvents(ctx context.Context, limit int) ([]*OutboxEvent, error)
	MarkProcessed(ctx context.Context, id string) error
}

type IdempotencyRepository interface {
	MarkProcessing(ctx context.Context, key string) (bool, error)
}

type TransactionUseCase interface {
	TransferFunds(ctx context.Context, idempotencyKey, sourceAccountID, destAccountID, currency string, amount int64) (*TransactionRecord, error)
}
