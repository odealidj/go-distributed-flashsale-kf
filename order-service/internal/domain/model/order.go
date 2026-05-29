package model

import "time"

type Order struct {
	ID          string
	UserID      string
	ProductID   string
	Quantity    int
	TotalAmount int64
	Status      string // PENDING, PAID, CANCELLED
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type StockReservedEvent struct {
	EventID   string `json:"event_id"`
	UserID    string `json:"user_id"`
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
	Price     int64  `json:"price"` // Untuk simplicity, simpan price agar bisa hitung total
}

type PaymentCompletedEvent struct {
	EventID string `json:"event_id"`
	OrderID string `json:"order_id"`
	Amount  int64  `json:"amount"`
}

type PaymentFailedEvent struct {
	EventID string `json:"event_id"`
	OrderID string `json:"order_id"`
	Amount  int64  `json:"amount"`
	Reason  string `json:"reason"`
}

type OrderCancelledEvent struct {
	EventID   string `json:"event_id"`
	OrderID   string `json:"order_id"`
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
	Reason    string `json:"reason"`
}
