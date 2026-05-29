package postgres

import (
	"context"
	"encoding/json"

	"github.com/jmoiron/sqlx"
	"flashsale/payment-service/internal/application/port"
	"flashsale/payment-service/internal/domain/model"
)

type paymentRepository struct {
	db *sqlx.DB
}

func NewPaymentRepository(db *sqlx.DB) port.PaymentRepository {
	return &paymentRepository{db: db}
}

func (r *paymentRepository) SavePaymentAndEmitEvent(ctx context.Context, payment *model.Payment, event *model.PaymentCompletedEvent) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Simpan Payment
	_, err = tx.ExecContext(ctx, `
		INSERT INTO payments (id, order_id, amount, status)
		VALUES ($1, $2, $3, $4)
	`, payment.ID, payment.OrderID, payment.Amount, payment.Status)
	if err != nil {
		return err
	}

	// 2. Simpan Outbox Event
	payloadBytes, _ := json.Marshal(event)
	_, err = tx.ExecContext(ctx, `
		INSERT INTO outbox_messages (aggregate_id, aggregate_type, event_type, payload, status)
		VALUES ($1, $2, $3, $4, 'PENDING')
	`, payment.OrderID, "order", "PaymentCompletedEvent", string(payloadBytes))
	if err != nil {
		return err
	}

	return tx.Commit()
}
