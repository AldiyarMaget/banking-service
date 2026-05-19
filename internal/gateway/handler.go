package gateway

import (
	"encoding/json"
	"net/http"

	accountv1 "github.com/AldiyarMaget/banking-generated/gen/go/proto/account/v1"
	transactionv1 "github.com/AldiyarMaget/banking-generated/gen/go/proto/transaction/v1"
	"github.com/go-chi/chi/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	userv1 "github.com/AldiyarMaget/banking-generated/gen/go/proto/user/v1"
	analyticsv1 "github.com/AldiyarMaget/banking-generated/gen/go/proto/analytics/v1"
)

type Handler struct {
	accountClient accountv1.AccountServiceClient
	txClient      transactionv1.TransactionServiceClient
	userClient    userv1.UserServiceClient
	analyticsClient analyticsv1.AnalyticsServiceClient
}

func NewHandler(ac accountv1.AccountServiceClient, tc transactionv1.TransactionServiceClient, uc userv1.UserServiceClient, anc analyticsv1.AnalyticsServiceClient) *Handler {
	return &Handler{
		accountClient: ac,
		txClient:      tc,
		userClient:    uc,
		analyticsClient: anc,
	}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	// Account Service
	r.Post("/api/v1/accounts", h.CreateAccount)
	r.Get("/api/v1/accounts/{id}", h.GetAccount)
	r.Post("/api/v1/accounts/{id}/balance", h.UpdateBalance)
	r.Get("/api/v1/accounts/{id}/history", h.GetAccountHistory)
	r.Post("/api/v1/accounts/{id}/freeze", h.FreezeAccount)
	r.Delete("/api/v1/accounts/{id}", h.CloseAccount)
	r.Patch("/api/v1/accounts/{id}/status", h.UpdateAccountStatus)

	// Transaction Service
	r.Post("/api/v1/transfers", h.TransferFunds)
	r.Get("/api/v1/transfers/{id}", h.GetTransactionStatus)

	// User Service
	r.Post("/api/v1/users/register", h.RegisterUser)
	r.Get("/api/v1/users/{id}", h.GetUserProfile)

	// Analytics Service
	r.Post("/api/v1/analytics/limit", h.SetDailyLimit)
}

func (h *Handler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	var dto CreateAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	req := &accountv1.CreateAccountRequest{
		CustomerId:     dto.CustomerID,
		Currency:       dto.Currency,
		IdempotencyKey: dto.IdempotencyKey,
	}

	res, err := h.accountClient.CreateAccount(r.Context(), req)
	if err != nil {
		h.handleGrpcError(w, err)
		return
	}

	h.respondJSON(w, http.StatusOK, CreateAccountResponse{
		AccountID: res.AccountId,
		Status:    res.Status,
	})
}

func (h *Handler) GetAccount(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	req := &accountv1.GetAccountRequest{
		AccountId: id,
	}

	res, err := h.accountClient.GetAccount(r.Context(), req)
	if err != nil {
		h.handleGrpcError(w, err)
		return
	}

	h.respondJSON(w, http.StatusOK, GetAccountResponse{
		AccountID:  res.AccountId,
		CustomerID: res.CustomerId,
		Balance:    res.Balance,
		Currency:   res.Currency,
	})
}

func (h *Handler) TransferFunds(w http.ResponseWriter, r *http.Request) {
	var dto TransferRequest
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	req := &transactionv1.TransferRequest{
		SourceAccountId:      dto.SourceAccountID,
		DestinationAccountId: dto.DestinationAccountID,
		Amount:               dto.Amount,
		Currency:             dto.Currency,
		IdempotencyKey:       dto.IdempotencyKey,
	}

	res, err := h.txClient.TransferFunds(r.Context(), req)
	if err != nil {
		h.handleGrpcError(w, err)
		return
	}

	h.respondJSON(w, http.StatusOK, TransferResponse{
		TransactionID: res.TransactionId,
		Status:        res.Status,
	})
}

func (h *Handler) handleGrpcError(w http.ResponseWriter, err error) {
	st, ok := status.FromError(err)
	if !ok {
		h.respondError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	var httpStatus int
	switch st.Code() {
	case codes.InvalidArgument:
		httpStatus = http.StatusBadRequest
	case codes.NotFound:
		httpStatus = http.StatusNotFound
	case codes.AlreadyExists:
		httpStatus = http.StatusConflict
	case codes.Unavailable:
		httpStatus = http.StatusServiceUnavailable
	case codes.Unimplemented:
		httpStatus = http.StatusNotImplemented
	default:
		httpStatus = http.StatusInternalServerError
	}

	h.respondError(w, httpStatus, st.Message())
}

func (h *Handler) respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

func (h *Handler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, map[string]string{"error": message})
}

func (h *Handler) UpdateBalance(w http.ResponseWriter, r *http.Request) { h.respondJSON(w, http.StatusOK, map[string]string{"status": "SUCCESS"}) }
func (h *Handler) GetAccountHistory(w http.ResponseWriter, r *http.Request) { h.respondJSON(w, http.StatusOK, map[string]string{"status": "SUCCESS"}) }
func (h *Handler) FreezeAccount(w http.ResponseWriter, r *http.Request) { h.respondJSON(w, http.StatusOK, map[string]string{"status": "SUCCESS"}) }
func (h *Handler) CloseAccount(w http.ResponseWriter, r *http.Request) { h.respondJSON(w, http.StatusOK, map[string]string{"status": "SUCCESS"}) }
func (h *Handler) UpdateAccountStatus(w http.ResponseWriter, r *http.Request) { h.respondJSON(w, http.StatusOK, map[string]string{"status": "SUCCESS"}) }
func (h *Handler) GetTransactionStatus(w http.ResponseWriter, r *http.Request) { h.respondJSON(w, http.StatusOK, map[string]string{"status": "SUCCESS"}) }
func (h *Handler) RegisterUser(w http.ResponseWriter, r *http.Request) { h.respondJSON(w, http.StatusOK, map[string]string{"status": "SUCCESS"}) }
func (h *Handler) GetUserProfile(w http.ResponseWriter, r *http.Request) { h.respondJSON(w, http.StatusOK, map[string]string{"status": "SUCCESS"}) }
func (h *Handler) SetDailyLimit(w http.ResponseWriter, r *http.Request) { h.respondJSON(w, http.StatusOK, map[string]string{"status": "SUCCESS"}) }
