package grpc

import (
	"context"

	pb "flashsale/proto/product/v1"
	"flashsale/product-service/internal/application/usecase"
)

type ProductServer struct {
	pb.UnimplementedProductServiceServer
	usecase *usecase.ListFlashSaleProductsUsecase
}

func NewProductServer(uc *usecase.ListFlashSaleProductsUsecase) *ProductServer {
	return &ProductServer{
		usecase: uc,
	}
}

func (s *ProductServer) ListFlashSaleProducts(ctx context.Context, req *pb.ListFlashSaleProductsRequest) (*pb.ListFlashSaleProductsResponse, error) {
	products, total, err := s.usecase.Execute(ctx, req.Page, req.PerPage)
	if err != nil {
		return nil, err
	}

	var pbProducts []*pb.ProductItem
	for _, p := range products {
		pbProducts = append(pbProducts, &pb.ProductItem{
			Id:             p.ID,
			Name:           p.Name,
			OriginalPrice:  p.OriginalPrice,
			FlashsalePrice: p.FlashSalePrice,
		})
	}

	return &pb.ListFlashSaleProductsResponse{
		Products:   pbProducts,
		TotalItems: total,
	}, nil
}
