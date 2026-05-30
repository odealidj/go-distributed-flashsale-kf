package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"flashsale/shared/pkg/outbox"
	"flashsale/shared/pkg/telemetry"
	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	kratosgrpc "github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"

	"flashsale/inventory-service/internal/adapter/inbound/kafka"
	redistore "flashsale/inventory-service/internal/adapter/outbound/redis"
	pb "flashsale/proto/inventory/v1"
)

func main() {
	// Construct Jaeger OTLP Endpoint
	jaegerEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if jaegerEndpoint == "" {
		jaegerHost := os.Getenv("JAEGER_HOST")
		if jaegerHost == "" {
			jaegerHost = "localhost"
		}
		jaegerPort := os.Getenv("JAEGER_OTLP_GRPC_PORT")
		if jaegerPort == "" {
			jaegerPort = "14317"
		}
		jaegerEndpoint = jaegerHost + ":" + jaegerPort
	}

	// Init Tracer
	tp, err := telemetry.InitTracer(context.Background(), "inventory-service", jaegerEndpoint)
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

	// 1. Inisialisasi Postgres
	dbDSN := os.Getenv("DATABASE_URL")
	if dbDSN == "" {
		dbHost := os.Getenv("DB_HOST")
		if dbHost == "" {
			dbHost = "localhost"
		}
		dbUser := os.Getenv("POSTGRES_USER")
		if dbUser == "" {
			dbUser = "root"
		}
		dbPassword := os.Getenv("POSTGRES_PASSWORD")
		if dbPassword == "" {
			dbPassword = "rootpassword"
		}
		dbPort := os.Getenv("DB_PORT")
		if dbPort == "" {
			dbPort = "15432"
		}
		dbDSN = fmt.Sprintf("host=%s user=%s password=%s dbname=db_inventory port=%s sslmode=disable", dbHost, dbUser, dbPassword, dbPort)
	}
	db, err := sqlx.Connect("postgres", dbDSN)
	if err != nil {
		log.NewHelper(logger).Warnf("Gagal terhubung ke Postgres (abaikan untuk scaffold): %v", err)
	}

	// 2. Inisialisasi Redis
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisHost := os.Getenv("REDIS_HOST")
		if redisHost == "" {
			redisHost = "localhost"
		}
		redisPort := os.Getenv("REDIS_PORT")
		if redisPort == "" {
			redisPort = "16379"
		}
		redisAddr = fmt.Sprintf("%s:%s", redisHost, redisPort)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
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

	inventoryPort := os.Getenv("INVENTORY_SERVICE_PORT")
	if inventoryPort == "" {
		inventoryPort = "19002"
	}

	// 4. Setup gRPC
	grpcServer := kratosgrpc.NewServer(
		kratosgrpc.Address(":" + inventoryPort),
		kratosgrpc.Logger(logger),
		kratosgrpc.Middleware(tracing.Server()),
	)
	pb.RegisterInventoryServiceServer(grpcServer, inventoryServer)

	// Construct Kafka Brokers
	kafkaBrokersStr := os.Getenv("KAFKA_BROKERS")
	var kafkaBrokers []string
	if kafkaBrokersStr != "" {
		kafkaBrokers = strings.Split(kafkaBrokersStr, ",")
	} else {
		kafkaHost := os.Getenv("KAFKA_HOST")
		if kafkaHost == "" {
			kafkaHost = "localhost"
		}
		kafkaPort := os.Getenv("KAFKA_EXTERNAL_PORT")
		if kafkaPort == "" {
			kafkaPort = "19094"
		}
		kafkaBrokers = []string{fmt.Sprintf("%s:%s", kafkaHost, kafkaPort)}
	}

	// 6. Jalankan Outbox Relay Worker di Background
	relay, err := outbox.NewRelayWorker(db, kafkaBrokers, logger)
	if err == nil {
		go relay.Start(context.Background(), "flashsale.inventory.events")
	} else {
		log.Errorf("Failed to start outbox relay: %v", err)
	}

	// 6.5. Jalankan Kafka Consumer
	redisPort := redistore.NewRedisPort(rdb)
	consumer, err := kafka.NewKafkaConsumer(kafkaBrokers, "inventory-service-group", redisPort, logger)
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
