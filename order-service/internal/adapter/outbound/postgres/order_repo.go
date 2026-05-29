package postgres

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/go-kratos/kratos/v2/log"

	"flashsale/order-service/internal/application/port"
	"flashsale/order-service/internal/domain/model"
)

type orderRepository struct {
	db     *sqlx.DB
	logger *log.Helper
}

func NewOrderRepository(db *sqlx.DB, logger log.Logger) port.OrderRepository {
	return &orderRepository{
		db:     db,
		logger: log.NewHelper(logger),
	}
}

func (r *orderRepository) CreateOrderIdempotent(ctx context.Context, order *model.Order, eventID string) (bool, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	// 1. Cek Idempotency
	var exists bool
	err = tx.GetContext(ctx, &exists, "SELECT EXISTS(SELECT 1 FROM processed_events WHERE event_id=$1)", eventID)
	if err != nil {
		return false, err
	}
	if exists {
		r.logger.Infof("Event %s already processed. Skipping create order.", eventID)
		return false, nil
	}

	// 2. Insert Order
	_, err = tx.ExecContext(ctx, `
		INSERT INTO orders (id, user_id, product_id, quantity, total_amount, status)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, order.ID, order.UserID, order.ProductID, order.Quantity, order.TotalAmount, order.Status)
	if err != nil {
		return false, err
	}

	// 3. Insert ke processed_events
	_, err = tx.ExecContext(ctx, "INSERT INTO processed_events (event_id) VALUES ($1)", eventID)
	if err != nil {
		return false, err
	}

	r.logger.Infof("Order %s created successfully for event %s", order.ID, eventID)
	return true, tx.Commit()
}

func (r *orderRepository) UpdateOrderStatusIdempotent(ctx context.Context, orderID, status, eventID string) (bool, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	// 1. Cek Idempotency
	var exists bool
	err = tx.GetContext(ctx, &exists, "SELECT EXISTS(SELECT 1 FROM processed_events WHERE event_id=$1)", eventID)
	if err != nil {
		return false, err
	}
	if exists {
		r.logger.Infof("Event %s already processed. Skipping update order status.", eventID)
		return false, nil
	}

	// 2. Update Order
	_, err = tx.ExecContext(ctx, "UPDATE orders SET status = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2", status, orderID)
	if err != nil {
		return false, err
	}

	// 3. Insert ke processed_events
	_, err = tx.ExecContext(ctx, "INSERT INTO processed_events (event_id) VALUES ($1)", eventID)
	if err != nil {
		return false, err
	}

	r.logger.Infof("Order %s status updated to %s for event %s", orderID, status, eventID)
	return true, tx.Commit()
}
