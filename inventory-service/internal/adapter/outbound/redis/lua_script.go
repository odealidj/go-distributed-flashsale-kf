package redis

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"flashsale/inventory-service/internal/application/port"
)

type redisPort struct {
	client *redis.Client
}

func NewRedisPort(client *redis.Client) port.RedisPort {
	return &redisPort{client: client}
}

// ReserveStockScript digunakan untuk mengecek stok, mengurangi stok,
// dan menyimpan event_id (idempotency_key) secara atomic dalam 1 operasi Redis.
const ReserveStockScript = `
local stock_key = KEYS[1]
local idemp_key = KEYS[2]
local amount = tonumber(ARGV[1])

-- Cek Idempotency
if redis.call("EXISTS", idemp_key) == 1 then
    return 0 -- Sudah pernah diproses (duplicate), return rejection
end

-- Cek Stok
local current_stock = tonumber(redis.call("GET", stock_key))
if current_stock == nil or current_stock < amount then
    return 0 -- Stok habis atau tidak cukup
end

-- Potong Stok
redis.call("DECRBY", stock_key, amount)
-- Simpan idempotency key (expire 1 jam cukup)
redis.call("SET", idemp_key, "1", "EX", 3600)

return 1
`

// RefundStockScript digunakan untuk mengembalikan stok (saat order dibatalkan)
// dan menghapus idempotency_key secara atomic dalam 1 operasi Redis.
const RefundStockScript = `
local stock_key = KEYS[1]
local idemp_key = KEYS[2]
local amount = tonumber(ARGV[1])

-- Kembalikan Stok
redis.call("INCRBY", stock_key, amount)
-- Hapus idempotency_key agar user bisa beli lagi kalau mau
redis.call("DEL", idemp_key)

return 1
`

var reserveLuaScript = redis.NewScript(ReserveStockScript)

func (r *redisPort) ReserveStock(ctx context.Context, productID string, eventID string) (bool, error) {
	stockKey := fmt.Sprintf("stock:%s", productID)
	idempKey := fmt.Sprintf("reserve_idemp:%s", eventID)

	res, err := reserveLuaScript.Run(ctx, r.client, []string{stockKey, idempKey}, 1).Int()
	if err != nil {
		if err == redis.Nil {
			return false, nil // Script error atau key tidak ada
		}
		return false, err // Koneksi putus
	}

	return res == 1, nil
}

var refundLuaScript = redis.NewScript(RefundStockScript)

func (r *redisPort) RefundStock(ctx context.Context, productID string, eventID string, quantity int) (bool, error) {
	stockKey := fmt.Sprintf("stock:%s", productID)
	idempKey := fmt.Sprintf("reserve_idemp:%s", eventID)

	res, err := refundLuaScript.Run(ctx, r.client, []string{stockKey, idempKey}, quantity).Int()
	if err != nil {
		if err == redis.Nil {
			return false, nil
		}
		return false, err
	}

	return res == 1, nil
}
