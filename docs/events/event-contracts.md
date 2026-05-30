# Kontrak Event & Payload

Dokumen ini menjelaskan struktur data event yang dipancarkan ke Kafka oleh masing-masing *microservice*. Definisi *machine-readable* (AsyncAPI) ada di `asyncapi.yaml`.

---

## 1. StockReservedEvent

Dipancarkan oleh **Inventory Service** ketika stok berhasil dikurangi di Redis.

- **Topik:** `flashsale.inventory.events`
- **Tujuan:** Memberitahu *Order Service* untuk membuat pesanan baru di database-nya.
- **Payload:**

| Field | Tipe | Keterangan |
| :--- | :--- | :--- |
| `event_id` | string | ID unik event |
| `user_id` | string | ID pembeli |
| `product_id` | string | ID produk yang dibeli |
| `quantity` | int | Jumlah item |
| `price` | int64 | Harga satuan (Flash Sale) untuk menghitung total |

---

## 2. PaymentCompletedEvent

Dipancarkan oleh **Payment Service** saat pembayaran berhasil diproses.

- **Topik:** `flashsale.payment.events`
- **Tujuan:** Memberitahu *Order Service* untuk mengubah status pesanan menjadi `PAID`.
- **Payload:**

| Field | Tipe | Keterangan |
| :--- | :--- | :--- |
| `event_id` | string | ID unik event |
| `order_id` | string | ID pesanan yang dibayar |
| `amount` | int64 | Jumlah pembayaran |

---

## 3. PaymentFailedEvent

Dipancarkan oleh **Payment Service** saat pembayaran gagal. Pada versi scaffold, kegagalan disimulasikan ketika `amount % 10 == 4` (berakhiran angka 4).

- **Topik:** `flashsale.payment.events`
- **Tujuan:** Memberitahu *Order Service* untuk membatalkan pesanan dan memicu *Saga Compensation* (mengembalikan stok via `OrderCancelledEvent`).
- **Payload:**

| Field | Tipe | Keterangan |
| :--- | :--- | :--- |
| `event_id` | string | ID unik event |
| `order_id` | string | ID pesanan yang gagal dibayar |
| `amount` | int64 | Jumlah pembayaran yang gagal |
| `reason` | string | Alasan kegagalan (misal: `"Payment declined by bank"`) |

---

## 4. OrderCancelledEvent

Dipancarkan oleh **Order Service** ketika pesanan dibatalkan — baik karena pembayaran gagal (`PaymentFailedEvent`) maupun karena *timeout*.

- **Topik:** `flashsale.order.events`
- **Tujuan:** Memicu *Saga Compensation*. *Inventory Service* mendengarkan event ini, mengembalikan stok ke Redis (via `RefundStockScript`), dan melepas reservasi di database.
- **Payload:**

| Field | Tipe | Keterangan |
| :--- | :--- | :--- |
| `event_id` | string | ID unik event |
| `order_id` | string | ID pesanan yang dibatalkan |
| `product_id` | string | ID produk terkait |
| `quantity` | int | Jumlah item yang dikembalikan |
| `reason` | string | Alasan pembatalan |
