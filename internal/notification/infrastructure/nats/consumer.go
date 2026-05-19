package nats

import (
	"context"
	"log"
	"math/rand"
	"strings"
	"time"

	"banking-service/internal/notification/domain"
	"github.com/nats-io/nats.go"
)

type Consumer struct {
	js      nats.JetStreamContext
	usecase domain.NotificationUseCase
}

func NewConsumer(js nats.JetStreamContext, uc domain.NotificationUseCase) *Consumer {
	return &Consumer{
		js:      js,
		usecase: uc,
	}
}

func (c *Consumer) Start(ctx context.Context, subject string, consumerName string) error {
	// Create durable consumer binding
	sub, err := c.js.PullSubscribe(subject, consumerName, nats.BindStream("BANKING"))
	if err != nil {
		// Fallback without bind if Stream config does not strictly match or hasn't bound subjects yet
		sub, err = c.js.PullSubscribe(subject, consumerName)
		if err != nil {
			return err
		}
	}

	log.Printf("Started Durable NATS Consumer '%s' on subject '%s'", consumerName, subject)

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			// Fetch up to 10 messages, block until ctx is canceled
			msgs, err := sub.Fetch(10, nats.Context(ctx))
			if err != nil {
				if err == context.Canceled {
					return nil
				}
				// Fetch timeout or empty batch is normal
				continue
			}

			for _, msg := range msgs {
				c.processMessage(ctx, msg)
			}
		}
	}
}

func (c *Consumer) processMessage(ctx context.Context, msg *nats.Msg) {
	// Extract Correlation ID from headers
	correlationID := msg.Header.Get("Correlation-ID")
	if correlationID == "" {
		correlationID = "missing-correlation-id"
	}

	// Retry Policy with Exponential Backoff + Jitter
	maxRetries := 5
	baseDelay := 100 * time.Millisecond

	for attempt := 0; attempt < maxRetries; attempt++ {
		var err error

		// Route message to correct UseCase handler
		if strings.HasSuffix(msg.Subject, "account.created") {
			err = c.usecase.HandleAccountCreated(ctx, correlationID, msg.Data)
		} else if strings.HasSuffix(msg.Subject, "transaction.completed") {
			err = c.usecase.HandleTransactionCompleted(ctx, correlationID, msg.Data)
		} else {
			log.Printf("[Consumer] Permanent error: unknown subject %s", msg.Subject)
			msg.Term() // Terminate message (Dead Letter)
			return
		}

		if err == nil {
			// Success -> Ack strictly AFTER mailer.Send completes successfully
			if errAck := msg.Ack(); errAck != nil {
				log.Printf("[Consumer] Failed to ack message %s: %v", correlationID, errAck)
			}
			return
		}

		// Check if permanent error (e.g., malformed JSON)
		if strings.HasPrefix(err.Error(), "permanent:") {
			log.Printf("[Consumer] Permanent error processing message %s: %v", correlationID, err)
			msg.Term() // Terminate message (Dead Letter)
			return
		}

		// Transient Error -> Backoff & Retry
		log.Printf("[Consumer] Transient error on attempt %d for %s: %v", attempt+1, correlationID, err)
		
		// Backoff with jitter calculation
		delay := baseDelay * (1 << attempt)
		jitter := time.Duration(rand.Intn(int(delay/2) + 1))
		
		select {
		case <-time.After(delay + jitter):
			// Retry loop continues
		case <-ctx.Done():
			// Shutting down, do NOT ack. Nak it so it goes back to queue immediately.
			msg.Nak()
			return
		}
	}

	// Max retries exceeded -> Nak or Term
	log.Printf("[Consumer] Max retries exceeded for message %s. NAKing.", correlationID)
	msg.Nak()
}
