# Strategi Redis Cache untuk Flash Sale

Dalam skenario *Flash Sale*, Redis adalah garis pertahanan pertama dan *Source of Truth* untuk stok sebelum pesanan diteruskan secara asinkron ke database.

## 1. Masalah dengan RDBMS (Kenapa Butuh Redis?)

Jika 10.000 user mencoba membeli produk A secara bersamaan, maka akan terjadi 10.000 query:

```sql
UPDATE inventories SET stock = stock - 1 WHERE product_id = 'A' AND stock > 0;
```

Ini menyebabkan **Row-Level Lock Contention**. Database akan tersendat, *query* menumpuk, memori habis, dan sistem *crash*.

## 2. Struktur Data Redis

Proyek ini menggunakan dua pola *key* di Redis:

| Key | Tipe | Keterangan |
| :--- | :--- | :--- |
| `stock:{productID}` | String (Integer) | Jumlah sisa stok produk |
| `reserve_idemp:{eventID}` | String (`"1"`) | *Idempotency key* berbasis event, expire 1 jam |

> **Perbedaan dengan pendekatan *user-based dedup*:** Proyek ini menggunakan **event-based idempotency**. Setiap event dari Kafka memiliki `event_id` unik. Jika event yang sama masuk dua kali (misal: karena *retry*), Lua Script akan mendeteksi key `reserve_idemp:{eventID}` sudah ada dan langsung mengembalikan sukses tanpa mengurangi stok ulang.

## 3. Inisialisasi Stok (*Pre-heating*)

Saat `main.go` dari *Inventory Service* dijalankan, stok awal dimasukkan ke Redis:

```go
rdb.Set(context.Background(), "stock:prod_1", 100, 0)
```

Di production, proses ini dilakukan secara terjadwal (misal H-1 jam sebelum Flash Sale) dengan menyinkronkan stok dari database ke Redis.

## 4. Atomic Deduction dengan Lua Script

Kita **TIDAK BOLEH** melakukan `GET stock`, cek di aplikasi, lalu `SET stock`. Itu akan menyebabkan *race condition*. Logika harus dikirim ke Redis menggunakan **Lua Script**. Redis mengeksekusi Lua Script secara *single-threaded*, sehingga dijamin atomik dan 100% aman dari *race condition*.

### 4.1 `ReserveStockScript` — Potong Stok

Dijalankan saat event reservasi stok masuk dari Kafka.

```lua
local stock_key = KEYS[1]
local idemp_key = KEYS[2]
local amount = tonumber(ARGV[1])

-- Cek Idempotency
if redis.call("EXISTS", idemp_key) == 1 then
    return 1 -- Sudah pernah diproses, return success
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
```

**Return value:**
- `1` → Berhasil (stok dikurangi, ATAU event sudah pernah diproses sebelumnya / idempotent).
- `0` → Gagal (stok habis atau tidak cukup).

**Pemanggilan dari Go:**

```go
stockKey  := fmt.Sprintf("stock:%s", productID)
idempKey  := fmt.Sprintf("reserve_idemp:%s", eventID)
result, _ := reserveLuaScript.Run(ctx, rdb, []string{stockKey, idempKey}, 1).Int()
```

### 4.2 `RefundStockScript` — Kembalikan Stok (Saga Compensation)

Dijalankan saat `OrderCancelledEvent` diterima (misal: pembayaran gagal atau *timeout*). Stok dikembalikan dan *idempotency key* dihapus agar *slot* tersebut bisa digunakan kembali.

```lua
local stock_key = KEYS[1]
local idemp_key = KEYS[2]
local amount = tonumber(ARGV[1])

-- Kembalikan Stok
redis.call("INCRBY", stock_key, amount)
-- Hapus idempotency_key agar user bisa beli lagi kalau mau
redis.call("DEL", idemp_key)

return 1
```

**Pemanggilan dari Go:**

```go
stockKey  := fmt.Sprintf("stock:%s", productID)
idempKey  := fmt.Sprintf("reserve_idemp:%s", eventID)
result, _ := refundLuaScript.Run(ctx, rdb, []string{stockKey, idempKey}, quantity).Int()
```

## 5. Kecepatan dan Ketahanan

- Redis mengeksekusi script di atas dalam orde *microsecond*.
- Satu node Redis mampu menangani ~100.000 operasi per detik. Lebih dari cukup untuk target 10.000 RPS.
- **Failover:** Jika node Redis utama mati, Sentinel/Cluster akan me-*route* ke replika. Namun, bisa ada *data loss* kecil di jeda failover. Untuk kasus ini, *append-only file* (AOF) dengan kebijakan sinkronisasi ketat (`appendfsync everysec` atau `always`) dianjurkan.

## 6. Sinkronisasi Asinkron (Redis → DB)

Setelah Lua Script mengembalikan `1 (SUCCESS)`, *Inventory Service* mencatat perubahan ini ke PostgreSQL melalui *Transactional Outbox Pattern*. Event disisipkan ke tabel `outbox_messages`, lalu di-*relay* ke Kafka oleh *background worker* agar tidak memblokir laju transaksi Redis.
