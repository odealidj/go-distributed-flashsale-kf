package usecase

import (
	"context"
	"encoding/json"
	"errors"

	"flashsale/inventory-service/internal/application/port"
)

type ReserveStockUsecase struct {
	redisPort  port.RedisPort
	outboxPort port.OutboxPort
}

func NewReserveStockUsecase(redis port.RedisPort, outbox port.OutboxPort) *ReserveStockUsecase {
	return &ReserveStockUsecase{
		redisPort:  redis,
		outboxPort: outbox,
	}
}

// Execute menjalankan Saga penguncian stok.
// 1. Potong di Redis secara atomik (Lua Script).
// 2. Jika sukses, catat event "StockReserved" ke Outbox Postgres.
func (uc *ReserveStockUsecase) Execute(ctx context.Context, productID string, userID string, eventID string) error {
	// 1. Eksekusi Redis Lua Script
	success, err := uc.redisPort.ReserveStock(ctx, productID, eventID)
	if err != nil {
		return err // Kesalahan koneksi Redis atau Internal Error
	}
	if !success {
		return errors.New("stok habis atau event idempotency gagal") // Domain error (HTTP 409)
	}

	// 2. Simpan ke Postgres Outbox (Transactional)
	// Payload JSON untuk Kafka
	payload := map[string]interface{}{
		"event_id":   eventID,
		"product_id": productID,
		"user_id":    userID,
		"status":     "RESERVED",
	}
	payloadBytes, _ := json.Marshal(payload)

	err = uc.outboxPort.InsertOutbox(ctx, eventID, "Order", "StockReservedEvent", payloadBytes)
	if err != nil {
		// PENTING: Dalam arsitektur asli, jika Postgres mati setelah Redis terpotong,
		// stok akan "bocor" (terpotong di Redis tapi event hilang).
		// Oleh karena itu, arsitektur yang lebih ketat akan menggunakan Event Sourcing
		// atau Compensating Transaction. Untuk keperluan Flash Sale ini, kita terima
		// anomali kecil ini (over-reserve lebih baik daripada over-sell).
		return errors.New("gagal menyimpan outbox: " + err.Error())
	}

	return nil
}
