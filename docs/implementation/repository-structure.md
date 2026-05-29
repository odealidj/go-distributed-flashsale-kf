# Struktur Repository (Monorepo Flash Sale)

## 1. Tujuan
Dokumen ini mendefinisikan struktur *repository* untuk implementasi arsitektur **Monorepo** pada proyek Sistem Flash Sale. Proyek ini terdiri dari 5 *microservices* + 1 Reverse Proxy, dikembangkan menggunakan Go (`go.work` workspaces) dan Hexagonal Architecture.

## 2. Struktur Root

```text
.
├── go.work
├── Makefile
├── docker-compose.yml
├── README.md
├── docs/                      # Dokumentasi Proyek
├── proto/                     # Definisi gRPC/Protobuf
│   ├── inventory/v1/
│   ├── order/v1/
│   ├── payment/v1/
│   └── product/v1/
├── proxy/                     # Reverse Proxy / API Gateway (Misal: NGINX/Envoy/Traefik)
├── services/                  # Business Microservices
│   ├── api-gateway/           # BFF (BFF / Gateway App Go)
│   ├── inventory-service/
│   ├── order-service/
│   ├── payment-service/
│   └── product-service/
├── shared/                    # Library Generic Lintas Service
│   ├── go.mod
│   ├── observability/
│   ├── response/
│   ├── messaging/
│   └── config/
├── scripts/                   # Script Load Test K6 / Seed Data
├── deployments/               # Konfigurasi Kubernetes/Otel/Docker
└── tests/
    └── e2e/
```

## 3. Aturan Root Folder

| Folder/File | Tanggung jawab |
| :--- | :--- |
| `go.work` | File workspace Golang (`go 1.22+`) yang menghubungkan semua modul Go service dan `shared` module. |
| `Makefile` | Entrypoint untuk operasional *developer* (Build, Migrate, Test, Run). |
| `docker-compose.yml` | Infrastruktur lokal (Postgres, Redis, Kafka, Jaeger, Prometheus) & Service Runtime. |
| `docs/` | Dokumentasi spesifikasi, kontrak, dan *architecture decision*. |
| `proto/` | Berisi kontrak antar-*service* (sinkron). Digunakan untuk men-*generate* kode gRPC klien/server. |
| `proxy/` | *Layer Reverse Proxy* terdepan (Traffic Routing, SSL Termination). |
| `services/` | Seluruh *source code* *microservices* (Setiap *service* memiliki `go.mod` independen). |
| `shared/` | *Utility* generik, **DILARANG** menaruh *business logic* toko di sini. |
| `scripts/` | Berisi script *load test* (k6) dan pembuatan database. |

## 3.1 Go Module Strategy
Kita menggunakan pendekatan `go.work` di root, tanpa `go.mod` tunggal di root.

Contoh `go.work`:
```text
go 1.22

use (
  ./shared
  ./services/api-gateway
  ./services/inventory-service
  ./services/order-service
  ./services/payment-service
  ./services/product-service
)
```
**Alasan:**
*   Setiap service memiliki *dependency version* sendiri yang diisolasi oleh `go.mod` masing-masing. (Misal: `inventory` butuh redis client v9, `payment` tidak butuh).
*   Mencegah *Dependency Hell* di dalam Monorepo.
*   Memudahkan kompilasi Docker *image* secara independen.

## 4. Struktur Per Service (Hexagonal Architecture)

Setiap layanan di dalam `services/` wajib mengikuti pola *ports and adapters*:

```text
services/{service-name}/
├── go.mod
├── cmd/
│   ├── api/
│   │   └── main.go        # Entrypoint HTTP/gRPC
│   └── worker/
│       └── main.go        # Entrypoint Kafka Consumer / Background Job
├── internal/
│   ├── domain/            # Core Business Logic (Dilarang ada import framework HTTP/DB)
│   │   ├── model/
│   │   └── service/
│   ├── application/       # Usecases, Saga, Port Interfaces
│   │   ├── port/
│   │   └── usecase/
│   ├── adapter/           # Implementasi spesifik teknologi
│   │   ├── inbound/       # Controller (REST, gRPC Server, Kafka Consumer)
│   │   │   ├── rest/
│   │   │   ├── grpc/
│   │   │   └── kafka/
│   │   └── outbound/      # External (Postgres sqlx, Redis, Kafka Producer, gRPC Client)
│   │       ├── postgres/
│   │       ├── redis/
│   │       └── kafka/
│   ├── config/
│   └── bootstrap/         # Dependency Injection / Wiring
├── migrations/            # File .sql migrasi DB (golang-migrate)
└── test/                  # Integration tests
```

### 4.1 Teknologi Akses Data (Database)
Semua service (Inventory, Order, Payment, Product) menggunakan:
*   **Driver**: `pgx`
*   **Library Query**: `sqlx` (Ekstensi `database/sql` standar Go, memetakan baris DB langsung ke Struct Go tanpa overhead ORM).

*Contoh di `order-service`:*
```text
services/order-service/internal/adapter/outbound/postgres/
├── order_repository_sqlx.go
├── outbox_repository_sqlx.go
└── transaction_runner_sqlx.go
```

## 5. Dependency Rule (Golden Rule)
Arah dependensi wajib mengarah ke dalam (*Domain* adalah raja).

✅ **DIZINKAN:**
`Adapter -> Application -> Domain`

❌ **DILARANG KERAS:**
*   `Domain -> Adapter` (Domain import *package* SQL / Redis)
*   `Domain -> Kratos` (Domain import HTTP *framework*)
*   `Application -> Concrete Adapter` (Usecase memanggil struct `postgres.OrderRepository` secara langsung. Harus pakai *Interface* dari `application/port/`).

## 6. Reverse Proxy dan API Gateway

**Apa bedanya?**
1.  `proxy/` (Reverse Proxy - NGINX/Traefik): Posisi paling depan. Mengurus Load Balancing dasar, Rate Limiting (anti-DDoS awal), SSL/TLS, dan rute trafik mentah ke API Gateway.
2.  `services/api-gateway/` (Go App): BFF (*Backend For Frontend*). Berisi logika menggabungkan data dari Product Service dan meneruskan perintah Flash Sale ke Inventory via gRPC. Ia mengubah gRPC *error* menjadi *Standard JSON Response*.

## 7. Makefile
Fungsi `Makefile` digunakan untuk kelancaran *Developer Experience*.

Contoh target:
```text
make infra-up      # Menjalankan Postgres, Redis, Kafka, Jaeger via docker-compose
make infra-down    # Mematikan infrastruktur

make build-all     # Mem-build kelima Go Service
make migrate-up    # Menjalankan migrasi SQL untuk semua DB

make generate      # Men-generate proto gRPC
```
