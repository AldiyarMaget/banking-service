package domain

import (
	"context"
	"errors"
	"time"
)

var ErrAlreadyProcessed = errors.New("request already processed")
var ErrInsufficientFunds = errors.New("insufficient funds")
var ErrAccountFrozenOrClosed = errors.New("account is frozen or closed")

type Account struct {
	ID           string
	UserID       string
	Balance      int64 // In cents/kopecks to avoid floating point issues
	Currency     string
	Status       string // 'ACTIVE', 'FROZEN', 'CLOSED', etc.
	FreezeReason *string
	CreatedAt    time.Time
}

type AccountHistory struct {
	ID             string
	AccountID      string
	Type           string // 'DEPOSIT', 'TRANSFER_IN', 'TRANSFER_OUT', etc.
	Amount         int64
	IdempotencyKey string
	Reason         string
	CreatedAt      time.Time
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
	GetAccount(ctx context.Context, id string) (*Account, error)
	UpdateBalance(ctx context.Context, id string, amount int64) (int64, error)
	FreezeAccount(ctx context.Context, id string, reason string) error
	CloseAccount(ctx context.Context, id string) error
	UpdateAccountStatus(ctx context.Context, id string, status string) error
	RecordHistory(ctx context.Context, history *AccountHistory) error
	GetAccountHistory(ctx context.Context, accountID string, limit, offset int32) ([]*AccountHistory, error)
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
	GetAccount(ctx context.Context, id string) (*Account, error)
	UpdateBalance(ctx context.Context, idempotencyKey, id string, amount int64, reason string) (int64, error)
	FreezeAccount(ctx context.Context, id string, reason string) error
	CloseAccount(ctx context.Context, id string) error
	UpdateAccountStatus(ctx context.Context, id string, status string) error
	GetAccountHistory(ctx context.Context, accountID string, limit, offset int32) ([]*AccountHistory, error)
}
