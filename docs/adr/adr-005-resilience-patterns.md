# ADR-005: Resilience Patterns — Circuit Breaker, Retry, DLQ

**Status:** Diterima  
**Tanggal:** 2026-05-29  
**Konteks:** Phase 07 — Resilience

---

## Konteks

Sistem Flash Sale terdiri dari 5 microservices yang berkomunikasi via gRPC dan Kafka. Tanpa mekanisme resilience, kegagalan satu komponen dapat menyebabkan **cascading failure** ke seluruh sistem:

- Inventory service down → API Gateway terus kirim gRPC → goroutine leak + koneksi exhausted
- Kafka sementara down → event dari Outbox hilang (tidak ada retry)
- Kafka consumer gagal proses 1 event → event hilang karena offset sudah di-commit

---

## Keputusan

### 1. Circuit Breaker per-Service Downstream (API Gateway)

**Library:** `github.com/sony/gobreaker` v0.5.0

**Pilihan yang Dipertimbangkan:**
- `sony/gobreaker`: Ringan, murni Go, tidak ada dependensi eksternal, API sederhana. ✅ **Dipilih**
- `afex/hystrix-go`: Port dari Netflix Hystrix, lebih kompleks, kurang aktif dikembangkan
- `go-resilience`: Kurang dokumentasi dan komunitas kecil

**Konfigurasi:**
```go
// Terbuka jika 50% dari min 10 request gagal dalam 10 detik
// Half-open setelah 5 detik, izinkan 5 request uji coba
DefaultCircuitBreakerConfig(name) → gobreaker.Settings{...}
```

**Alasan Satu CB per Service:**
Jika CB digabung (satu CB untuk semua downstream), kegagalan inventory akan memblokir akses ke product dan payment. Dengan CB terpisah, isolasi granular terjaga.

---

### 2. Retry dengan Exponential Backoff + Jitter

**Implementasi:** Pure Go di `shared/pkg/resilience/retry.go` (tanpa dependensi eksternal)

**Alasan menggunakan jitter:**
Tanpa jitter, semua goroutine yang gagal bersamaan akan retry pada waktu yang sama persis → thundering herd saat recovery. Jitter ±30% menyebarkan retry secara merata.

**Operasi yang TIDAK di-retry:**
| Operasi | Alasan |
|---------|--------|
| `ReserveStock` gRPC call | Non-idempoten: retry dengan event_id baru = pemotongan stok baru |
| Kafka payload parsing | Permanent error, tidak akan sembuh dengan retry |

**Operasi yang di-retry:**
| Operasi | Max Retry | Backoff |
|---------|-----------|---------|
| Outbox publish ke Kafka | 5x | 200ms → 10s |
| Order Consumer process event | 3x | 500ms → 5s |

---

### 3. Dead Letter Queue (DLQ) — Kafka Consumer

**Topic DLQ:** `flashsale.order.dlq`

**Alasan:**
Tanpa DLQ, ada dua pilihan buruk:
- **(a)** Terus retry tanpa batas → consumer lag menggunung, event baru terhambat
- **(b)** Drop event gagal → data hilang, tidak bisa di-audit

DLQ adalah jalan tengah: event gagal tidak dibuang, disimpan dengan metadata (original topic, error message, timestamp) untuk diinspeksi dan di-replay manual.

**Manual Commit Offset:**
`DisableAutoCommit()` digunakan agar offset hanya di-commit **setelah** pemrosesan sukses atau pengiriman ke DLQ. Jika proses crash sebelum commit, Kafka akan redelivery event (at-least-once). Idempotency di sisi consumer (tabel `processed_events`) mencegah pemrosesan ganda.

---

### 4. gRPC Timeout per-Call

**Nilai:** 3 detik

**Alasan pemilihan nilai:**
- Lebih kecil dari proxy timeout Nginx (biasanya 60s default, atau dikonfigurasi 10s)
- Cukup untuk operasi DB yang berat sekalipun
- Memberikan feedback cepat ke client daripada menunggu goroutine hang selamanya

**Keepalive:**
`Time=10s, Timeout=5s` untuk mendeteksi koneksi mati tanpa menunggu request berikutnya timeout.

---

### 5. Database Connection Pool

**Nilai:**
```
MaxOpenConns    = 25   (per service instance)
MaxIdleConns    = 10
ConnMaxLifetime = 5 menit
ConnMaxIdleTime = 2 menit
```

**Alasan MaxOpenConns=25:**
PostgreSQL default `max_connections=100`. Dengan 4 service yang terhubung ke DB (inventory, payment, order, product), 25 per service = 100 total. Pas di batas aman.

---

## Konsekuensi

**Positif:**
- Kegagalan satu service tidak merembet ke service lain
- Event tidak pernah hilang (tersimpan di outbox/DLQ)
- Koneksi idle tidak menyebabkan connection exhaustion

**Negatif:**
- Kompleksitas bertambah: DLQ butuh proses monitoring dan replay manual
- CB bisa menghasilkan false-positive di kondisi network spike sementara
- Manual commit Kafka memerlukan penanganan rebalancing yang lebih hati-hati
