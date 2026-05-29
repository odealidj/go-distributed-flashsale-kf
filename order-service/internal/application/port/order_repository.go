package port

import (
	"context"

	"flashsale/order-service/internal/domain/model"
)

type OrderRepository interface {
	// CreateOrderIdempotent menyimpan order baru. Jika eventID sudah ada di processed_events, abaikan (return false tanpa error)
	CreateOrderIdempotent(ctx context.Context, order *model.Order, eventID string) (bool, error)
	
	// UpdateOrderStatusIdempotent mengupdate status order. Jika eventID sudah ada, abaikan.
	UpdateOrderStatusIdempotent(ctx context.Context, orderID, status, eventID string) (bool, error)
}
