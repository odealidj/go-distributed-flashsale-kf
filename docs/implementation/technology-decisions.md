# Keputusan Teknologi (Tech Stack)

Dokumen ini mendokumentasikan pustaka (*library*) dan teknologi yang dipilih untuk implementasi proyek Flash Sale, beserta alasan di baliknya.

## 1. Bahasa & Workspace
*   **Go (1.21+)**: Dipilih karena kinerja konkurensi (goroutine) yang sangat efisien, konsumsi memori rendah, dan ekosistem *cloud-native* yang kuat. Sangat krusial untuk menahan *load* *Flash Sale*.
*   **Workspace**: `go.work` (Go Workspaces) digunakan untuk mengatur struktur Monorepo agar setiap service memiliki independensi dependensi tanpa harus memisahkannya ke *repository* git yang berbeda.

## 2. Microservice Framework & RPC
*   **Framework**: `go-kratos/kratos/v2`
    *   **Alasan**: Framework berbobot ringan (*lightweight*) yang dirancang khusus untuk memfasilitasi integrasi HTTP REST dan gRPC secara mulus (*seamless*). Sangat cocok dipasangkan dengan pola *Hexagonal Architecture*. Kratos tidak memaksakan struktur folder, ia hanya menyediakan *toolkit* (Middleware, Transport, Config) yang elegan.
*   **Kontrak Komunikasi Internal**: `gRPC` & `Protobuf`
    *   **Alasan**: Komunikasi antar-*service* (seperti API Gateway memanggil Reserve Stock ke Inventory Service) dilakukan via gRPC agar sangat cepat (*binary serialized*) dan memiliki kontrak yang strongly-typed.

## 3. Database & Cache
*   **Database Relasional**: `PostgreSQL`
    *   **Alasan**: Tangguh, mendukung fitur transaksional (ACID) kuat, dan memiliki isolasi level yang baik untuk operasi kritikal (Saga Pattern, Outbox Pattern).
*   **Driver DB**: `jackc/pgx`
    *   **Alasan**: Driver standar *de facto* untuk Postgres di Go yang memiliki performa jauh lebih baik dari `lib/pq` lama.
*   **Query Builder / Mapper**: `jmoiron/sqlx`
    *   **Alasan**: Memungkinkan penulisan raw SQL (menjaga visibilitas performa index/query) sambil memberikan kenyamanan *struct scanning/mapping* tanpa *overhead* yang biasanya ditimbulkan oleh ORM penuh seperti GORM. Kecepatan query murni sangat ditekankan pada skenario *Flash Sale*.
*   **Caching & Atomic Operations**: `Redis`
    *   **Alasan**: Redis memegang peranan paling sentral di proyek *Flash Sale*. Digunakan untuk menyimpan stok awal. Redis Lua Script digunakan untuk mengunci kuota dan mengurangi stok secara *atomic* dan *thread-safe* menghindari *Race Condition* di konkurensi ekstrem.

## 4. Message Broker (Event-Driven Async)
*   **Broker**: `Apache Kafka`
    *   **Alasan**: Mampu memproses jutaan pesan per detik (*High Throughput*). Mendukung *log append-only* sehingga *event* (*Event Sourcing* / Saga) bersifat persisten dan tidak hilang meskipun *service* *consumer* mati.
*   **Go Client**: `twmb/franz-go`
    *   **Alasan**: Client Kafka untuk Go yang dikembangkan khusus untuk kecepatan maksimum, dengan API yang modern dan dukungan context secara *native*, mengalahkan performa `sarama` (Shopify) dan `confluent-kafka-go` (butuh CGO).

## 5. Observability (Log & Trace)
*   **Log**: `log/slog` (Go 1.21+ bawaan)
    *   **Alasan**: Standar logger terstruktur bawaan Go. Digunakan bersama JSON handler.
*   **Tracing**: `OpenTelemetry` (OTel)
    *   **Alasan**: Standar industri modern untuk *distributed tracing*. Jejak *request* dari API Gateway -> Kafka -> Order Service bisa dilacak secara visual menggunakan Jaeger di infrastruktur lokal.

## 6. Reverse Proxy & API Gateway
*   **Reverse Proxy**: `NGINX` (Ditaruh pada layer paling depan di Docker Compose). Mengurus *Rate Limiting* dan pemantauan trafik L7.
*   **API Gateway (BFF)**: Aplikasi Go murni yang bertugas mem-parsing Auth/JWT pengguna, mengagregasi data, dan menjadi tameng sebelum *request* dikirimkan secara internal lewat gRPC ke *backend services*.

## 7. Resilience (Ketahanan Sistem)
*   **Circuit Breaker**: `github.com/sony/gobreaker`
    *   **Alasan**: Ringan, murni Go (tanpa CGO), tidak ada dependensi eksternal, API sederhana dan idiomatis. Alternatif seperti `afex/hystrix-go` lebih kompleks dan kurang aktif. CB ditempatkan di API Gateway karena dialah yang memanggil semua service downstream.
*   **Retry Strategy**: Implementasi kustom murni Go (`shared/pkg/resilience/retry.go`)
    *   **Alasan**: Tidak perlu library eksternal untuk logika retry sederhana. Exponential backoff dengan jitter ±30% cukup untuk mencegah thundering herd saat recovery.
*   **Dead Letter Queue (DLQ)**: Kafka topic `flashsale.order.dlq` (untuk Order Consumer) dan `flashsale.inventory.dlq` (untuk Inventory Consumer)
    *   **Alasan**: Alternatif (drop atau retry tanpa batas) keduanya buruk untuk produksi. DLQ menjamin event tidak hilang sekaligus tidak memblokir consumer pipeline.
