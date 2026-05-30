# Struktur Repositori (Monorepo)

Proyek ini dirancang sebagai **Go Workspace Monorepo**, yang berarti semua layanan (*microservices*) dan pustaka bersama (*shared libraries*) berada dalam satu repositori tunggal, namun dikompilasi secara independen (memiliki `go.mod` masing-masing).

Pendekatan ini mempermudah pengembangan lokal (tidak perlu berpindah antar banyak repositori git) namun tetap mempertahankan batas (*boundary*) kode yang ketat antar-*service*.

---

## 1. Topologi Root

Struktur tertinggi pada repositori:

```text
flashsale-kf-basic-go/
├── go.work                          # Mendefinisikan workspace Go
├── Makefile                         # Skrip otomatisasi (infra-up, build, up)
├── docker-compose.yml               # Definisi infrastruktur (DB, Redis, Kafka)
├── nginx.conf                       # Konfigurasi reverse proxy
├── init.sql                         # Skema database awal untuk semua service
├── README.md                        # Dokumentasi utama proyek
├── .github/workflows/               # CI/CD Pipeline (GitHub Actions)
├── docs/                            # Dokumentasi arsitektur, API, dan keputusan (ADR)
├── proto/                           # Definisi gRPC (*.proto) lintas service
├── shared/                          # Pustaka bersama (DB, Outbox, Resilience, Telemetry)
├── api-gateway/                     # BFF (Backend for Frontend)
├── product-service/                 # Layanan Katalog Produk
├── inventory-service/               # Layanan Stok Barang (Core Flashsale)
├── order-service/                   # Layanan Transaksi Pesanan (Saga Coordinator)
├── payment-service/                 # Layanan Pembayaran
├── k6/                              # Script performa uji beban
└── performance-tests/               # Konfigurasi K6 lanjutan & bash scripts
```

> **Catatan Penting:** 
> - Tidak ada folder `services/` yang membungkus. Setiap service diangkat ke tingkat root.
> - `init.sql` menyimpan seluruh tabel dalam 1 database scaffold `flashsale`.

---

## 2. Struktur Pustaka Bersama (`shared/`)

Kode yang dikerjakan ulang dan digunakan oleh lebih dari satu *service* ditempatkan di sini.

```text
shared/
└── pkg/
    ├── database/
    │   └── postgres.go          # Helper koneksi DB dengan sqlx dan connection pool
    ├── outbox/
    │   └── relay.go             # Worker poller (FOR UPDATE SKIP LOCKED) untuk Kafka
    ├── resilience/
    │   ├── circuit_breaker.go   # Wrapper sony/gobreaker
    │   ├── retry.go             # Exponential Backoff dengan Jitter
    │   └── doc.go
    └── telemetry/
        ├── tracer.go            # Inisialisasi OpenTelemetry (Jaeger)
        └── extractor.go         # Helper untuk mengekstrak konteks
```

---

## 3. Struktur Internal Service (Hexagonal Architecture)

Masing-masing service (seperti `inventory-service` atau `payment-service`) dibangun di atas struktur yang sama — yang secara ketat memisahkan kode bisnis dari infrastruktur.

```text
<nama-service>/
├── cmd/
│   └── <nama-service>/
│       ├── main.go              # Entry point utama aplikasi
│       ├── wire.go              # (Opsional) Deklarasi Dependency Injection (Wire)
│       └── wire_gen.go          # (Opsional) Hasil auto-generate Wire
└── internal/
    ├── domain/                  # Lapis 1: Core Domain (Tidak punya dependensi luar)
    │   └── model/               # Struct data utama (Product, Order, Inventory)
    │
    ├── application/             # Lapis 2: Business Rules / Use Cases
    │   ├── port/                # Interface untuk Inbound & Outbound (Repository)
    │   └── usecase/             # Logika bisnis (Memanggil port, tidak peduli DB/gRPC)
    │
    └── adapter/                 # Lapis 3: Infrastruktur (Menghubungkan Port)
        ├── inbound/             # Driver (Primary)
        │   ├── grpc/            # Menerima request dari luar via gRPC
        │   ├── rest/            # Menerima request via HTTP (khusus di api-gateway)
        │   └── kafka/           # Menerima request via Kafka Consumer
        │
        └── outbound/            # Driven (Secondary)
            ├── postgres/        # Implementasi Repository Port untuk Database
            ├── redis/           # Implementasi eksekusi Lua Script (Atomic ops)
            └── grpc/            # Memanggil service lain (Circuit Breaker & Retry)
```

**Aturan Emas Dependensi (*Dependency Rule*):**
1. `domain` **tidak boleh** meng-import `application` atau `adapter`.
2. `application` **tidak boleh** meng-import `adapter`. Ia hanya mendefinisikan *interface* (di `port`).
3. `adapter` bergantung pada `application` (untuk mengimplementasikan *port*) dan `domain` (untuk melakukan *mapping* data).
4. `cmd/main.go` (atau *Wire*) bertanggung jawab menyatukan semuanya (*Dependency Injection*). Service `product`, `inventory`, dan `payment` menggunakan `google/wire` untuk DI, sementara `order` manual.
