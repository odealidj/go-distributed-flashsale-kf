package main

import (
	"context"
	"os"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	kratoshttp "github.com/go-kratos/kratos/v2/transport/http"

	"flashsale/api-gateway/internal/adapter/inbound/rest"
	"flashsale/api-gateway/internal/adapter/outbound/grpc"
	"flashsale/api-gateway/internal/application/usecase"
	"flashsale/shared/pkg/telemetry"
)

func main() {
	logger := log.With(log.NewStdLogger(os.Stdout),
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
		"service.id", "api-gateway",
		"service.name", "api-gateway",
		"service.version", "v1.0.0",
	)

	// Init Tracer
	tp, err := telemetry.InitTracer(context.Background(), "api-gateway", "localhost:4317")
	if err != nil {
		panic(err)
	}
	defer tp.Shutdown(context.Background())

	productEndpoint := os.Getenv("PRODUCT_SERVICE_ENDPOINT")
	if productEndpoint == "" {
		productEndpoint = "localhost:9001"
	}
	inventoryEndpoint := os.Getenv("INVENTORY_SERVICE_ENDPOINT")
	if inventoryEndpoint == "" {
		inventoryEndpoint = "localhost:9002"
	}
	paymentEndpoint := os.Getenv("PAYMENT_SERVICE_ENDPOINT")
	if paymentEndpoint == "" {
		paymentEndpoint = "localhost:9004"
	}

	prodClient, invClient, payClient, err := grpc.NewGrpcClients(productEndpoint, inventoryEndpoint, paymentEndpoint)
	if err != nil {
		panic(err)
	}

	uc := usecase.NewGatewayUsecase(prodClient, invClient, payClient)

	httpSrv := kratoshttp.NewServer(
		kratoshttp.Address(":8000"),
		kratoshttp.Logger(logger),
		kratoshttp.Middleware(tracing.Server()),
	)
	rest.RegisterHTTPServer(httpSrv, uc, logger)

	app := kratos.New(
		kratos.Name("api-gateway"),
		kratos.Server(httpSrv),
		kratos.Logger(logger),
	)

	if err := app.Run(); err != nil {
		panic(err)
	}
}
