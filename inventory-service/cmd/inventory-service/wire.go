//go:build wireinject
// +build wireinject

package main

import (
	"github.com/google/wire"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	"flashsale/inventory-service/internal/application/usecase"
	"flashsale/inventory-service/internal/adapter/inbound/grpc"
	"flashsale/inventory-service/internal/adapter/outbound/postgres"
	redisAdapter "flashsale/inventory-service/internal/adapter/outbound/redis"
)

func initApp(db *sqlx.DB, rdb *redis.Client, logger log.Logger) (*grpc.InventoryServiceServer, error) {
	panic(wire.Build(
		postgres.NewOutboxRepo,
		redisAdapter.NewRedisPort,
		usecase.NewReserveStockUsecase,
		grpc.NewInventoryServiceServer,
	))
}
