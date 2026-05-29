package usecase

import (
	"context"
	"github.com/google/uuid"

	"flashsale/payment-service/internal/application/port"
	"flashsale/payment-service/internal/domain/model"
)

type ProcessPaymentUsecase struct {
	repo port.PaymentRepository
}

func NewProcessPaymentUsecase(repo port.PaymentRepository) *ProcessPaymentUsecase {
	return &ProcessPaymentUsecase{repo: repo}
}

func (uc *ProcessPaymentUsecase) Execute(ctx context.Context, orderID string, amount int64) (bool, error) {
	payment := &model.Payment{
		ID:      uuid.New().String(),
		OrderID: orderID,
		Amount:  amount,
		Status:  "SUCCESS", // Scaffold: asumsikan selalu sukses
	}

	event := &model.PaymentCompletedEvent{
		EventID: uuid.New().String(),
		OrderID: payment.OrderID,
		Amount:  payment.Amount,
	}

	// Simpan ke DB dan catat ke Outbox secara atomik
	err := uc.repo.SavePaymentAndEmitEvent(ctx, payment, event)
	if err != nil {
		return false, err
	}

	return true, nil
}
