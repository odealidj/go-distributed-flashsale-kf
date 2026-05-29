#!/bin/bash
for s in api-gateway product-service inventory-service order-service payment-service; do
  sed -i '/import (/a \	"flashsale/shared/pkg/telemetry"\n	"github.com/go-kratos/kratos/v2/middleware/tracing"\n	"go.opentelemetry.io/otel"\n	"go.opentelemetry.io/otel/trace"' $s/cmd/$s/main.go
  sed -i '/func main() {/a \	// Init Tracer\n	tp, err := telemetry.InitTracer(context.Background(), "'$s'", "localhost:4317")\n	if err != nil {\n		panic(err)\n	}\n	defer tp.Shutdown(context.Background())' $s/cmd/$s/main.go
done

# Add Server middlewares
for s in product-service inventory-service payment-service; do
  sed -i 's/kratosgrpc.Logger(logger),/kratosgrpc.Logger(logger),\n		kratosgrpc.Middleware(tracing.Server()),/g' $s/cmd/$s/main.go
done

# Api gateway HTTP middleware
sed -i 's/kratoshttp.Logger(logger),/kratoshttp.Logger(logger),\n		kratoshttp.Middleware(tracing.Server()),/g' api-gateway/cmd/api-gateway/main.go

# Api gateway gRPC client middleware
sed -i 's/grpc.WithEndpoint(productEndpoint)/grpc.WithEndpoint(productEndpoint), grpc.WithMiddleware(tracing.Client())/g' api-gateway/internal/adapter/outbound/grpc/clients.go
sed -i 's/grpc.WithEndpoint(inventoryEndpoint)/grpc.WithEndpoint(inventoryEndpoint), grpc.WithMiddleware(tracing.Client())/g' api-gateway/internal/adapter/outbound/grpc/clients.go
sed -i 's/grpc.WithEndpoint(paymentEndpoint)/grpc.WithEndpoint(paymentEndpoint), grpc.WithMiddleware(tracing.Client())/g' api-gateway/internal/adapter/outbound/grpc/clients.go

# Add import trace for api-gateway handler
sed -i '/import (/a \	"go.opentelemetry.io/otel/trace"' api-gateway/internal/adapter/inbound/rest/handler.go
