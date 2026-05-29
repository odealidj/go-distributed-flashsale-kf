# Phase Implementasi (Flash Sale System)

## 1. Tujuan
Dokumen ini membagi pekerjaan pembuatan 5 *microservices* + API Gateway untuk sistem Flash Sale menjadi fase-fase yang jelas, *incremental*, dan terstruktur. Pendekatan ini memastikan kita bisa memvalidasi logika inti (Redis Lua Script & Kafka Saga) secepat mungkin sebelum mempercantik UI atau menambah fitur sekunder.

## 2. Definition of Done Umum
Sebuah fase dianggap selesai jika:
*   Kode mengikuti *Hexagonal Architecture*.
*   REST Response API Gateway mengikuti `docs/api/response-standard.md`.
*   Operasi database tidak lagi menggunakan `sqlc`, melainkan di-mapping manual dengan `sqlx`.
*   Semua perubahan berhasil dijalankan secara lokal via `docker-compose`.

---

## 3. Fase 01 - Fondasi Monorepo & Infrastruktur
**Target:** Menyiapkan kerangka kerja kosong namun bisa di-*build*.
*   Inisialisasi `go.work` di root dan `go.mod` di 5 service (`api-gateway`, `inventory-service`, `order-service`, `payment-service`, `product-service`) plus `shared`.
*   Membuat `docker-compose.yml` untuk PostgreSQL, Redis, Kafka (dengan Kafka UI), Jaeger, dan Reverse Proxy (Traefik/NGINX).
*   Menyiapkan skrip migrasi awal.
*   Membuat *proto files* (gRPC contracts) untuk semua komunikasi internal.

## 4. Fase 02 - Katalog Produk & Inventory (Core Flash Sale)
**Target:** Fitur paling kritikal, yaitu memotong stok dengan kecepatan cahaya, dapat berfungsi secara independen.
*   **Product Service:** Membuat REST/gRPC endpoint untuk membaca katalog produk. Caching menggunakan Redis.
*   **Inventory Service:** Mengimplementasikan **Redis Lua Script** untuk atomic counter (Reserve Stock). Jika stok di Redis berhasil dipotong, buat transaksi *Outbox* di PostgreSQL untuk secara async mengirim pesan ke Kafka.
*   Endpoint testing independen untuk memborbardir Inventory Service.

## 5. Fase 03 - API Gateway & Reverse Proxy
**Target:** Menyediakan satu pintu masuk terpusat untuk klien eksternal.
*   Setup *Reverse Proxy* (Traefik/NGINX) untuk *rate-limiting*.
*   **API Gateway:** Membuat BFF (*Backend for Frontend*) menggunakan Go. Menerima *request HTTP REST*, memvalidasi Auth (opsional), lalu mengonversinya menjadi panggilan `gRPC` ke *Product Service* dan *Inventory Service*.
*   Format response di-standarisasi di sini.

## 6. Fase 04 - Order Core & Payment (Saga Choreography)
**Target:** Mengimplementasikan mesin *state* (Saga) berbasis *Event-Driven* Kafka.
*   **Order Service:** Mendengarkan Kafka (Kafka Consumer via `franz-go`). Saat menerima `StockReservedEvent` dari Inventory, simpan pesanan dengan status `PENDING_PAYMENT`.
*   **Payment Service:** Membuat endpoint gRPC `/pay` (dipanggil oleh API Gateway). Mensimulasikan sukses/gagal bayar. Jika sukses, lempar `PaymentCompletedEvent` ke Kafka.
*   **Order Service (Compensation):** Jika *Order Service* menyadari pesanan kadaluwarsa (15 menit), lempar `OrderCancelledEvent`. *Inventory Service* mendengar ini dan mengembalikan stok ke Redis.

## 7. Fase 05 - Observability (Tracing & Idempotency)
**Target:** Mencegah masalah sistem terdistribusi (duplikasi pesan & sulit dilacak).
*   Penerapan *Idempotency Key* di setiap *Kafka Consumer* (menggunakan tabel `inbox_messages` di Postgres) agar tidak ada pesanan ganda jika Kafka mengirim ulang pesan (at-least-once delivery).
*   Injeksi *OpenTelemetry* Trace ID dari API Gateway, diteruskan ke Metadata gRPC, hingga masuk ke Header Kafka agar satu *checkout* bisa divisualisasikan dari ujung ke ujung di Jaeger.

## 8. Fase 06 - Performance Testing (K6)
**Target:** Membuktikan arsitektur tahan menghadapi *Thundering Herd Problem*.
*   Membuat *script* k6 di folder `scripts/` untuk mensimulasikan 10,000 *concurrent users* mencoba membeli 100 stok barang dalam detik yang sama.
*   Verifikasi bahwa *Redis Lua Script* sukses mencegah stok minus (*overselling*).
*   Verifikasi bahwa *API Gateway* dapat merespons di bawah 200ms meskipun Kafka di-*backend* sibuk memproses pesanan secara asinkron.
