package main

import (
	"context"
	"flashsale/shared/pkg/telemetry"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"os"

	"flashsale/shared/pkg/outbox"
	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	kratosgrpc "github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"

	"flashsale/inventory-service/internal/adapter/inbound/kafka"
	redistore "flashsale/inventory-service/internal/adapter/outbound/redis"
	pb "flashsale/proto/inventory/v1"
)

func main() {
	// Init Tracer
	tp, err := telemetry.InitTracer(context.Background(), "inventory-service", "localhost:4317")
	if err != nil {
		panic(err)
	}
	defer tp.Shutdown(context.Background())
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
		kratosgrpc.Middleware(tracing.Server()),
	)
	pb.RegisterInventoryServiceServer(grpcServer, inventoryServer)

	// 6. Jalankan Outbox Relay Worker di Background
	relay, err := outbox.NewRelayWorker(db, []string{"localhost:9092"}, logger)
	if err == nil {
		go relay.Start(context.Background(), "flashsale.inventory.events")
	} else {
		log.Errorf("Failed to start outbox relay: %v", err)
	}

	// 6.5. Jalankan Kafka Consumer
	redisPort := redistore.NewRedisPort(rdb)
	consumer, err := kafka.NewKafkaConsumer([]string{"localhost:9092"}, "inventory-service-group", redisPort, logger)
	if err != nil {
		panic(err)
	}
	go consumer.Start(context.Background())

	// 7. Jalankan App
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
