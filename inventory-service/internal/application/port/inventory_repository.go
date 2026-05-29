package port

import (
	"context"
)

// RedisPort menangani operasi atomik di Redis (Atomic Counter & Idempotency).
type RedisPort interface {
	// ReserveStock menjalankan Lua Script: cek stok > 0, kurangi 1, simpan event_id (idempotency key).
	ReserveStock(ctx context.Context, productID string, eventID string) (bool, error)
	// RefundStock menjalankan Lua Script untuk mengembalikan stok dan menghapus idempotency key.
	RefundStock(ctx context.Context, productID string, eventID string, quantity int) (bool, error)
}

// OutboxPort menangani penyimpanan event ke database Postgres di dalam transaksi.
type OutboxPort interface {
	// InsertOutbox menyimpan payload (misal: JSON StockReservedEvent) ke tabel outbox_messages.
	InsertOutbox(ctx context.Context, aggregateID string, aggregateType string, eventType string, payload []byte) error
}
