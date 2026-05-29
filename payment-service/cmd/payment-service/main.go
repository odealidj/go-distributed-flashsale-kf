package main

import (
	"context"
	"flashsale/shared/pkg/telemetry"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"os"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	kratosgrpc "github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	pb "flashsale/proto/payment/v1"
	"flashsale/shared/pkg/outbox"
)

func main() {
	// Init Tracer
	tp, err := telemetry.InitTracer(context.Background(), "payment-service", "localhost:4317")
	if err != nil {
		panic(err)
	}
	defer tp.Shutdown(context.Background())
	logger := log.With(log.NewStdLogger(os.Stdout),
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
		"service.id", "payment-service",
		"service.name", "payment-service",
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

	// 2. Setup Server menggunakan Wire
	paymentServer := InitializePaymentServer(db)

	// 3. Setup gRPC Server Kratos
	grpcServer := kratosgrpc.NewServer(
		kratosgrpc.Address(":9003"),
		kratosgrpc.Logger(logger),
		kratosgrpc.Middleware(tracing.Server()),
	)
	pb.RegisterPaymentServiceServer(grpcServer, paymentServer)

	// 4. Jalankan Outbox Relay Worker di Background
	relay, err := outbox.NewRelayWorker(db, []string{"localhost:9092"}, logger)
	if err == nil {
		go relay.Start(context.Background(), "flashsale.payment.events")
	} else {
		log.Errorf("Failed to start outbox relay: %v", err)
	}

	// 5. Jalankan App
	app := kratos.New(
		kratos.Name("payment-service"),
		kratos.Server(
			grpcServer,
		),
		kratos.Logger(logger),
	)

	if err := app.Run(); err != nil {
		panic(err)
	}
}
