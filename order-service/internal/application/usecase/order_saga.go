package usecase

import (
	"context"
	"github.com/google/uuid"

	"flashsale/order-service/internal/application/port"
	"flashsale/order-service/internal/domain/model"
)

type OrderSagaUsecase struct {
	repo port.OrderRepository
}

func NewOrderSagaUsecase(repo port.OrderRepository) *OrderSagaUsecase {
	return &OrderSagaUsecase{repo: repo}
}

// HandleStockReserved dipanggil saat ada event StockReservedEvent dari Kafka
func (uc *OrderSagaUsecase) HandleStockReserved(ctx context.Context, event *model.StockReservedEvent) error {
	order := &model.Order{
		ID:          uuid.New().String(),
		UserID:      event.UserID,
		ProductID:   event.ProductID,
		Quantity:    event.Quantity,
		TotalAmount: int64(event.Quantity) * event.Price,
		Status:      "PENDING",
	}

	_, err := uc.repo.CreateOrderIdempotent(ctx, order, event.EventID)
	return err
}

// HandlePaymentCompleted dipanggil saat ada event PaymentCompletedEvent dari Kafka
func (uc *OrderSagaUsecase) HandlePaymentCompleted(ctx context.Context, event *model.PaymentCompletedEvent) error {
	_, err := uc.repo.UpdateOrderStatusIdempotent(ctx, event.OrderID, "PAID", event.EventID)
	return err
}
