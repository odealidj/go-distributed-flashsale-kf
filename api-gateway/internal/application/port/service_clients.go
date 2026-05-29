package port

import (
	"context"
	productv1 "flashsale/proto/product/v1"
)

type ProductServiceClient interface {
	ListFlashSaleProducts(ctx context.Context, page, perPage int32) (*productv1.ListFlashSaleProductsResponse, error)
}

type InventoryServiceClient interface {
	ReserveStock(ctx context.Context, productID, userID, eventID string) (bool, error)
}
