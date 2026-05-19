package repository

import (
	"context"

	"banking-service/internal/transaction/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type outboxRepository struct {
	pool *pgxpool.Pool
}

func NewOutboxRepository(pool *pgxpool.Pool) domain.OutboxRepository {
	return &outboxRepository{pool: pool}
}

func (r *outboxRepository) CreateEvent(ctx context.Context, event *domain.OutboxEvent) error {
	query := `INSERT INTO outbox_events (aggregate_type, aggregate_id, event_type, payload, created_at) VALUES ($1, $2, $3, $4, $5)`
	
	tx := extractTx(ctx)
	if tx != nil {
		_, err := tx.Exec(ctx, query, event.AggregateType, event.AggregateID, event.EventType, event.Payload, event.CreatedAt)
		return err
	}
	_, err := r.pool.Exec(ctx, query, event.AggregateType, event.AggregateID, event.EventType, event.Payload, event.CreatedAt)
	return err
}

func (r *outboxRepository) FetchUnprocessedEvents(ctx context.Context, limit int) ([]*domain.OutboxEvent, error) {
	query := `SELECT id, aggregate_type, aggregate_id, event_type, payload, created_at, processed_at 
	          FROM outbox_events 
	          WHERE processed_at IS NULL 
	          ORDER BY created_at ASC 
	          LIMIT $1 
	          FOR UPDATE SKIP LOCKED`
	
	tx := extractTx(ctx)
	var rows pgx.Rows
	var err error
	if tx != nil {
		rows, err = tx.Query(ctx, query, limit)
	} else {
		rows, err = r.pool.Query(ctx, query, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*domain.OutboxEvent
	for rows.Next() {
		var e domain.OutboxEvent
		err := rows.Scan(&e.ID, &e.AggregateType, &e.AggregateID, &e.EventType, &e.Payload, &e.CreatedAt, &e.ProcessedAt)
		if err != nil {
			return nil, err
		}
		events = append(events, &e)
	}
	return events, rows.Err()
}

func (r *outboxRepository) MarkProcessed(ctx context.Context, id string) error {
	query := `DELETE FROM outbox_events WHERE id = $1`
	tx := extractTx(ctx)
	if tx != nil {
		_, err := tx.Exec(ctx, query, id)
		return err
	}
	_, err := r.pool.Exec(ctx, query, id)
	return err
}
