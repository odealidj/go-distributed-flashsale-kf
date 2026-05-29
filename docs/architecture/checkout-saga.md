# Checkout Saga Design (Choreography)

Di dalam sistem *microservices*, satu transaksi bisnis (checkout) melintasi beberapa *database* (Inventory, Order, Payment). Kita tidak bisa menggunakan ACID Transactions biasa. Kita menggunakan **Saga Pattern**.

Untuk *Flash Sale*, kita menggunakan **Choreography-based Saga** berbasis Kafka. Setiap *service* memancarkan *event*, dan *service* lain bereaksi terhadap *event* tersebut.

## 1. Happy Path (Pesanan Sukses Dibayar)

```mermaid
sequenceDiagram
    participant U as User
    participant O as Order Service
    participant I as Inventory Service
    participant K as Kafka
    participant P as Payment Service

    U->>O: POST /checkout (product_id, qty)
    O->>I: gRPC ReserveStock(product_id, qty)
    Note over I: Redis Lua DECR
    I-->>O: Reserve SUCCESS
    O->>O: DB: Create Order (Status: PENDING)
    O->>K: Emit `OrderCreatedEvent`
    K-->>I: Consume `OrderCreatedEvent`
    I->>I: DB: Persist Reserved Stock
    O-->>U: HTTP 202 Accepted (Order Created)
    
    Note over U, P: Beberapa menit kemudian...
    U->>P: POST /pay (order_id)
    P->>P: Proses mock payment
    P->>K: Emit `PaymentCompletedEvent`
    K-->>O: Consume `PaymentCompletedEvent`
    O->>O: DB: Update Order (Status: PAID)
    K-->>I: Consume `PaymentCompletedEvent`
    I->>I: DB: Update Stock (Reserved -> Deducted)
```

## 2. Compensation Path (Gagal Bayar / Timeout)

Jika pengguna tidak membayar dalam waktu 15 menit, pesanan harus dibatalkan, dan stok di Redis serta Database harus dikembalikan (Rollback).

```mermaid
sequenceDiagram
    participant O as Order Service
    participant I as Inventory Service
    participant K as Kafka

    Note over O: Timer 15 menit berakhir (Cron/Delayed Queue)
    O->>O: Cek status: Masih PENDING?
    O->>O: DB: Update Order (Status: CANCELLED)
    O->>K: Emit `OrderCancelledEvent`
    
    K-->>I: Consume `OrderCancelledEvent`
    Note over I: Mulai Compensation!
    I->>I: Redis Lua INCR (Kembalikan stok cache)
    I->>I: DB: Update Stock (Release Reserved)
```

## 3. Aturan Idempotency & Inbox Pattern
Sangat mungkin Kafka mengirimkan *event* yang sama dua kali (At-Least-Once Delivery).
- Setiap *consumer* harus mencatat `event_id` yang sudah diproses di sebuah tabel `processed_events` (Inbox Pattern).
- Sebelum memproses *event* (misal: `PaymentCompletedEvent`), service harus mengecek apakah `event_id` tersebut sudah ada di tabel `processed_events`. Jika sudah, abaikan *event* tersebut (*return success* ke Kafka agar *offset* maju). Pengecekan dan insert harus dilakukan dalam satu transaksi SQL yang sama bersamaan dengan logika bisnis utama.

## 4. Transactional Outbox Pattern & Worker
Untuk menghindari hilangnya *event* saat mengirim ke Kafka, setiap *publisher* (seperti Inventory atau Payment) menyimpan *event* ke tabel `outbox_messages` terlebih dahulu bersamaan dengan transaksi *database* utama.
- Sebuah **Goroutine Poller Worker** (sebagai pengganti Debezium untuk environment *scaffold* lokal) akan berjalan di *background* setiap *service* produsen.
- Worker ini akan membaca baris di `outbox_messages` dengan status `PENDING`, mengirimkannya ke Kafka menggunakan library `franz-go`, lalu meng-*update* statusnya menjadi `SENT`.
