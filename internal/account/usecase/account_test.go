package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"banking-service/internal/account/domain"
)

type testTxManager struct {
	called bool
}

func (m *testTxManager) DoInTx(ctx context.Context, fn func(context.Context) error) error {
	m.called = true
	return fn(ctx)
}

type testAccountRepo struct {
	created *domain.Account
	err     error
}

func (r *testAccountRepo) CreateAccount(ctx context.Context, account *domain.Account) error {
	r.created = account
	return r.err
}

type testOutboxRepo struct {
	created *domain.OutboxEvent
	err     error
}

func (r *testOutboxRepo) CreateEvent(ctx context.Context, event *domain.OutboxEvent) error {
	r.created = event
	return r.err
}
func (r *testOutboxRepo) FetchUnprocessedEvents(ctx context.Context, limit int) ([]*domain.OutboxEvent, error) { return nil, nil }
func (r *testOutboxRepo) MarkProcessed(ctx context.Context, id string) error { return nil }

type testIdempotencyRepo struct {
	inserted bool
	key      string
	err      error
}

func (r *testIdempotencyRepo) MarkProcessing(ctx context.Context, key string) (bool, error) {
	r.key = key
	return r.inserted, r.err
}

func TestCreateAccountCreatesAccountAndOutboxEvent(t *testing.T) {
	txManager := &testTxManager{}
	accountRepo := &testAccountRepo{}
	outboxRepo := &testOutboxRepo{}
	idemRepo := &testIdempotencyRepo{inserted: true}

	uc := NewAccountUseCase(txManager, accountRepo, outboxRepo, idemRepo)

	account, err := uc.CreateAccount(context.Background(), "idem-1", "acc-1", "customer-1", "KZT")
	if err != nil {
		t.Fatalf("CreateAccount returned error: %v", err)
	}
	if !txManager.called {
		t.Fatal("expected use case to run inside transaction")
	}
	if account.ID != "acc-1" || account.UserID != "customer-1" || account.Currency != "KZT" {
		t.Fatalf("unexpected account: %+v", account)
	}
	if accountRepo.created == nil {
		t.Fatal("expected account repository to be called")
	}
	if outboxRepo.created == nil {
		t.Fatal("expected outbox event to be created")
	}
	if outboxRepo.created.EventType != "AccountCreated" || outboxRepo.created.AggregateID != "acc-1" {
		t.Fatalf("unexpected outbox event: %+v", outboxRepo.created)
	}

	var payload AccountCreatedPayload
	if err := json.Unmarshal(outboxRepo.created.Payload, &payload); err != nil {
		t.Fatalf("outbox payload is invalid JSON: %v", err)
	}
	if payload.ID != "acc-1" || payload.UserID != "customer-1" || payload.Currency != "KZT" {
		t.Fatalf("unexpected outbox payload: %+v", payload)
	}
}

func TestCreateAccountRejectsDuplicateIdempotencyKey(t *testing.T) {
	uc := NewAccountUseCase(
		&testTxManager{},
		&testAccountRepo{},
		&testOutboxRepo{},
		&testIdempotencyRepo{inserted: false},
	)

	_, err := uc.CreateAccount(context.Background(), "idem-1", "acc-1", "customer-1", "KZT")
	if !errors.Is(err, domain.ErrAlreadyProcessed) {
		t.Fatalf("expected ErrAlreadyProcessed, got %v", err)
	}
}
