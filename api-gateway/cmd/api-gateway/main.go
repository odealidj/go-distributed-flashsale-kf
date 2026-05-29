package main

import (
	"os"

	kratoshttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"

	"flashsale/api-gateway/internal/adapter/inbound/rest"
	"flashsale/api-gateway/internal/adapter/outbound/grpc"
	"flashsale/api-gateway/internal/application/usecase"
)

func main() {
	logger := log.With(log.NewStdLogger(os.Stdout),
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
		"service.id", "api-gateway",
		"service.name", "api-gateway",
		"service.version", "v1.0.0",
	)

	// 1. Setup gRPC Clients (Akan diarahkan ke docker DNS nantinya)
	// Di local tanpa docker, arahkan ke localhost:9001 dan localhost:9002
	productEndpoint := os.Getenv("PRODUCT_GRPC_ENDPOINT")
	if productEndpoint == "" {
		productEndpoint = "localhost:9001"
	}
	inventoryEndpoint := os.Getenv("INVENTORY_GRPC_ENDPOINT")
	if inventoryEndpoint == "" {
		inventoryEndpoint = "localhost:9002"
	}

	prodClient, invClient, err := grpc.NewGrpcClients(productEndpoint, inventoryEndpoint)
	if err != nil {
		panic(err)
	}

	// 2. Setup Usecase
	uc := usecase.NewGatewayUsecase(prodClient, invClient)

	// 3. Setup HTTP Server
	httpServer := kratoshttp.NewServer(
		kratoshttp.Address(":8080"),
		kratoshttp.Logger(logger),
	)

	rest.RegisterHTTPServer(httpServer, uc, logger)

	// 4. Jalankan App
	app := kratos.New(
		kratos.Name("api-gateway"),
		kratos.Server(
			httpServer,
		),
		kratos.Logger(logger),
	)

	if err := app.Run(); err != nil {
		panic(err)
	}
}
