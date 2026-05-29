# Panduan Implementasi untuk AI & Developer

File ini adalah instruksi bagi Anda (atau AI *agent* lain) yang akan melanjutkan implementasi kode (*codebase generation*) berdasarkan folder dokumentasi ini.

## 1. Aturan Emas (Golden Rules)
- **Jangan mulai menulis kode sebelum membaca PRD (`docs/product/prd.md`) dan System Architecture (`docs/architecture/system-architecture.md`).**
- Pahami *Saga Pattern* dan alur Kafka di `docs/architecture/checkout-saga.md` sebelum mendesain `OrderService`.
- Implementasi Go HARUS menggunakan Hexagonal Architecture (lihat `docs/implementation/go-hexagonal-architecture.md`).

## 2. Urutan Pembangunan (Build Order)

Disarankan membangun sistem secara bertahap dari bawah ke atas (*bottom-up infrastructure, top-down feature*):

1. **Infrastruktur Dasar**: Buat file `docker-compose.yml` yang berisi PostgreSQL, Redis, Kafka + UI, Jaeger, dan Prometheus.
2. **Library Bersama (pkg)**: Buat konfigurasi Logger (slog dengan Trace ID), Kafka Client Wrapper, dan Error Handler terpusat.
3. **Inventory Service (Core)**: 
   - Mulai dari Redis Lua Script implementation.
   - Buat gRPC Handler untuk ReserveStock.
4. **API Gateway**: Buat gerbang depan REST yang meneruskan request ke Inventory gRPC.
5. **Order Service (Saga Core)**:
   - Buat Kafka Consumer untuk `StockReservedEvent`.
   - Implementasikan *Idempotency inbox table*.
6. **Payment Service**: Implementasikan *mock* webhook dan pengiriman *event* sukses ke Kafka.

## 3. Komunikasi Antar Proses
- **Synchronous**: API Gateway memanggil layanan *backend* menggunakan gRPC berdasarkan kontrak proto di `docs/grpc/`.
- **Asynchronous**: Layanan *backend* berkomunikasi satu sama lain menggunakan Kafka berdasarkan kontrak `asyncapi.yaml`. **DILARANG KERAS** memanggil gRPC antar *backend service* untuk urusan mutasi data (misal: Order tidak boleh memanggil gRPC Payment untuk membayar, harus via event).

Jika ada ambiguitas dalam instruksi masa depan, merujuklah pada aturan bisnis di `docs/specs/business-rules.yaml` sebagai keputusan final.
