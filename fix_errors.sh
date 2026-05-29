#!/bin/bash
# 1. Product wire.go
cat << 'WIREE' > product-service/cmd/product-service/wire.go
//go:build wireinject
// +build wireinject

package main

import (
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"github.com/jmoiron/sqlx"

	"flashsale/product-service/internal/adapter/inbound/grpc"
	"flashsale/product-service/internal/adapter/outbound/postgres"
	"flashsale/product-service/internal/application/usecase"
)

func initApp(db *sqlx.DB, logger log.Logger) (*grpc.ProductServer, error) {
	panic(wire.Build(
		postgres.NewProductRepo,
		usecase.NewListFlashSaleProductsUsecase,
		grpc.NewProductServer,
	))
}
WIREE

# 2. Inventory server
cat << 'INVS' > inventory-service/internal/adapter/inbound/grpc/inventory_server.go
package grpc

import (
	"context"

	pb "flashsale/proto/inventory/v1"
	"flashsale/inventory-service/internal/application/usecase"
)

type InventoryServer struct {
	pb.UnimplementedInventoryServiceServer
	usecase *usecase.ReserveStockUsecase
}

func NewInventoryServer(uc *usecase.ReserveStockUsecase) *InventoryServer {
	return &InventoryServer{
		usecase: uc,
	}
}

func (s *InventoryServer) ReserveStock(ctx context.Context, req *pb.ReserveStockRequest) (*pb.ReserveStockResponse, error) {
	eventID, success, err := s.usecase.Execute(ctx, req.GetProductId(), req.GetUserId(), req.GetIdempotencyKey())
	if err != nil {
		return &pb.ReserveStockResponse{
			Success: false,
			EventId: "",
			Message: err.Error(),
		}, nil
	}

	if !success {
		return &pb.ReserveStockResponse{
			Success: false,
			EventId: "",
			Message: "failed to reserve stock",
		}, nil
	}

	return &pb.ReserveStockResponse{
		Success: true,
		EventId: eventID,
		Message: "stock reserved",
	}, nil
}
INVS

# 3. Payment server
cat << 'PAYS' > payment-service/internal/adapter/inbound/grpc/payment_server.go
package grpc

import (
	"context"

	pb "flashsale/proto/payment/v1"
	"flashsale/payment-service/internal/application/usecase"
)

type PaymentServer struct {
	pb.UnimplementedPaymentServiceServer
	usecase *usecase.ProcessPaymentUsecase
}

func NewPaymentServer(uc *usecase.ProcessPaymentUsecase) *PaymentServer {
	return &PaymentServer{
		usecase: uc,
	}
}

func (s *PaymentServer) ProcessPayment(ctx context.Context, req *pb.ProcessPaymentRequest) (*pb.ProcessPaymentResponse, error) {
	payment, err := s.usecase.Execute(ctx, req.GetOrderId(), req.GetUserId(), req.GetAmount())
	if err != nil {
		return nil, err
	}

	return &pb.ProcessPaymentResponse{
		PaymentId:  payment.ID,
		PaymentUrl: payment.PaymentURL,
	}, nil
}
PAYS

# 4. Remove unused imports
sed -i '/"encoding\/json"/d' inventory-service/internal/adapter/outbound/postgres/outbox_repo.go
sed -i '/"encoding\/json"/d' shared/pkg/outbox/relay.go

# Rewire product
cd product-service/cmd/product-service && wire && cd ../../..
