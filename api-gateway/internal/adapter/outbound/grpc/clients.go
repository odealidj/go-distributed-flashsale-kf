package grpc

import (
	"context"

	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/middleware/tracing"

	"flashsale/api-gateway/internal/application/port"
	inventoryv1 "flashsale/proto/inventory/v1"
	paymentv1 "flashsale/proto/payment/v1"
	productv1 "flashsale/proto/product/v1"
)

type grpcClients struct {
	productClient   productv1.ProductServiceClient
	inventoryClient inventoryv1.InventoryServiceClient
	paymentClient   paymentv1.PaymentServiceClient
}

func NewGrpcClients(productEndpoint, inventoryEndpoint, paymentEndpoint string) (port.ProductServiceClient, port.InventoryServiceClient, port.PaymentServiceClient, error) {
	// Koneksi ke Product Service
	connProd, err := grpc.DialInsecure(context.Background(), grpc.WithEndpoint(productEndpoint), grpc.WithMiddleware(tracing.Client()))
	if err != nil {
		return nil, nil, nil, err
	}
	prodClient := productv1.NewProductServiceClient(connProd)

	// Koneksi ke Inventory Service
	connInv, err := grpc.DialInsecure(context.Background(), grpc.WithEndpoint(inventoryEndpoint), grpc.WithMiddleware(tracing.Client()))
	if err != nil {
		return nil, nil, nil, err
	}
	invClient := inventoryv1.NewInventoryServiceClient(connInv)

	// Koneksi ke Payment Service
	connPay, err := grpc.DialInsecure(context.Background(), grpc.WithEndpoint(paymentEndpoint), grpc.WithMiddleware(tracing.Client()))
	if err != nil {
		return nil, nil, nil, err
	}
	payClient := paymentv1.NewPaymentServiceClient(connPay)

	clients := &grpcClients{
		productClient:   prodClient,
		inventoryClient: invClient,
		paymentClient:   payClient,
	}

	return clients, clients, clients, nil
}

func (c *grpcClients) ListFlashSaleProducts(ctx context.Context, page, perPage int32) (*productv1.ListFlashSaleProductsResponse, error) {
	return c.productClient.ListFlashSaleProducts(ctx, &productv1.ListFlashSaleProductsRequest{
		Page:    page,
		PerPage: perPage,
	})
}

func (c *grpcClients) ReserveStock(ctx context.Context, productID, userID, eventID string) (bool, error) {
	resp, err := c.inventoryClient.ReserveStock(ctx, &inventoryv1.ReserveStockRequest{
		ProductId:      productID,
		UserId:         userID,
		IdempotencyKey: eventID,
		Quantity:       1,
	})
	if err != nil {
		return false, err
	}
	return resp.GetSuccess(), nil
}

func (c *grpcClients) ProcessPayment(ctx context.Context, orderID string, amount int64) (bool, error) {
	resp, err := c.paymentClient.ProcessPayment(ctx, &paymentv1.ProcessPaymentRequest{
		OrderId: orderID,
		Amount:  amount,
	})
	if err != nil {
		return false, err
	}
	return resp != nil, nil
}
