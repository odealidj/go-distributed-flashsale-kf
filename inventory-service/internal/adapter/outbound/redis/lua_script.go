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

// ReserveStock mengeksekusi Lua Script.
// KEYS[1] = "stock:{product_id}" (Menyimpan jumlah stok)
// KEYS[2] = "reserve_idemp:{event_id}" (Mencegah idempotency, double reserve)
// ARGV[1] = 1 (Jumlah yang dikurangi)
//
// Return 1 jika sukses, 0 jika gagal (stok habis atau sudah pernah di-reserve).
var reserveLuaScript = redis.NewScript(`
	local stock_key = KEYS[1]
	local idemp_key = KEYS[2]
	local amount = tonumber(ARGV[1])

	-- Cek Idempotency
	local is_reserved = redis.call("EXISTS", idemp_key)
	if is_reserved == 1 then
		return 0 -- Gagal, sudah pernah di-reserve
	end

	-- Cek Stok
	local current_stock = tonumber(redis.call("GET", stock_key))
	if current_stock == nil or current_stock < amount then
		return 0 -- Gagal, stok tidak cukup
	end

	-- Potong Stok
	redis.call("DECRBY", stock_key, amount)
	-- Tandai Idempotency (set expired misal 1 jam)
	redis.call("SET", idemp_key, "1", "EX", 3600)

	return 1 -- Sukses
`)

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
