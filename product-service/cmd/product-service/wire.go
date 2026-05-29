//go:build wireinject
// +build wireinject

package main

import (
	"github.com/google/wire"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/jmoiron/sqlx"
	"flashsale/product-service/internal/application/usecase"
	"flashsale/product-service/internal/adapter/inbound/grpc"
	"flashsale/product-service/internal/adapter/outbound/postgres"
)

// initApp meng-inject seluruh dependensi dari Repository hingga gRPC Server.
func initApp(db *sqlx.DB, logger log.Logger) (*grpc.ProductServiceServer, error) {
	panic(wire.Build(
		postgres.NewProductRepo,
		usecase.NewListFlashSaleProductsUsecase,
		grpc.NewProductServiceServer,
	))
}
