//go:build wireinject
// +build wireinject

package main

import (
	"github.com/google/wire"
	"github.com/jmoiron/sqlx"

	"flashsale/payment-service/internal/adapter/inbound/grpc"
	"flashsale/payment-service/internal/adapter/outbound/postgres"
	"flashsale/payment-service/internal/application/usecase"
)

func InitializePaymentServer(db *sqlx.DB) *grpc.PaymentServer {
	wire.Build(
		postgres.NewPaymentRepository,
		usecase.NewProcessPaymentUsecase,
		grpc.NewPaymentServer,
	)
	return &grpc.PaymentServer{}
}
