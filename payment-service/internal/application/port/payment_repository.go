package port

import (
	"context"

	"flashsale/payment-service/internal/domain/model"
)

type PaymentRepository interface {
	SavePaymentAndEmitEvent(ctx context.Context, payment *model.Payment, event *model.PaymentCompletedEvent) error
}
