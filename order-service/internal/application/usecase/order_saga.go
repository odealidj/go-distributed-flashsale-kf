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

// HandlePaymentFailed dipanggil saat ada event PaymentFailedEvent dari Kafka
func (uc *OrderSagaUsecase) HandlePaymentFailed(ctx context.Context, event *model.PaymentFailedEvent) error {
	order, err := uc.repo.GetOrder(ctx, event.OrderID)
	if err != nil {
		return err
	}

	cancelEvent := &model.OrderCancelledEvent{
		EventID:   event.EventID, // Gunakan eventID dari payment failed sebagai idempotency key
		OrderID:   order.ID,
		ProductID: order.ProductID,
		Quantity:  order.Quantity,
		Reason:    event.Reason,
	}

	return uc.repo.CancelOrderAndEmitEvent(ctx, order, cancelEvent)
}
