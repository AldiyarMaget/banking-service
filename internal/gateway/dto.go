package gateway

type CreateAccountRequest struct {
	CustomerID     string `json:"customer_id"`
	Currency       string `json:"currency"`
	IdempotencyKey string `json:"idempotency_key"`
}

type CreateAccountResponse struct {
	AccountID string `json:"account_id"`
	Status    string `json:"status"`
}

type GetAccountResponse struct {
	AccountID  string `json:"account_id"`
	CustomerID string `json:"customer_id"`
	Balance    int64  `json:"balance"`
	Currency   string `json:"currency"`
}

type TransferRequest struct {
	SourceAccountID      string `json:"source_account_id"`
	DestinationAccountID string `json:"destination_account_id"`
	Amount               int64  `json:"amount"`
	Currency             string `json:"currency"`
	IdempotencyKey       string `json:"idempotency_key"`
}

type TransferResponse struct {
	TransactionID string `json:"transaction_id"`
	Status        string `json:"status"`
}
