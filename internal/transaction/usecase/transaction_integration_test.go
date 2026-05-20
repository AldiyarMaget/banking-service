//go:build integration

package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"banking-service/internal/transaction/repository"
	txworker "banking-service/internal/transaction/worker"
	accountv1 "github.com/AldiyarMaget/banking-generated/gen/go/proto/account/v1"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
)

type integrationAccountClient struct {
	updates []*accountv1.UpdateBalanceRequest
}

func (c *integrationAccountClient) CreateAccount(ctx context.Context, in *accountv1.CreateAccountRequest, opts ...grpc.CallOption) (*accountv1.CreateAccountResponse, error) {
	return nil, errors.New("CreateAccount is not used by transaction integration tests")
}

func (c *integrationAccountClient) GetAccount(ctx context.Context, in *accountv1.GetAccountRequest, opts ...grpc.CallOption) (*accountv1.GetAccountResponse, error) {
	return nil, errors.New("GetAccount is not used by transaction integration tests")
}

func (c *integrationAccountClient) UpdateBalance(ctx context.Context, in *accountv1.UpdateBalanceRequest, opts ...grpc.CallOption) (*accountv1.UpdateBalanceResponse, error) {
	c.updates = append(c.updates, in)
	return &accountv1.UpdateBalanceResponse{AccountId: in.AccountId}, nil
}

func (c *integrationAccountClient) GetAccountHistory(ctx context.Context, in *accountv1.GetAccountHistoryRequest, opts ...grpc.CallOption) (*accountv1.GetAccountHistoryResponse, error) {
	return nil, errors.New("GetAccountHistory is not used by transaction integration tests")
}

func (c *integrationAccountClient) FreezeAccount(ctx context.Context, in *accountv1.FreezeAccountRequest, opts ...grpc.CallOption) (*accountv1.FreezeAccountResponse, error) {
	return nil, errors.New("FreezeAccount is not used by transaction integration tests")
}

func (c *integrationAccountClient) CloseAccount(ctx context.Context, in *accountv1.CloseAccountRequest, opts ...grpc.CallOption) (*accountv1.CloseAccountResponse, error) {
	return nil, errors.New("CloseAccount is not used by transaction integration tests")
}

func (c *integrationAccountClient) UpdateAccountStatus(ctx context.Context, in *accountv1.UpdateAccountStatusRequest, opts ...grpc.CallOption) (*accountv1.UpdateAccountStatusResponse, error) {
	return nil, errors.New("UpdateAccountStatus is not used by transaction integration tests")
}

type recordingPublisher struct {
	subjects []string
	payloads [][]byte
}

func (p *recordingPublisher) Publish(subject string, payload []byte) error {
	p.subjects = append(p.subjects, subject)
	copied := append([]byte(nil), payload...)
	p.payloads = append(p.payloads, copied)
	return nil
}

func TestTransferFundsCreatesTransactionAndOutboxEventIntegration(t *testing.T) {
	ctx := context.Background()
	pool := openIntegrationPool(t, ctx)

	resetTransactionTables(t, ctx, pool)

	txManager := repository.NewTransactionManager(pool)
	txRepo := repository.NewTransactionRepository(pool)
	outboxRepo := repository.NewOutboxRepository(pool)
	idemRepo := repository.NewIdempotencyRepository(pool)
	accountClient := &integrationAccountClient{}

	uc := NewTransactionUseCase(txManager, txRepo, outboxRepo, idemRepo, accountClient)

	record, err := uc.TransferFunds(
		ctx,
		"itest-transfer-001",
		"acc-source-itest",
		"acc-destination-itest",
		"KZT",
		2500,
	)
	if err != nil {
		t.Fatalf("TransferFunds returned error: %v", err)
	}
	if record == nil {
		t.Fatal("expected transaction record, got nil")
	}
	if record.Status != "COMPLETED" {
		t.Fatalf("expected transaction status COMPLETED, got %q", record.Status)
	}
	if len(accountClient.updates) != 2 {
		t.Fatalf("expected 2 account UpdateBalance calls, got %d", len(accountClient.updates))
	}
	if accountClient.updates[0].AccountId != "acc-source-itest" || accountClient.updates[0].Amount != -2500 {
		t.Fatalf("unexpected debit request: %+v", accountClient.updates[0])
	}
	if accountClient.updates[1].AccountId != "acc-destination-itest" || accountClient.updates[1].Amount != 2500 {
		t.Fatalf("unexpected credit request: %+v", accountClient.updates[1])
	}

	var transactionCount int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM transactions
		WHERE id = $1
		  AND source_account_id = $2
		  AND destination_account_id = $3
		  AND amount = $4
		  AND currency = $5
		  AND status = $6
	`, record.ID, "acc-source-itest", "acc-destination-itest", int64(2500), "KZT", "COMPLETED").Scan(&transactionCount)
	if err != nil {
		t.Fatalf("query transaction row: %v", err)
	}
	if transactionCount != 1 {
		t.Fatalf("expected 1 saved transaction row, got %d", transactionCount)
	}

	var idempotencyCount int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM processed_requests
		WHERE idempotency_key = $1
		  AND status = $2
	`, "itest-transfer-001", "COMPLETED").Scan(&idempotencyCount)
	if err != nil {
		t.Fatalf("query idempotency row: %v", err)
	}
	if idempotencyCount != 1 {
		t.Fatalf("expected 1 idempotency row, got %d", idempotencyCount)
	}

	var eventType string
	var aggregateType string
	var aggregateID string
	var rawPayload []byte
	err = pool.QueryRow(ctx, `
		SELECT event_type, aggregate_type, aggregate_id, payload
		FROM outbox_events
		WHERE aggregate_id = $1
	`, record.ID).Scan(&eventType, &aggregateType, &aggregateID, &rawPayload)
	if err != nil {
		t.Fatalf("query outbox event: %v", err)
	}
	if eventType != "TransferCompleted" {
		t.Fatalf("expected event type TransferCompleted, got %q", eventType)
	}
	if aggregateType != "transaction" {
		t.Fatalf("expected aggregate type transaction, got %q", aggregateType)
	}
	if aggregateID != record.ID {
		t.Fatalf("expected aggregate id %q, got %q", record.ID, aggregateID)
	}

	var payload TransferCompletedPayload
	if err := json.Unmarshal(rawPayload, &payload); err != nil {
		t.Fatalf("outbox payload is invalid JSON: %v", err)
	}
	if payload.TransactionID != record.ID || payload.Source != "acc-source-itest" || payload.Destination != "acc-destination-itest" || payload.Amount != 2500 || payload.Currency != "KZT" {
		t.Fatalf("unexpected outbox payload: %+v", payload)
	}
}

func TestTransactionOutboxRelayProcessesPendingEventsIntegration(t *testing.T) {
	ctx := context.Background()
	pool := openIntegrationPool(t, ctx)

	resetTransactionTables(t, ctx, pool)

	_, err := pool.Exec(ctx, `
		INSERT INTO outbox_events (aggregate_type, aggregate_id, event_type, payload, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, "transaction", "tx-itest-001", "TransferCompleted", []byte(`{"transaction_id":"tx-itest-001","amount":1000}`), time.Now().UTC())
	if err != nil {
		t.Fatalf("insert pending outbox event: %v", err)
	}

	txManager := repository.NewTransactionManager(pool)
	outboxRepo := repository.NewOutboxRepository(pool)
	publisher := &recordingPublisher{}
	relay := txworker.NewOutboxRelay(txManager, outboxRepo, publisher)

	processed, err := relay.ProcessOnce(ctx)
	if err != nil {
		t.Fatalf("ProcessOnce returned error: %v", err)
	}
	if processed != 1 {
		t.Fatalf("expected 1 processed event, got %d", processed)
	}
	if len(publisher.payloads) != 1 {
		t.Fatalf("expected 1 published message, got %d", len(publisher.payloads))
	}
	if len(publisher.subjects) != 1 || publisher.subjects[0] != "banking.transfer.completed" {
		t.Fatalf("unexpected publish subject list: %+v", publisher.subjects)
	}

	var remaining int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM outbox_events
		WHERE aggregate_id = $1
	`, "tx-itest-001").Scan(&remaining)
	if err != nil {
		t.Fatalf("query remaining outbox events: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("expected processed event to be deleted from outbox_events, remaining rows: %d", remaining)
	}
}

func openIntegrationPool(t *testing.T, ctx context.Context) *pgxpool.Pool {
	t.Helper()

	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set TEST_DATABASE_URL to run integration tests, for example: postgres://user:password@localhost:5433/banking_transactions?sslmode=disable")
	}

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect to integration database: %v", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatalf("ping integration database: %v", err)
	}

	createTransactionSchema(t, ctx, pool)
	t.Cleanup(pool.Close)

	return pool
}

func createTransactionSchema(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()

	statements := []string{
		`CREATE EXTENSION IF NOT EXISTS pgcrypto`,
		`CREATE TABLE IF NOT EXISTS transactions (
			id VARCHAR(36) PRIMARY KEY,
			source_account_id VARCHAR(36) NOT NULL,
			destination_account_id VARCHAR(36) NOT NULL,
			amount BIGINT NOT NULL,
			currency VARCHAR(3) NOT NULL,
			status VARCHAR(50) NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS outbox_events (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			aggregate_type VARCHAR(255) NOT NULL,
			aggregate_id VARCHAR(36) NOT NULL,
			event_type VARCHAR(255) NOT NULL,
			payload JSONB NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			processed_at TIMESTAMP WITH TIME ZONE
		)`,
		`CREATE TABLE IF NOT EXISTS processed_requests (
			idempotency_key VARCHAR(256) PRIMARY KEY,
			status VARCHAR(50),
			responded_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, statement := range statements {
		if _, err := pool.Exec(ctx, statement); err != nil {
			t.Fatalf("apply integration schema: %v\nSQL: %s", err, statement)
		}
	}
}

func resetTransactionTables(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()

	if _, err := pool.Exec(ctx, `TRUNCATE TABLE outbox_events, processed_requests, transactions`); err != nil {
		t.Fatalf("reset integration tables: %v", err)
	}
}
