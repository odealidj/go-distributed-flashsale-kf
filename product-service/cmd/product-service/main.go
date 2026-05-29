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

	pb "flashsale/proto/product/v1"
)

func main() {
	// Init Tracer
	tp, err := telemetry.InitTracer(context.Background(), "product-service", "localhost:4317")
	if err != nil {
		panic(err)
	}
	defer tp.Shutdown(context.Background())
	logger := log.With(log.NewStdLogger(os.Stdout),
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
		"service.id", "product-service",
		"service.name", "product-service",
		"service.version", "v1.0.0",
	)

	// Inisialisasi Database (Dummy koneksi sementara)
	// Di Production, gunakan env var untuk DSN
	db, err := sqlx.Connect("postgres", "user=root password=rootpassword dbname=flashsale_master sslmode=disable host=localhost port=5432")
	if err != nil {
		log.NewHelper(logger).Warnf("Gagal terhubung ke database (abaikan sementara karena scaffold): %v", err)
		// Kita biarkan jalan menggunakan nil db untuk demo dummy data jika postgres belum siap
	}

	// Inject dependensi menggunakan Wire
	productServer, err := initApp(db, logger)
	if err != nil {
		panic(err)
	}

	// Setup gRPC Server Kratos
	grpcServer := kratosgrpc.NewServer(
		kratosgrpc.Address(":9001"),
		kratosgrpc.Logger(logger),
		kratosgrpc.Middleware(tracing.Server()),
	)
	pb.RegisterProductServiceServer(grpcServer, productServer)

	// Jalankan Kratos Application Lifecycle
	app := kratos.New(
		kratos.Name("product-service"),
		kratos.Server(
			grpcServer,
		),
		kratos.Logger(logger),
	)

	if err := app.Run(); err != nil {
		panic(err)
	}
}
