# Checkout Saga Design (Choreography)

Di dalam sistem *microservices*, satu transaksi bisnis (checkout) melintasi beberapa *service* (Inventory, Order, Payment). Kita tidak bisa menggunakan ACID Transactions biasa. Kita menggunakan **Saga Pattern**.

Untuk *Flash Sale*, kita menggunakan **Choreography-based Saga** berbasis Kafka. Setiap *service* memancarkan *event*, dan *service* lain bereaksi terhadap *event* tersebut.

## 1. Happy Path (Pesanan Sukses Dibayar)

```mermaid
sequenceDiagram
    participant U as User
    participant GW as API Gateway
    participant I as Inventory Service
    participant K as Kafka
    participant O as Order Service
    participant P as Payment Service
    U->>GW: POST /api/v1/checkout (product_id, qty)
    GW->>I: gRPC ReserveStock(product_id, user_id, event_id)
    Note over I: Redis Lua Script (DECRBY + SET idemp_key)
    I-->>GW: Reserve SUCCESS
    I->>I: DB: Insert Outbox (StockReservedEvent)
    GW-->>U: HTTP 200 (Reserve Success, event_id)
    Note over I: Outbox Relay Worker mempublikasikan ke Kafka
    I->>K: Publish StockReservedEvent
    K-->>O: Consume StockReservedEvent
    O->>O: DB: Create Order (Status: PENDING) + Insert processed_events
    Note over U, P: Beberapa menit kemudian...
    U->>GW: POST /api/v1/pay (order_id, amount)
    GW->>P: gRPC ProcessPayment(order_id, amount)
    P->>P: DB: Insert Payment (SUCCESS) + Insert Outbox (PaymentCompletedEvent)
    P-->>GW: Payment Success
    Note over P: Outbox Relay Worker mempublikasikan ke Kafka
    P->>K: Publish PaymentCompletedEvent
    K-->>O: Consume PaymentCompletedEvent
    O->>O: DB: Update Order (Status: PAID) + Insert processed_events
```

## 2. Compensation Path: Pembayaran Gagal

Jika pembayaran ditolak (simulasi: `amount % 10 == 4`), Payment Service menerbitkan `PaymentFailedEvent`.

```mermaid
sequenceDiagram
    participant U as User
    participant GW as API Gateway
    participant P as Payment Service
    participant K as Kafka
    participant O as Order Service
    participant I as Inventory Service
    U->>GW: POST /api/v1/pay (order_id, amount berakhiran 4)
    GW->>P: gRPC ProcessPayment
    P->>P: DB: Insert Payment (FAILED) + Insert Outbox (PaymentFailedEvent)
    P-->>GW: Payment Failed
    Note over P: Outbox Relay Worker
    P->>K: Publish PaymentFailedEvent
    K-->>O: Consume PaymentFailedEvent
    O->>O: DB: Get Order, Set CANCELLED + Insert Outbox (OrderCancelledEvent)
    Note over O: Outbox / Timeout Worker
    O->>K: Publish OrderCancelledEvent
    K-->>I: Consume OrderCancelledEvent
    Note over I: Saga Compensation!
    I->>I: Redis Lua RefundStock (INCRBY + DEL idemp_key)
```

## 3. Compensation Path: Timeout (15 Menit)

Jika pengguna tidak membayar dalam waktu 15 menit, pesanan harus dibatalkan dan stok dikembalikan.

```mermaid
sequenceDiagram
    participant O as Order Service
    participant K as Kafka
    participant I as Inventory Service
    Note over O: TimeoutWorker (ticker 30 detik)
    O->>O: Query: PENDING + created_at < 15 menit (FOR UPDATE SKIP LOCKED)
    O->>O: DB: Update Order (Status: CANCELLED) + Insert Outbox (OrderCancelledEvent)
    Note over O: Outbox Relay Worker
    O->>K: Publish OrderCancelledEvent
    K-->>I: Consume OrderCancelledEvent
    Note over I: Saga Compensation!
    I->>I: Redis Lua RefundStock (INCRBY + DEL idemp_key)
```

## 4. Aturan Idempotency (processed_events Table)
Sangat mungkin Kafka mengirimkan *event* yang sama dua kali (At-Least-Once Delivery).
- Setiap *consumer* harus mencatat `event_id` yang sudah diproses di tabel `processed_events`.
- Sebelum memproses *event*, service mengecek apakah `event_id` sudah ada. Jika sudah, abaikan (return success agar Kafka offset maju).
- Pengecekan dan insert dilakukan dalam satu transaksi SQL bersamaan dengan logika bisnis utama.

## 5. Transactional Outbox Pattern & Goroutine Poller Worker
Untuk menghindari hilangnya *event* saat mengirim ke Kafka, setiap *publisher* menyimpan *event* ke tabel `outbox_messages` terlebih dahulu bersamaan dengan transaksi *database* utama.
- Sebuah **Goroutine Poller Worker** berjalan di *background* setiap *service* produsen (Inventory Service dan Payment Service).
- Worker ini membaca baris `outbox_messages` dengan status `PENDING` menggunakan `FOR UPDATE SKIP LOCKED`, mengirimkannya ke Kafka via `franz-go`, lalu meng-*update* statusnya menjadi `SENT`. Jika gagal setelah 5 retry, statusnya menjadi `FAILED`.
