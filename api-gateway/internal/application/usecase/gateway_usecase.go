package usecase

import (
	"context"

	"github.com/google/uuid"
	"flashsale/api-gateway/internal/application/port"
	productv1 "flashsale/proto/product/v1"
)

type GatewayUsecase struct {
	productClient   port.ProductServiceClient
	inventoryClient port.InventoryServiceClient
	paymentClient   port.PaymentServiceClient
}

func NewGatewayUsecase(p port.ProductServiceClient, i port.InventoryServiceClient, pay port.PaymentServiceClient) *GatewayUsecase {
	return &GatewayUsecase{
		productClient:   p,
		inventoryClient: i,
		paymentClient:   pay,
	}
}

func (uc *GatewayUsecase) GetProducts(ctx context.Context, page, perPage int32) (*productv1.ListFlashSaleProductsResponse, error) {
	return uc.productClient.ListFlashSaleProducts(ctx, page, perPage)
}

func (uc *GatewayUsecase) Checkout(ctx context.Context, userID, productID string, idempKey string) (string, bool, error) {
	// 1. Gunakan Idempotency Key dari client atau generate UUID baru jika kosong
	eventID := idempKey
	if eventID == "" {
		eventID = uuid.New().String()
	}

	// 2. Hubungi Inventory Service untuk reservasi stok secara synchronous
	// Jika berhasil, Inventory akan emit event Kafka ke Order Service secara asynchronous
	success, err := uc.inventoryClient.ReserveStock(ctx, productID, userID, eventID)
	
	return eventID, success, err
}

func (uc *GatewayUsecase) ProcessPayment(ctx context.Context, orderID string, amount int64) (bool, error) {
	return uc.paymentClient.ProcessPayment(ctx, orderID, amount)
}
