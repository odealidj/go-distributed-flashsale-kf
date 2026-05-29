package grpc

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	pb "flashsale/proto/product/v1"
	"flashsale/product-service/internal/application/usecase"
)

// ProductServiceServer adalah Inbound Adapter gRPC.
type ProductServiceServer struct {
	pb.UnimplementedProductServiceServer
	uc  *usecase.ListFlashSaleProductsUsecase
	log *log.Helper
}

func NewProductServiceServer(uc *usecase.ListFlashSaleProductsUsecase, logger log.Logger) *ProductServiceServer {
	return &ProductServiceServer{
		uc:  uc,
		log: log.NewHelper(logger),
	}
}

func (s *ProductServiceServer) ListFlashSaleProducts(ctx context.Context, req *pb.ListFlashSaleProductsRequest) (*pb.ListFlashSaleProductsResponse, error) {
	products, total, err := s.uc.Execute(ctx, req.GetPage(), req.GetPerPage())
	if err != nil {
		return nil, err
	}

	var pbProducts []*pb.Product
	for _, p := range products {
		pbProducts = append(pbProducts, &pb.Product{
			Id:             p.ID,
			Name:           p.Name,
			OriginalPrice:  p.OriginalPrice,
			FlashSalePrice: p.FlashSalePrice,
		})
	}

	return &pb.ListFlashSaleProductsResponse{
		Meta: &pb.ListFlashSaleProductsResponse_Meta{
			TraceId: "TODO-TRACE", // Akan di-inject middleware nanti
			Message: "success",
			Total:   total,
			Page:    req.GetPage(),
			PerPage: req.GetPerPage(),
		},
		Data: pbProducts,
	}, nil
}
