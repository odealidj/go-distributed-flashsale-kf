package postgres

import (
	"context"

	"github.com/jmoiron/sqlx"
	"flashsale/inventory-service/internal/application/port"
	"flashsale/shared/pkg/telemetry"
)

type outboxRepo struct {
	db *sqlx.DB
}

func NewOutboxRepo(db *sqlx.DB) port.OutboxPort {
	return &outboxRepo{db: db}
}

func (r *outboxRepo) InsertOutbox(ctx context.Context, aggregateID string, aggregateType string, eventType string, payload []byte) error {
	tracePayload := telemetry.ExtractTraceparent(ctx)
	query := `
		INSERT INTO outbox_messages (aggregate_id, aggregate_type, event_type, payload, trace_payload, status)
		VALUES ($1, $2, $3, $4, $5, 'PENDING')
	`
	_, err := r.db.ExecContext(ctx, query, aggregateID, aggregateType, eventType, payload, tracePayload)
	return err
}
