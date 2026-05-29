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
