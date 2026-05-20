package worker

import (
	"context"
	"fmt"
	"log"

	"banking-service/internal/transaction/domain"
)

// Publisher is intentionally small so the relay can be tested without NATS.
// In production, adapt your real message broker client to this interface.
type Publisher interface {
	Publish(subject string, payload []byte) error
}

type OutboxRelay struct {
	txManager  domain.TransactionManager
	outboxRepo domain.OutboxRepository
	publisher  Publisher
	subject    string
	batchSize  int
}

func NewOutboxRelay(
	txManager domain.TransactionManager,
	outboxRepo domain.OutboxRepository,
	publisher Publisher,
) *OutboxRelay {
	return &OutboxRelay{
		txManager:  txManager,
		outboxRepo: outboxRepo,
		publisher:  publisher,
		subject:    "banking.transfer.completed",
		batchSize:  100,
	}
}

// ProcessOnce fetches one batch of pending outbox events, publishes each event,
// and marks successfully published events as processed.
//
// In this project, MarkProcessed deletes the event from outbox_events. Therefore,
// successful processing is verified by the row disappearing from the table.
func (r *OutboxRelay) ProcessOnce(ctx context.Context) (int, error) {
	processed := 0

	err := r.txManager.DoInTx(ctx, func(txCtx context.Context) error {
		events, err := r.outboxRepo.FetchUnprocessedEvents(txCtx, r.batchSize)
		if err != nil {
			return fmt.Errorf("fetch outbox events: %w", err)
		}

		for _, event := range events {
			if err := r.publisher.Publish(r.subject, event.Payload); err != nil {
				return fmt.Errorf("publish outbox event %s: %w", event.ID, err)
			}

			if err := r.outboxRepo.MarkProcessed(txCtx, event.ID); err != nil {
				return fmt.Errorf("mark outbox event %s processed: %w", event.ID, err)
			}

			processed++
			log.Printf("processed transaction outbox event id=%s type=%s", event.ID, event.EventType)
		}

		return nil
	})
	if err != nil {
		return processed, err
	}

	return processed, nil
}
