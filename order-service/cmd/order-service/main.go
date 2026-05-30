package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	_ "go.uber.org/automaxprocs"

	"flashsale/shared/pkg/telemetry"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"flashsale/order-service/internal/adapter/inbound/kafka"
	"flashsale/order-service/internal/adapter/outbound/postgres"
	"flashsale/order-service/internal/application/usecase"
	"flashsale/order-service/internal/application/worker"
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
	tp, err := telemetry.InitTracer(context.Background(), "order-service", jaegerEndpoint)
	if err != nil {
		panic(err)
	}
	defer tp.Shutdown(context.Background())
	logger := log.With(log.NewStdLogger(os.Stdout),
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
		"service.id", "order-service",
		"service.name", "order-service",
		"service.version", "v1.0.0",
	)

	// 1. Setup Postgres
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
		dbDSN = fmt.Sprintf("host=%s user=%s password=%s dbname=db_order port=%s sslmode=disable", dbHost, dbUser, dbPassword, dbPort)
	}
	db, err := sqlx.Connect("postgres", dbDSN)
	if err != nil {
		panic(err)
	}
	// Konfigurasi TCP Connection Pool Postgres
	db.SetMaxOpenConns(100)
	db.SetMaxIdleConns(20)
	db.SetConnMaxLifetime(30 * time.Minute)

	// 2. Init Dependencies Manual (tanpa Wire agar cepat untuk worker)
	repo := postgres.NewOrderRepository(db, logger)
	uc := usecase.NewOrderSagaUsecase(repo)

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

	consumer, err := kafka.NewKafkaConsumer(kafkaBrokers, "order-service-group", uc, logger)
	if err != nil {
		panic(err)
	}

	// 3. Jalankan Kafka Consumer & Timeout Worker
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	timeoutWorker := worker.NewTimeoutWorker(db, repo, logger)

	go consumer.Start(ctx)
	go timeoutWorker.Start(ctx)

	// 4. Wait for interrupt signal to gracefully shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	log.Info("Shutting down Order Service...")
}
