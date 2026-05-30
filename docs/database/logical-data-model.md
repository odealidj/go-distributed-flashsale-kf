# Logical Data Model (Flash Sale — Scaffold)

Dokumen ini mendefinisikan skema tabel *PostgreSQL* untuk proyek *scaffold/demo* Flash Sale.

> **Catatan:** Pada versi scaffold ini, semua tabel berada dalam **satu database tunggal** bernama `flashsale`. File `init.sql` di-*mount* langsung ke kontainer PostgreSQL saat `docker-compose up`. Di arsitektur *production* yang sesungguhnya, setiap *microservice* idealnya memiliki database terpisah (*Database per Service*) dan tabel-tabel **TIDAK BOLEH** di-*JOIN* langsung via SQL lintas *service*.

---

## Standar Kolom Audit & Optimistic Locking (Best Practice Production)

Sesuai standar industri skala besar, setiap tabel utama sebaiknya memiliki kolom audit berikut:

1. `created_at` (TIMESTAMPTZ): Waktu *record* dibuat.
2. `updated_at` (TIMESTAMPTZ): Waktu *record* terakhir diubah.
3. `created_by` / `updated_by` (VARCHAR): Berisi **user_id** (jika aksi dipicu oleh pelanggan/admin) ATAU **nama_service** (misal: `system-kafka-consumer`).
4. `version` (INT): Digunakan untuk *Optimistic Concurrency Control*. Setiap `UPDATE` harus mengecek `WHERE id=? AND version=?`, lalu menaikkan `version = version + 1`.

> **Catatan Scaffold:** Versi scaffold ini menggunakan pola audit yang disederhanakan. Tidak semua tabel memiliki keempat kolom di atas — hanya yang relevan saja yang diterapkan (misal: tabel `payments` hanya punya `created_at`). Di production, lengkapi sesuai kebutuhan.

---

## 1. Tabel `products`

Menyimpan data produk yang dijual pada Flash Sale.

| Kolom | Tipe Data | Constraint | Keterangan |
| :--- | :--- | :--- | :--- |
| `id` | VARCHAR(50) | PK | Format: `prod_<id>` |
| `name` | VARCHAR(255) | NOT NULL | Nama produk |
| `original_price` | BIGINT | NOT NULL | Harga normal |
| `flash_sale_price` | BIGINT | NOT NULL | Harga Flash Sale (diskon) |
| `created_at` | TIMESTAMPTZ | DEFAULT CURRENT_TIMESTAMP | |
| `updated_at` | TIMESTAMPTZ | DEFAULT CURRENT_TIMESTAMP | |
| `updated_by` | VARCHAR(100) | | ID admin/sistem yang mengubah |
| `version` | INTEGER | DEFAULT 1 | Optimistic lock |

---

## 2. Tabel `inventories`

Menyimpan data stok produk. Stok utama saat Flash Sale hidup di Redis; tabel ini berfungsi sebagai *backup* persisten.

| Kolom | Tipe Data | Constraint | Keterangan |
| :--- | :--- | :--- | :--- |
| `product_id` | VARCHAR(50) | PK | Merujuk ke `products.id` |
| `stock` | BIGINT | NOT NULL | Jumlah stok tersedia |
| `updated_at` | TIMESTAMPTZ | DEFAULT CURRENT_TIMESTAMP | |
| `updated_by` | VARCHAR(100) | | Misal: `system` |
| `version` | INTEGER | DEFAULT 1 | Optimistic lock |

---

## 3. Tabel `orders`

Menyimpan pesanan yang dibuat oleh *Order Service* melalui alur Saga.

| Kolom | Tipe Data | Constraint | Keterangan |
| :--- | :--- | :--- | :--- |
| `id` | VARCHAR(50) | PK | Format: UUID |
| `user_id` | VARCHAR(50) | NOT NULL | ID pembeli |
| `product_id` | VARCHAR(50) | NOT NULL | ID produk yang dibeli |
| `quantity` | INTEGER | NOT NULL | Jumlah item |
| `total_amount` | BIGINT | NOT NULL | Total harga |
| `status` | VARCHAR(50) | NOT NULL, DEFAULT `'PENDING'` | `PENDING`, `PAID`, `CANCELLED` |
| `created_at` | TIMESTAMPTZ | DEFAULT CURRENT_TIMESTAMP | |
| `updated_at` | TIMESTAMPTZ | DEFAULT CURRENT_TIMESTAMP | |

---

## 4. Tabel `payments`

Menyimpan hasil pembayaran yang diproses oleh *Payment Service*.

| Kolom | Tipe Data | Constraint | Keterangan |
| :--- | :--- | :--- | :--- |
| `id` | VARCHAR(50) | PK | Format: UUID |
| `order_id` | VARCHAR(50) | NOT NULL | ID pesanan terkait |
| `amount` | BIGINT | NOT NULL | Jumlah pembayaran |
| `status` | VARCHAR(50) | NOT NULL, DEFAULT `'SUCCESS'` | `SUCCESS`, `FAILED` |
| `created_at` | TIMESTAMPTZ | DEFAULT CURRENT_TIMESTAMP | |

---

## 5. Tabel `outbox_messages`

Digunakan oleh *Transactional Outbox Pattern*. Pesan disisipkan bersamaan dengan transaksi bisnis utama, lalu di-*relay* ke Kafka oleh *background worker*.

| Kolom | Tipe Data | Constraint | Keterangan |
| :--- | :--- | :--- | :--- |
| `id` | SERIAL | PK | Auto-increment |
| `aggregate_id` | VARCHAR(255) | NOT NULL | ID entitas terkait (misal: order ID) |
| `aggregate_type` | VARCHAR(255) | NOT NULL | Tipe entitas (misal: `Order`, `Payment`) |
| `event_type` | VARCHAR(255) | NOT NULL | Nama event (misal: `StockReservedEvent`) |
| `payload` | JSONB | NOT NULL | Isi JSON event |
| `trace_payload` | VARCHAR(512) | | Data *tracing* (OpenTelemetry) |
| `status` | VARCHAR(50) | NOT NULL, DEFAULT `'PENDING'` | `PENDING`, `PUBLISHED` |
| `created_at` | TIMESTAMPTZ | DEFAULT CURRENT_TIMESTAMP | Waktu pesan disisipkan |

---

## 6. Tabel `processed_events`

Digunakan sebagai tabel idempotency. Setiap *event* dari Kafka yang berhasil diproses dicatat di sini agar tidak diproses ulang.

| Kolom | Tipe Data | Constraint | Keterangan |
| :--- | :--- | :--- | :--- |
| `event_id` | VARCHAR(255) | PK | ID unik event dari Kafka |
| `processed_at` | TIMESTAMPTZ | DEFAULT CURRENT_TIMESTAMP | Waktu event berhasil diproses |

---

## 7. Data Seed Awal

File `init.sql` menyisipkan data *dummy* untuk keperluan demo/testing:

### Produk (`products`)

| `id` | `name` | `original_price` | `flash_sale_price` | `updated_by` |
| :--- | :--- | ---: | ---: | :--- |
| `prod_1` | Sepatu Lari X | 500.000 | 150.000 | `system` |
| `prod_2` | Tas Ransel Y | 300.000 | 99.000 | `system` |

### Inventori (`inventories`)

| `product_id` | `stock` | `updated_by` |
| :--- | ---: | :--- |
| `prod_1` | 100 | `system` |
| `prod_2` | 50 | `system` |

Insert menggunakan `ON CONFLICT (id/product_id) DO NOTHING` agar aman dijalankan berulang kali.
