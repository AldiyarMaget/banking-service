package worker

import (
	"context"
	"log"
	"time"

	"banking-service/internal/account/domain"
	"banking-service/internal/account/infrastructure/nats"
)

type OutboxRelay struct {
	txManager  domain.TransactionManager
	outboxRepo domain.OutboxRepository
	jsClient   *nats.JetStreamClient
}

func NewOutboxRelay(tm domain.TransactionManager, or domain.OutboxRepository, js *nats.JetStreamClient) *OutboxRelay {
	return &OutboxRelay{
		txManager:  tm,
		outboxRepo: or,
		jsClient:   js,
	}
}

func (r *OutboxRelay) Start(ctx context.Context) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	log.Println("OutboxRelay worker started")

	for {
		select {
		case <-ctx.Done():
			log.Println("OutboxRelay worker stopped gracefully")
			return
		case <-ticker.C:
			r.processOutbox(ctx)
		}
	}
}

func (r *OutboxRelay) processOutbox(ctx context.Context) {
	// Open a transaction to lock the events
	err := r.txManager.DoInTx(ctx, func(txCtx context.Context) error {
		events, err := r.outboxRepo.FetchUnprocessedEvents(txCtx, 100) // Batch size 100
		if err != nil {
			return err
		}

		for _, evt := range events {
			// Publish to NATS
			err := r.jsClient.Publish("banking.account.created", evt.Payload)
			if err != nil {
				// NATS is unavailable or publish failed.
				// We log the error and CONTINUE. This event will remain locked until the tx commits,
				// but because we didn't call MarkProcessed, it will be picked up again in the next tick.
				log.Printf("[OutboxRelay] Failed to publish event %s (CorrelationID: %s): %v", evt.EventType, evt.ID, err)
				continue
			}

			// Successfully published, mark as processed (delete)
			err = r.outboxRepo.MarkProcessed(txCtx, evt.ID)
			if err != nil {
				// If we fail to delete, it's safer to rollback to avoid phantom events,
				// but returning error here will rollback the ENTIRE batch.
				// Let's log it and let the transaction commit what it can.
				log.Printf("[OutboxRelay] Failed to mark event %s as processed: %v", evt.ID, err)
				return err
			}
			
			log.Printf("[OutboxRelay] Successfully published and processed event %s (CorrelationID: %s)", evt.EventType, evt.ID)
		}

		return nil
	})

	if err != nil {
		log.Printf("[OutboxRelay] Transaction error: %v", err)
	}
}
