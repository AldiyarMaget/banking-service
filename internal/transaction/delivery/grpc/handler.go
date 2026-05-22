package grpc

import (
	"context"
	"errors"

	"banking-service/internal/transaction/domain"
	transactionv1 "github.com/AldiyarMaget/banking-generated/gen/go/proto/transaction/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type TransactionHandler struct {
	transactionv1.UnimplementedTransactionServiceServer
	usecase domain.TransactionUseCase
}

func NewTransactionHandler(u domain.TransactionUseCase) *TransactionHandler {
	return &TransactionHandler{
		usecase: u,
	}
}

func (h *TransactionHandler) TransferFunds(ctx context.Context, req *transactionv1.TransferRequest) (*transactionv1.TransferResponse, error) {
	if req.IdempotencyKey == "" {
		return nil, status.Error(codes.InvalidArgument, "idempotency_key is required")
	}
	if req.SourceAccountId == "" || req.DestinationAccountId == "" {
		return nil, status.Error(codes.InvalidArgument, "source and destination accounts are required")
	}
	if req.Amount <= 0 {
		return nil, status.Error(codes.InvalidArgument, "amount must be greater than zero")
	}

	record, err := h.usecase.TransferFunds(ctx, req.IdempotencyKey, req.SourceAccountId, req.DestinationAccountId, req.Currency, req.Amount)
	if err != nil {
		if errors.Is(err, domain.ErrAlreadyProcessed) {
			return &transactionv1.TransferResponse{
				Status: "ALREADY_PROCESSED",
			}, nil
		}
		var grpcStatus interface{ GRPCStatus() *status.Status }
		if errors.As(err, &grpcStatus) {
			return nil, grpcStatus.GRPCStatus().Err()
		}
		return nil, status.Errorf(codes.Internal, "transfer failed: %v", err)
	}

	return &transactionv1.TransferResponse{
		TransactionId: record.ID,
		Status:        record.Status,
	}, nil
}

func (h *TransactionHandler) GetTransactionStatus(ctx context.Context, req *transactionv1.GetTransactionStatusRequest) (*transactionv1.GetTransactionStatusResponse, error) {
	return &transactionv1.GetTransactionStatusResponse{}, nil
}
