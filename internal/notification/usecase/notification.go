package usecase

import (
	"context"
	"encoding/json"
	"fmt"

	"banking-service/internal/notification/domain"
)

type notificationUseCase struct {
	mailer domain.EmailSender
}

func NewNotificationUseCase(mailer domain.EmailSender) domain.NotificationUseCase {
	return &notificationUseCase{mailer: mailer}
}

type AccountCreatedPayload struct {
	ID       string `json:"id"`
	UserID   string `json:"user_id"`
	Currency string `json:"currency"`
}

func (u *notificationUseCase) HandleAccountCreated(ctx context.Context, correlationID string, payload []byte) error {
	var evt AccountCreatedPayload
	if err := json.Unmarshal(payload, &evt); err != nil {
		// Permanent error indicator
		return fmt.Errorf("permanent: failed to unmarshal account payload: %w", err)
	}

	html := fmt.Sprintf("<h1>Welcome!</h1><p>Your account <b>%s</b> has been created.</p><p>Currency: %s</p>", evt.ID, evt.Currency)
	
	job := domain.EmailJob{
		CorrelationID: correlationID,
		To:            "user_" + evt.UserID + "@example.com", // Mock email lookup based on UserID
		Subject:       "Welcome to Banking Service",
		HTMLBody:      html,
	}

	return u.mailer.Send(ctx, job)
}

type TransferCompletedPayload struct {
	TransactionID string `json:"transaction_id"`
	Source        string `json:"source_account_id"`
	Destination   string `json:"destination_account_id"`
	Amount        int64  `json:"amount"`
	Currency      string `json:"currency"`
}

func (u *notificationUseCase) HandleTransactionCompleted(ctx context.Context, correlationID string, payload []byte) error {
	var evt TransferCompletedPayload
	if err := json.Unmarshal(payload, &evt); err != nil {
		return fmt.Errorf("permanent: failed to unmarshal transaction payload: %w", err)
	}

	// Assuming amount is in cents, simple division for display
	displayAmount := float64(evt.Amount) / 100.0
	html := fmt.Sprintf("<h1>Transfer Successful</h1><p>TxID: %s</p><p>Amount: %.2f %s</p><p>Destination: %s</p>", 
		evt.TransactionID, displayAmount, evt.Currency, evt.Destination)

	job := domain.EmailJob{
		CorrelationID: correlationID,
		To:            "user_tx_" + evt.Source + "@example.com",
		Subject:       "Transfer Completed Successfully",
		HTMLBody:      html,
	}

	return u.mailer.Send(ctx, job)
}
