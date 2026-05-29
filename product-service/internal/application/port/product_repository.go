package port

import (
	"context"
	"flashsale/product-service/internal/domain/model"
)

// ProductRepository adalah Outbound Port.
// Usecase akan menggunakan interface ini tanpa peduli apakah ini ditenagai oleh Postgres, Redis, atau Mock.
type ProductRepository interface {
	ListFlashSaleProducts(ctx context.Context, page int32, perPage int32) ([]*model.Product, int32, error)
}
