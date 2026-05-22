package grpc

import (
	"context"
	"errors"
	"time"

	"banking-service/internal/account/domain"
	accountv1 "github.com/AldiyarMaget/banking-generated/gen/go/proto/account/v1"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AccountHandler struct {
	accountv1.UnimplementedAccountServiceServer
	usecase domain.AccountUseCase
}

func NewAccountHandler(u domain.AccountUseCase) *AccountHandler {
	return &AccountHandler{
		usecase: u,
	}
}

func (h *AccountHandler) CreateAccount(ctx context.Context, req *accountv1.CreateAccountRequest) (*accountv1.CreateAccountResponse, error) {
	// Validate input
	if req.IdempotencyKey == "" {
		return nil, status.Error(codes.InvalidArgument, "idempotency_key is required")
	}
	if req.CustomerId == "" {
		return nil, status.Error(codes.InvalidArgument, "customer_id is required")
	}
	if req.Currency == "" {
		return nil, status.Error(codes.InvalidArgument, "currency is required")
	}

	accountID := uuid.New().String()

	// Call Usecase, passing the context and idempotency_key
	_, err := h.usecase.CreateAccount(ctx, req.IdempotencyKey, accountID, req.CustomerId, req.Currency)
	if err != nil {
		// Handle idempotency pattern gracefully (200 OK equivalent)
		if errors.Is(err, domain.ErrAlreadyProcessed) {
			return &accountv1.CreateAccountResponse{
				Status: "ALREADY_PROCESSED",
			}, nil
		}
		return nil, status.Errorf(codes.Internal, "failed to create account: %v", err)
	}

	return &accountv1.CreateAccountResponse{
		AccountId: accountID,
		Status:    "CREATED",
	}, nil
}

func (h *AccountHandler) GetAccount(ctx context.Context, req *accountv1.GetAccountRequest) (*accountv1.GetAccountResponse, error) {
	acc, err := h.usecase.GetAccount(ctx, req.AccountId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "account not found: %v", err)
	}

	return &accountv1.GetAccountResponse{
		AccountId:  acc.ID,
		CustomerId: acc.UserID,
		Balance:    acc.Balance,
		Currency:   acc.Currency,
	}, nil
}

// UpdateBalance is a low-level internal gRPC method responsible for atomically updating an account's balance in the database.
// It is NOT intended to be exposed directly to the public API (e.g., via API Gateway), as doing so would bypass 
// business-level constraints and open up fraud vulnerabilities.
// Instead, it should be called securely by internal services like TransactionService (during TransferFunds) 
// or by specific authorized flows (like DepositMoney in the API Gateway).
func (h *AccountHandler) UpdateBalance(ctx context.Context, req *accountv1.UpdateBalanceRequest) (*accountv1.UpdateBalanceResponse, error) {
	if req.IdempotencyKey == "" {
		return nil, status.Error(codes.InvalidArgument, "idempotency_key is required")
	}
	if req.AccountId == "" {
		return nil, status.Error(codes.InvalidArgument, "account_id is required")
	}

	newBalance, err := h.usecase.UpdateBalance(ctx, req.IdempotencyKey, req.AccountId, req.Amount, req.TransactionReason)
	if err != nil {
		if errors.Is(err, domain.ErrAlreadyProcessed) {
			return &accountv1.UpdateBalanceResponse{
				AccountId: req.AccountId,
			}, nil
		}
		if errors.Is(err, domain.ErrInsufficientFunds) {
			return nil, status.Error(codes.FailedPrecondition, "insufficient funds")
		}
		if errors.Is(err, domain.ErrAccountFrozenOrClosed) {
			return nil, status.Error(codes.FailedPrecondition, "account is frozen or closed")
		}
		return nil, status.Errorf(codes.Internal, "failed to update balance: %v", err)
	}

	return &accountv1.UpdateBalanceResponse{
		AccountId:  req.AccountId,
		NewBalance: newBalance,
	}, nil
}

func (h *AccountHandler) GetAccountHistory(ctx context.Context, req *accountv1.GetAccountHistoryRequest) (*accountv1.GetAccountHistoryResponse, error) {
	history, err := h.usecase.GetAccountHistory(ctx, req.AccountId, req.Limit, req.Offset)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get account history: %v", err)
	}

	var txs []*accountv1.Transaction
	for _, hist := range history {
		txs = append(txs, &accountv1.Transaction{
			TransactionId: hist.ID,
			Amount:        hist.Amount,
			Timestamp:     hist.CreatedAt.Format(time.RFC3339),
			Reason:        hist.Reason,
		})
	}

	return &accountv1.GetAccountHistoryResponse{
		Transactions: txs,
	}, nil
}

func (h *AccountHandler) FreezeAccount(ctx context.Context, req *accountv1.FreezeAccountRequest) (*accountv1.FreezeAccountResponse, error) {
	err := h.usecase.FreezeAccount(ctx, req.AccountId, req.Reason)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to freeze account: %v", err)
	}
	return &accountv1.FreezeAccountResponse{
		Status: "SUCCESS",
	}, nil
}

func (h *AccountHandler) CloseAccount(ctx context.Context, req *accountv1.CloseAccountRequest) (*accountv1.CloseAccountResponse, error) {
	err := h.usecase.CloseAccount(ctx, req.AccountId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to close account: %v", err)
	}
	return &accountv1.CloseAccountResponse{
		Status: "SUCCESS",
	}, nil
}

func (h *AccountHandler) UpdateAccountStatus(ctx context.Context, req *accountv1.UpdateAccountStatusRequest) (*accountv1.UpdateAccountStatusResponse, error) {
	err := h.usecase.UpdateAccountStatus(ctx, req.AccountId, req.NewStatus)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update account status: %v", err)
	}
	return &accountv1.UpdateAccountStatusResponse{
		Status: "SUCCESS",
	}, nil
}
