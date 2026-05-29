package grpc

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	pb "flashsale/proto/inventory/v1"
	"flashsale/inventory-service/internal/application/usecase"
)

type InventoryServiceServer struct {
	pb.UnimplementedInventoryServiceServer
	uc  *usecase.ReserveStockUsecase
	log *log.Helper
}

func NewInventoryServiceServer(uc *usecase.ReserveStockUsecase, logger log.Logger) *InventoryServiceServer {
	return &InventoryServiceServer{
		uc:  uc,
		log: log.NewHelper(logger),
	}
}

func (s *InventoryServiceServer) ReserveStock(ctx context.Context, req *pb.ReserveStockRequest) (*pb.ReserveStockResponse, error) {
	err := s.uc.Execute(ctx, req.GetProductId(), req.GetUserId(), req.GetEventId())
	if err != nil {
		// Asumsi error disini adalah error stok habis (di prod butuh pemetaan grpc.code)
		return &pb.ReserveStockResponse{
			Meta: &pb.ReserveStockResponse_Meta{
				TraceId: "TODO-TRACE",
				Message: "failed: " + err.Error(),
			},
			Data: &pb.ReserveStockResponse_Data{
				Success: false,
			},
		}, nil
	}

	return &pb.ReserveStockResponse{
		Meta: &pb.ReserveStockResponse_Meta{
			TraceId: "TODO-TRACE",
			Message: "success",
		},
		Data: &pb.ReserveStockResponse_Data{
			Success: true,
		},
	}, nil
}
