package usecase

import (
	"context"
	"errors"
	"testing"

	"banking-service/internal/transaction/domain"
	accountv1 "github.com/AldiyarMaget/banking-generated/gen/go/proto/account/v1"
	"google.golang.org/grpc"
)

type testTxManager struct{ called bool }
func (m *testTxManager) DoInTx(ctx context.Context, fn func(context.Context) error) error { m.called = true; return fn(ctx) }

type testTransactionRepo struct{ created *domain.TransactionRecord }
func (r *testTransactionRepo) CreateTransaction(ctx context.Context, record *domain.TransactionRecord) error { r.created = record; return nil }

type testOutboxRepo struct{ created *domain.OutboxEvent }
func (r *testOutboxRepo) CreateEvent(ctx context.Context, event *domain.OutboxEvent) error { r.created = event; return nil }
func (r *testOutboxRepo) FetchUnprocessedEvents(ctx context.Context, limit int) ([]*domain.OutboxEvent, error) { return nil, nil }
func (r *testOutboxRepo) MarkProcessed(ctx context.Context, id string) error { return nil }

type testIdempotencyRepo struct{ inserted bool }
func (r *testIdempotencyRepo) MarkProcessing(ctx context.Context, key string) (bool, error) { return r.inserted, nil }

type testAccountClient struct{ updates []*accountv1.UpdateBalanceRequest }
func (c *testAccountClient) CreateAccount(ctx context.Context, in *accountv1.CreateAccountRequest, opts ...grpc.CallOption) (*accountv1.CreateAccountResponse, error) { return nil, errors.New("not used") }
func (c *testAccountClient) GetAccount(ctx context.Context, in *accountv1.GetAccountRequest, opts ...grpc.CallOption) (*accountv1.GetAccountResponse, error) { return nil, errors.New("not used") }
func (c *testAccountClient) UpdateBalance(ctx context.Context, in *accountv1.UpdateBalanceRequest, opts ...grpc.CallOption) (*accountv1.UpdateBalanceResponse, error) {
	c.updates = append(c.updates, in)
	return &accountv1.UpdateBalanceResponse{AccountId: in.AccountId}, nil
}
func (c *testAccountClient) GetAccountHistory(ctx context.Context, in *accountv1.GetAccountHistoryRequest, opts ...grpc.CallOption) (*accountv1.GetAccountHistoryResponse, error) { return nil, errors.New("not used") }
func (c *testAccountClient) FreezeAccount(ctx context.Context, in *accountv1.FreezeAccountRequest, opts ...grpc.CallOption) (*accountv1.FreezeAccountResponse, error) { return nil, errors.New("not used") }
func (c *testAccountClient) CloseAccount(ctx context.Context, in *accountv1.CloseAccountRequest, opts ...grpc.CallOption) (*accountv1.CloseAccountResponse, error) { return nil, errors.New("not used") }
func (c *testAccountClient) UpdateAccountStatus(ctx context.Context, in *accountv1.UpdateAccountStatusRequest, opts ...grpc.CallOption) (*accountv1.UpdateAccountStatusResponse, error) { return nil, errors.New("not used") }

func TestTransferFundsDebitsCreditsAndWritesOutbox(t *testing.T) {
	txManager := &testTxManager{}
	txRepo := &testTransactionRepo{}
	outboxRepo := &testOutboxRepo{}
	accountClient := &testAccountClient{}

	uc := NewTransactionUseCase(txManager, txRepo, outboxRepo, &testIdempotencyRepo{inserted: true}, accountClient)

	record, err := uc.TransferFunds(context.Background(), "idem-1", "acc-source", "acc-dest", "KZT", 2500)
	if err != nil {
		t.Fatalf("TransferFunds returned error: %v", err)
	}
	if !txManager.called {
		t.Fatal("expected local persistence to run inside transaction")
	}
	if len(accountClient.updates) != 2 {
		t.Fatalf("expected debit and credit calls, got %d", len(accountClient.updates))
	}
	if accountClient.updates[0].AccountId != "acc-source" || accountClient.updates[0].Amount != -2500 {
		t.Fatalf("unexpected debit call: %+v", accountClient.updates[0])
	}
	if accountClient.updates[1].AccountId != "acc-dest" || accountClient.updates[1].Amount != 2500 {
		t.Fatalf("unexpected credit call: %+v", accountClient.updates[1])
	}
	if record.Status != "COMPLETED" || txRepo.created == nil {
		t.Fatalf("transaction was not saved correctly: %+v", record)
	}
	if outboxRepo.created == nil || outboxRepo.created.EventType != "TransferCompleted" {
		t.Fatalf("expected transfer outbox event, got %+v", outboxRepo.created)
	}
}
