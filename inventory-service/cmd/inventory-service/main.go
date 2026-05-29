package main

import (
	"context"
	"os"

	kratosgrpc "github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"

	pb "flashsale/proto/inventory/v1"
)

func main() {
	logger := log.With(log.NewStdLogger(os.Stdout),
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
		"service.id", "inventory-service",
		"service.name", "inventory-service",
		"service.version", "v1.0.0",
	)

	// 1. Inisialisasi Postgres (Dummy DSN)
	db, err := sqlx.Connect("postgres", "user=root password=rootpassword dbname=flashsale_master sslmode=disable host=localhost port=5432")
	if err != nil {
		log.NewHelper(logger).Warnf("Gagal terhubung ke Postgres (abaikan untuk scaffold): %v", err)
	}

	// 2. Inisialisasi Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.NewHelper(logger).Warnf("Gagal terhubung ke Redis (abaikan untuk scaffold): %v", err)
	} else {
		// Mock Data Stok Awal untuk test
		// Men-set stok "prod_1" menjadi 100
		rdb.Set(context.Background(), "stock:prod_1", 100, 0)
	}

	// 3. Inject Dependensi
	inventoryServer, err := initApp(db, rdb, logger)
	if err != nil {
		panic(err)
	}

	// 4. Setup gRPC
	grpcServer := kratosgrpc.NewServer(
		kratosgrpc.Address(":9002"),
		kratosgrpc.Logger(logger),
	)
	pb.RegisterInventoryServiceServer(grpcServer, inventoryServer)

	// 5. Jalankan App
	app := kratos.New(
		kratos.Name("inventory-service"),
		kratos.Server(
			grpcServer,
		),
		kratos.Logger(logger),
	)

	if err := app.Run(); err != nil {
		panic(err)
	}
}
