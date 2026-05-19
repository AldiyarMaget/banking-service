package domain

import "context"

type EmailJob struct {
	CorrelationID string
	To            string
	Subject       string
	HTMLBody      string
}

type EmailSender interface {
	Send(ctx context.Context, job EmailJob) error
}

type NotificationUseCase interface {
	HandleAccountCreated(ctx context.Context, correlationID string, payload []byte) error
	HandleTransactionCompleted(ctx context.Context, correlationID string, payload []byte) error
}
