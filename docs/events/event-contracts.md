# Kontrak Event & Payload

Dokumen ini menjelaskan struktur data yang dipancarkan ke Kafka. Definisi pastinya (Machine Readable) ada di `asyncapi.yaml`.

## 1. StockReservedEvent
Dipancarkan oleh **Inventory Service** ketika stok berhasil dikurangi di Redis.
- **Topik:** `flashsale.inventory.events`
- **Tujuan:** Memberitahu Order Service untuk membuat draft pesanan di database-nya.
- **Payload:** `event_id`, `user_id`, `product_id`, `qty`, `timestamp`.

## 2. OrderCreatedEvent
Dipancarkan oleh **Order Service** setelah pesanan berhasil dicatat di DB.
- **Topik:** `flashsale.order.events`
- **Tujuan:** Memberitahu Payment Service bahwa ada pesanan baru yang menunggu pembayaran, serta memberitahu Inventory Service untuk melakukan *hard lock* pada database relasionalnya.

## 3. PaymentCompletedEvent
Dipancarkan oleh **Payment Service** saat webhook pembayaran sukses diterima.
- **Topik:** `flashsale.payment.events`
- **Tujuan:** Memberitahu Order Service untuk mengubah status pesanan jadi PAID, dan Inventory Service untuk mengubah status reservasi jadi sukses absolut.

## 4. OrderCancelledEvent
Dipancarkan oleh **Order Service** (via Cron/Delayed Job) jika batas waktu 15 menit habis.
- **Topik:** `flashsale.order.events`
- **Tujuan:** Memicu *Saga Compensation*. Inventory Service harus mendengarkan ini, mengembalikan stok ke Redis (INCR), dan merilis status *reserved* di database.
