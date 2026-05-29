package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"flashsale/order-service/internal/adapter/inbound/kafka"
	"flashsale/order-service/internal/adapter/outbound/postgres"
	"flashsale/order-service/internal/application/usecase"
)

func main() {
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
		dbDSN = "host=localhost user=postgres password=postgres dbname=flashsale port=5432 sslmode=disable"
	}
	db, err := sqlx.Connect("postgres", dbDSN)
	if err != nil {
		panic(err)
	}

	// 2. Init Dependencies Manual (tanpa Wire agar cepat untuk worker)
	repo := postgres.NewOrderRepository(db, logger)
	uc := usecase.NewOrderSagaUsecase(repo)
	
	consumer, err := kafka.NewKafkaConsumer([]string{"localhost:9092"}, "order-service-group", uc, logger)
	if err != nil {
		panic(err)
	}

	// 3. Jalankan Kafka Consumer
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go consumer.Start(ctx)

	// 4. Wait for interrupt signal to gracefully shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	
	log.Info("Shutting down Order Service...")
}
