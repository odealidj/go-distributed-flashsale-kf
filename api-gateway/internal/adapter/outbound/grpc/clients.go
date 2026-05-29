package grpc

import (
	"context"

	"github.com/go-kratos/kratos/v2/transport/grpc"
	grpc_api "google.golang.org/grpc"

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
	connProd, err := grpc.DialInsecure(context.Background(), grpc.WithEndpoint(productEndpoint))
	if err != nil {
		return nil, nil, err
	}
	prodClient := productv1.NewProductServiceClient(connProd)

	// Koneksi ke Inventory Service
	connInv, err := grpc.DialInsecure(context.Background(), grpc.WithEndpoint(inventoryEndpoint))
	if err != nil {
		return nil, nil, err
	}
	invClient := inventoryv1.NewInventoryServiceClient(connInv)

	// Koneksi ke Payment Service
	connPay, err := grpc.DialInsecure(context.Background(), grpc.WithEndpoint(paymentEndpoint))
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
		ProductId: productID,
		UserId:    userID,
		EventId:   eventID,
	})
	if err != nil {
		return false, err
	}
	return resp.GetData().GetSuccess(), nil
}

func (c *grpcClients) ProcessPayment(ctx context.Context, orderID string, amount int64) (bool, error) {
	resp, err := c.paymentClient.ProcessPayment(ctx, &paymentv1.ProcessPaymentRequest{
		OrderId: orderID,
		Amount:  amount,
	})
	if err != nil {
		return false, err
	}
	return resp.GetData().GetSuccess(), nil
}
