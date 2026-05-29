# Logical Data Model (Flash Sale Microservices)

Dokumen ini mendefinisikan skema tabel *PostgreSQL* untuk masing-masing *microservice*. Karena kita menganut *Database per Service*, tabel-tabel ini terpisah ke dalam 4 *logical database* yang berbeda dan **TIDAK BOLEH** di-*JOIN* secara langsung melalui SQL.

---

## Standar Kolom Audit & Optimistic Locking (Best Practice Production)
Sesuai standar industri skala besar, setiap tabel utama wajib memiliki 4 kolom ini:
1. `created_at` (TIMESTAMP): Waktu *record* dibuat.
2. `updated_at` (TIMESTAMP): Waktu *record* terakhir diubah.
3. `created_by` / `updated_by` (VARCHAR): Berisi **user_id** (jika aksi dipicu oleh pelanggan/admin) ATAU **nama_service** (misal: `system-order-saga` jika diubah otomatis oleh *background job/Kafka*). Ini sangat penting di *production* untuk mengetahui *siapa* atau *sistem apa* yang terakhir menyentuh data tersebut.
4. `version` (INT): Digunakan untuk *Optimistic Concurrency Control*. Setiap ada klausa `UPDATE`, kode kita harus mengecek `WHERE id=? AND version=?`, lalu menaikkan `version = version + 1`. Jika *affected_rows = 0*, berarti ada sistem lain yang mengubah data di detik yang sama, dan transaksi kita digagalkan.

---

## 1. Database: `db_product`

Digunakan oleh *Product Service*.

### Tabel `products`
| Kolom | Tipe Data | Constraint | Keterangan |
| :--- | :--- | :--- | :--- |
| `id` | VARCHAR(50) | PK | Format: `prod_<UUID>` |
| `name` | VARCHAR(255) | NOT NULL | Nama produk |
| `original_price` | BIGINT | NOT NULL | Harga normal |
| `flashsale_price` | BIGINT | NOT NULL | Harga coret / diskon |
| `created_at` | TIMESTAMP | NOT NULL | |
| `updated_at` | TIMESTAMP | NOT NULL | |
| `updated_by` | VARCHAR(50) | | ID Admin yang mengubah |
| `version` | INT | NOT NULL DEFAULT 1| Optimistic lock |

---

## 2. Database: `db_inventory`

Digunakan oleh *Inventory Service*. Sangat kritikal.

### Tabel `inventories`
| Kolom | Tipe Data | Constraint | Keterangan |
| :--- | :--- | :--- | :--- |
| `product_id` | VARCHAR(50) | PK | Merujuk ke ID dari db_product |
| `total_stock` | INT | NOT NULL | Total barang fisik di gudang |
| `available_stock`| INT | NOT NULL, >=0 | Stok yang masih bisa dibeli |
| `reserved_stock` | INT | NOT NULL, >=0 | Stok yang sedang dipegang di dalam *Cart / Pending Order* |
| `created_at` | TIMESTAMP | NOT NULL | |
| `updated_at` | TIMESTAMP | NOT NULL | |
| `updated_by` | VARCHAR(50) | | Misal: `system-kafka-consumer` |
| `version` | INT | NOT NULL DEFAULT 1| Optimistic lock |

*(Catatan: Stok utama hidup di RAM/Redis. Tabel ini adalah *backup* persisten yang disinkronkan secara asinkron dari Redis).*

---

## 3. Database: `db_order`

Digunakan oleh *Order Service*. Menyimpan *state machine* Saga.

### Tabel `orders`
| Kolom | Tipe Data | Constraint | Keterangan |
| :--- | :--- | :--- | :--- |
| `id` | VARCHAR(50) | PK | Format: `ord_<UUID>` |
| `user_id` | VARCHAR(50) | NOT NULL | Pembeli |
| `status` | VARCHAR(20) | NOT NULL | `PENDING_PAYMENT`, `PAID`, `CANCELLED` |
| `total_amount` | BIGINT | NOT NULL | |
| `created_at` | TIMESTAMP | NOT NULL | |
| `updated_at` | TIMESTAMP | NOT NULL | |
| `updated_by` | VARCHAR(50) | | `usr_999` atau `order-saga-timeout` |
| `version` | INT | NOT NULL DEFAULT 1| Optimistic lock |

### Tabel `order_items`
| Kolom | Tipe Data | Constraint | Keterangan |
| :--- | :--- | :--- | :--- |
| `id` | VARCHAR(50) | PK | |
| `order_id` | VARCHAR(50) | FK -> orders.id | |
| `product_id` | VARCHAR(50) | NOT NULL | |
| `qty` | INT | NOT NULL | Saat Flash Sale biasanya bernilai 1 |
| `price` | BIGINT | NOT NULL | Snapshot harga saat transaksi |

---

## 4. Database: `db_payment`

Digunakan oleh *Payment Service*.

### Tabel `payments`
| Kolom | Tipe Data | Constraint | Keterangan |
| :--- | :--- | :--- | :--- |
| `id` | VARCHAR(50) | PK | Format: `pay_<UUID>` |
| `order_id` | VARCHAR(50) | NOT NULL, UNIQUE | |
| `user_id` | VARCHAR(50) | NOT NULL | |
| `amount` | BIGINT | NOT NULL | |
| `status` | VARCHAR(20) | NOT NULL | `PENDING`, `SUCCESS`, `FAILED` |
| `created_at` | TIMESTAMP | NOT NULL | |
| `updated_at` | TIMESTAMP | NOT NULL | |
| `updated_by` | VARCHAR(50) | | Misal: `midtrans-webhook` |
| `version` | INT | NOT NULL DEFAULT 1| Optimistic lock |

---

## 5. Tabel Wajib di Tiap Database: `outbox` & `inbox`

Tabel ini WAJIB ada di `db_inventory`, `db_order`, dan `db_payment` untuk menjamin konsistensi Kafka (*Transactional Outbox Pattern*).

### Tabel `outbox_messages` (Pesan yang akan dikirim ke Kafka)
| Kolom | Tipe Data | Constraint | Keterangan |
| :--- | :--- | :--- | :--- |
| `id` | VARCHAR(50) | PK | UUID Pesan |
| `topic` | VARCHAR(100)| NOT NULL | Tujuan Topik Kafka |
| `payload` | JSONB | NOT NULL | Isi JSON Pesan |
| `status` | VARCHAR(20) | NOT NULL | `PENDING`, `PUBLISHED` |
| `created_at` | TIMESTAMP | NOT NULL | Disisipkan bersamaan dengan transaksi bisnis utama (Terkunci) |

### Tabel `inbox_messages` (Pesan yang masuk dari Kafka / Idempotency Key)
| Kolom | Tipe Data | Constraint | Keterangan |
| :--- | :--- | :--- | :--- |
| `id` | VARCHAR(50) | PK | UUID Event dari Kafka (Menjamin *Idempotency*) |
| `topic` | VARCHAR(100)| NOT NULL | Asal Topik Kafka |
| `status` | VARCHAR(20) | NOT NULL | `PROCESSED` |
| `created_at` | TIMESTAMP | NOT NULL | Waktu pesan berhasil dieksekusi |
