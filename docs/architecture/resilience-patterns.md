# Strategi Resilience — Flash Sale System

Dokumen ini menjelaskan semua pola ketahanan sistem (*resilience patterns*) yang diterapkan
dan alasan desain di balik setiap keputusan.

## 1. Peta Komponen dan Risiko Kegagalan

```
User → Nginx → API Gateway ──gRPC──→ Inventory Service → Redis + Postgres Outbox
                                   ↘ Product Service   → Postgres
                                   ↘ Payment Service   → Postgres Outbox
                          Outbox Relay Worker → Kafka
                                   ↓
                            Order Service (Consumer) → Postgres Inbox
```

| Titik Kegagalan | Dampak Tanpa Resilience | Pola yang Diterapkan |
|----------------|------------------------|---------------------|
| Inventory/Payment Service down | CB tidak ada → goroutine pile up di GW | **Circuit Breaker** |
| gRPC call hang | Goroutine leak di API Gateway | **Timeout per-call** |
| Koneksi gRPC mati diam-diam | Request ke koneksi mati → timeout lambat | **Keepalive** |
| Kafka sementara down | Event dari outbox hilang | **Retry + status FAILED** |
| Kafka consumer crash setelah proses | Event diproses ulang → duplikat | **Inbox Pattern** |
| Event consumer gagal permanen | Event drop | **Dead Letter Queue** |
| Banyak koneksi DB serentak | `too many connections` PostgreSQL | **Connection Pool Limit** |
| Banyak request HTTP serentak | API Gateway kewalahan | **Rate Limiting (Nginx)** |

---

## 2. Circuit Breaker (API Gateway)

**Library:** `github.com/sony/gobreaker`  
**File:** `shared/pkg/resilience/circuit_breaker.go`  
**Penggunaan:** `api-gateway/internal/adapter/outbound/grpc/clients.go`

### State Machine

```
           failure_ratio >= 50%        timeout (5s) berlalu
CLOSED  ──────────────────────→  OPEN  ──────────────────→  HALF-OPEN
  ↑                                                              │
  │ semua probe sukses                                           │
  └──────────────────────────────────────────────────────────────┘
```

### Konfigurasi Default

```go
CircuitBreakerConfig{
    MaxRequests:  5,              // max request saat half-open
    Interval:     10 * time.Second, // periode evaluasi
    Timeout:      5 * time.Second,  // lama state OPEN sebelum half-open
    FailureRatio: 0.5,            // 50% failure rate = trip
    MinRequests:  10,             // minimum sample sebelum dievaluasi
}
```

### Isolasi Per-Service

Setiap service downstream memiliki CB **sendiri**. Kegagalan `inventory-service`
TIDAK menutup CB `payment-service` atau `product-service`.

---

## 3. Timeout Per-Call gRPC

**Nilai:** 3 detik  
**Implementasi:** `context.WithTimeout(ctx, 3*time.Second)` di setiap method gRPC client.

### Kenapa 3 Detik?
- Lebih kecil dari timeout upstream Nginx (biasanya 10-60 detik)
- Cukup untuk query DB berat sekalipun
- Memberikan feedback cepat ke user daripada menunggu 60 detik

### Keepalive gRPC
```go
keepalive.ClientParameters{
    Time:                10 * time.Second,
    Timeout:             5 * time.Second,
    PermitWithoutStream: true,
}
```
Ping dikirim setiap 10 detik meski tidak ada request aktif, sehingga koneksi mati
terdeteksi dalam 15 detik (bukan baru terdeteksi saat request berikutnya timeout).

---

## 4. Retry + Exponential Backoff + Jitter

**File:** `shared/pkg/resilience/retry.go`

### Rumus Jeda

```
jeda = min(InitialInterval × Multiplier^(attempt-1), MaxInterval) ± 30% jitter
```

Contoh dengan `InitialInterval=200ms, Multiplier=2, MaxInterval=10s`:
- Attempt 1 → gagal → jeda ~200ms
- Attempt 2 → gagal → jeda ~400ms  
- Attempt 3 → gagal → jeda ~800ms
- ...dst

### Mengapa Jitter Penting?

Tanpa jitter: semua goroutine yang gagal bersamaan retry pada interval yang **sama persis**
→ thundering herd saat recovery. Jitter ±30% menyebar retry secara alami.

### Operasi yang TIDAK Di-Retry

| Operasi | Alasan |
|---------|--------|
| `ReserveStock` gRPC | Non-idempoten. Retry dengan event_id berbeda = pemotongan stok baru. Idempotency dijaga Redis Lua Script lewat `idempotency_key`. |
| Kafka payload parsing error | Permanent error. Payload corrupt tidak akan sembuh. Langsung ke DLQ. |

---

## 5. Outbox Relay Worker — Retry + Status FAILED

**File:** `shared/pkg/outbox/relay.go`

### Alur

```
Poll PENDING (FOR UPDATE SKIP LOCKED)
  → Publish ke Kafka (retry 5x, backoff 200ms→10s)
    ✅ Sukses → UPDATE status = 'SENT'
    ❌ Gagal semua retry → UPDATE status = 'FAILED'
```

**Status FAILED** memastikan event tidak hilang dan bisa dipantau via query SQL:
```sql
SELECT * FROM outbox_messages WHERE status = 'FAILED';
```

**RequiredAcks:** `AllISRAcks()` — Kafka hanya mengkonfirmasi penerimaan setelah
semua in-sync replica menyimpan pesan (durabilitas tinggi).

---

## 6. Kafka Consumer — Manual Commit + Dead Letter Queue

**File:** `order-service/internal/adapter/inbound/kafka/consumer.go`

### Alur Pemrosesan

```
Poll record
  → Extract traceparent dari header
  → Retry 3x process (backoff 500ms→5s)
    ✅ Sukses → CommitOffset
    ❌ Gagal semua retry → Kirim ke DLQ topic → CommitOffset
```

### Manual Commit

`DisableAutoCommit()` memastikan offset hanya di-commit setelah pemrosesan selesai.
Jika service crash **sebelum** commit, Kafka akan redelivery. Idempotency di sisi
consumer (tabel `processed_events` / `inbox_messages`) mencegah pemrosesan ganda.

### Dead Letter Queue

**Topic:** `flashsale.order.dlq`

Setiap pesan DLQ menyertakan header:
- `dlq.original.topic` — topic asal
- `dlq.error` — pesan error
- `dlq.timestamp` — waktu kegagalan terakhir

Pesan DLQ dapat di-replay ke topic asal setelah penyebab masalah diperbaiki.

---

## 7. Database Connection Pool

**File:** `shared/pkg/database/postgres.go`

| Parameter | Nilai | Alasan |
|-----------|-------|--------|
| `MaxOpenConns` | 25 | PostgreSQL default max 100. 4 services × 25 = 100 (aman) |
| `MaxIdleConns` | 10 | Pertahankan pool hangat tanpa membuka terlalu banyak koneksi idle |
| `ConnMaxLifetime` | 5 menit | Hindari koneksi basi yang sudah ditutup sisi server |
| `ConnMaxIdleTime` | 2 menit | Lepaskan koneksi idle yang tidak dipakai lama |

---

## 8. Rate Limiting (Nginx)

**File:** `nginx.conf`

```nginx
limit_req_zone $binary_remote_addr zone=api_limit:10m rate=10r/s;

location /api/ {
    limit_req zone=api_limit burst=20 nodelay;
    limit_req_status 429;
}
```

- **10 req/s per IP** sebagai baseline (bisa disesuaikan saat event)
- **Burst 20**: Lonjakan singkat hingga 20 request diizinkan tanpa delay
- **nodelay**: Request di atas burst langsung 429 (tidak mengantri)
- HTTP **429** dikembalikan sehingga client tahu untuk back-off
