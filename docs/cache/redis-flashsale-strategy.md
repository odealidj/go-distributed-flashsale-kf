# Strategi Redis Cache untuk Flash Sale

Dalam skenario *Flash Sale*, Redis adalah garis pertahanan pertama dan alat ukur kebenaran utama (*Source of Truth*) sebelum pesanan diteruskan secara asinkron ke database.

## 1. Masalah dengan RDBMS Relasional (Kenapa butuh Redis?)
Jika 10.000 user mencoba membeli produk A bersamaan, maka akan terjadi 10.000 query:
`UPDATE inventory SET stock = stock - 1 WHERE product_id = 'A' AND stock > 0;`

Ini menyebabkan **Row-Level Lock Contention**. Database akan tersendat, *query* menumpuk, memori habis, dan sistem *crash*.

## 2. Arsitektur Pre-heating

Sebelum jadwal *Flash Sale* dimulai (misalnya H-1 jam), kita melakukan sinkronisasi stok dari DB ke Redis.

**Data Structures di Redis:**
1. `flashsale:stock:{product_id}` -> Tipe Data: *String (Integer)*. Menyimpan jumlah sisa stok.
2. `flashsale:user_bought:{product_id}` -> Tipe Data: *Set*. Menyimpan kumpulan `user_id` yang sudah berhasil mendapat kuota (mencegah user beli > 1 kali).

## 3. Atomic Deduction dengan Lua Script

Kita TIDAK boleh melakukan `GET stock`, cek di aplikasi, lalu `SET stock`. Itu akan menyebabkan *race condition*.
Kita harus mengirim logika ke Redis menggunakan **Lua Script**. Redis mengeksekusi Lua Script secara tunggal (Single-threaded), sehingga dijamin atomik dan 100% aman dari *race condition*.

**Contoh Lua Script `reserve_stock.lua`:**

```lua
local stock_key = KEYS[1]
local user_set_key = KEYS[2]

local qty_requested = tonumber(ARGV[1])
local user_id = ARGV[2]

-- Cek apakah user sudah beli sebelumnya
if redis.call("SISMEMBER", user_set_key, user_id) == 1 then
    return -1 -- Error: User Already Bought
end

-- Ambil sisa stok
local current_stock = tonumber(redis.call("GET", stock_key))
if current_stock == nil then
    return -2 -- Error: Product not found / Sale ended
end

-- Cek apakah stok cukup
if current_stock >= qty_requested then
    -- Kurangi stok
    redis.call("DECRBY", stock_key, qty_requested)
    -- Catat user
    redis.call("SADD", user_set_key, user_id)
    return 1 -- SUCCESS
else
    return 0 -- Error: Out of Stock
end
```

## 4. Kecepatan dan Ketahanan
- Redis mengeksekusi script di atas dalam orde *microsecond*.
- Satu node Redis bisa menangani ~100.000 operasi per detik. Ini lebih dari cukup untuk target 10.000 RPS.
- **Failover:** Jika node Redis utama mati, Sentinel/Cluster akan me-rutekan ke replika. Namun, bisa ada *data loss* kecil di jeda failover. Untuk kasus ini, *append-only file* (AOF) dengan kebijakan sinkronisasi ketat (`appendfsync everysec` atau `always`) dianjurkan.

## 5. Sinkronisasi Asinkron (Redis to DB)
Setelah Redis me-return `1 (SUCCESS)`, Inventory Service harus mencatat perubahan ini ke PostgreSQL. Ini dilakukan secara **asinkron** (via Kafka) agar tidak memblokir laju transaksi Redis.
