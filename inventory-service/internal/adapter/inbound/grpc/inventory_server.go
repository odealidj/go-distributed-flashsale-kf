package grpc

import (
	"context"

	pb "flashsale/proto/inventory/v1"
	"flashsale/inventory-service/internal/application/usecase"
)

type InventoryServer struct {
	pb.UnimplementedInventoryServiceServer
	usecase *usecase.ReserveStockUsecase
}

func NewInventoryServer(uc *usecase.ReserveStockUsecase) *InventoryServer {
	return &InventoryServer{
		usecase: uc,
	}
}

func (s *InventoryServer) ReserveStock(ctx context.Context, req *pb.ReserveStockRequest) (*pb.ReserveStockResponse, error) {
	err := s.usecase.Execute(ctx, req.GetProductId(), req.GetUserId(), req.GetIdempotencyKey())
	if err != nil {
		return &pb.ReserveStockResponse{
			Success: false,
			EventId: "",
			Message: err.Error(),
		}, nil
	}

	return &pb.ReserveStockResponse{
		Success: true,
		EventId: req.GetIdempotencyKey(),
		Message: "stock reserved",
	}, nil
}
