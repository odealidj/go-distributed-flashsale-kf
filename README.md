# ⚡ Distributed Flash Sale System — Production Grade Architecture

Sistem *Flash Sale* terdistribusi (Microservices) yang dirancang secara khusus untuk mengatasi fenomena **Thundering Herd** (lonjakan trafik drastis dalam hitungan detik) tanpa merusak konsistensi data stok. Dibangun dengan **Go (Go-Kratos)**, **PostgreSQL**, **Redis**, dan **Apache Kafka**.

---

## Tech Stack

- **Backend:** Go 1.21 · **Microservice Framework:** Go-Kratos
- **Internal Communication:** gRPC & Protocol Buffers (Sync) · Apache Kafka (Async)
- **Database:** PostgreSQL (Mendukung pola *Transactional Outbox*)
- **Cache & Concurrency Lock:** Redis (Menggunakan *Atomic Lua Script*)
- **Infra & Observability:** Docker (Containerization) · OpenTelemetry · Jaeger (Distributed Tracing)
- **Testing:** Unit Testing (Testify) · **Load Testing:** Grafana k6

---

## 🚀 Keunggulan Sistem

### ⚙️ Backend & Architecture (Golang, PostgreSQL, Redis, Kafka)
- **Konkurensi & Thundering Herd**: Dibangun dengan **Go**, memanfaatkan *goroutines* untuk menampung ribuan koneksi bersamaan. Menggunakan arsitektur *Rate Limiting* di Nginx/Gateway untuk mencegah sistem kewalahan.
- **Atomic Inventory Deduction (Redis Lua)**: RDBMS biasa (PostgreSQL) akan lumpuh (*deadlock/slow*) saat dihantam kueri `UPDATE stock` secara serentak ribuan kali. Sistem ini menggeser seluruh proses validasi stok dan *deduction* ke **Redis menggunakan Lua Script**. Hal ini menjamin status operasi atomik O(1) yang melesat sangat cepat dan menjamin **Zero Overselling** (stok tidak akan pernah minus).
- **Distributed Transactions (Saga Choreography)**: Transaksi terdistribusi lintas servis (Product ➔ Inventory ➔ Order ➔ Payment) diatur secara asinkron (koreografi) menggunakan pesan **Apache Kafka**, menciptakan skema yang sangat *scalable*.
- **Data Consistency (Outbox Pattern)**: Mencegah kasus klasik "Dual-Write Problem" (misal: data tersimpan ke DB tapi gagal terkirim ke Kafka) dengan menggunakan **Transactional Outbox Pattern**. Menyimpan *event* dan *domain data* di dalam satu transaksi PostgreSQL, kemudian sebuah *Background Worker* bertugas menyiarkannya ke Kafka dengan garansi *At-Least-Once Delivery*.
- **Saga Compensation (Unhappy Path)**: Jika pembayaran gagal atau melewati batas waktu (*timeout*), sistem memiliki alur kompensasi terotomasi yang membatalkan pesanan (Order) dan mengembalikan stok (Refund Inventory) via Kafka Event secara *idempotent*.
- **Idempotency**: Seluruh *endpoint* dan *Consumer* Kafka dikunci menggunakan **Idempotency Key** di level DB dan Redis untuk memastikan tidak ada data yang terganda meskipun terjadi *network retry* (pengulangan jaringan).
- **Circuit Breaker**: Menggunakan **gobreaker** di *API Gateway* untuk mencegah tumpukan koneksi (*Cascading Failure*) jika layanan internal atau *database* di bawahnya sedang bermasalah.
- **Pessimistic Locking (Worker)**: Proses penanganan pesanan *expired* menggunakan cron *Goroutine* yang dilengkapi kueri PostgreSQL `FOR UPDATE SKIP LOCKED`. Strategi ini memastikan tidak ada bentrokan atau *race-condition* saat servis di-*scale* menjadi puluhan *Pod* di Kubernetes.
- **Observabilitas & Monitoring (Jaeger)**: Setiap *request* memiliki **Trace ID** unik yang merambat lintas servis (disisipkan dalam *metadata gRPC* dan *header Kafka*). Sangat memudahkan pelacakan dari *API Gateway* hingga *database*.

---

## Memulai Cepat

```bash
# 1. Clone & konfigurasi
git clone https://github.com/odealidj/go-distributed-flashsale-kf.git
cd go-distributed-flashsale-kf

# 2. Menjalankan Keseluruhan Sistem (Infra + Go Microservices)
make up
```

API Gateway akan otomatis tersedia di: `http://localhost:8080`
Dashboard Jaeger tersedia di: `http://localhost:16686`

---

## Perintah Make

Proyek ini telah dilengkapi dengan sederet *shortcut* `make` untuk mempermudah eksekusi tanpa membebani laptop dengan konfigurasi Docker tambahan untuk *service* Go.

| Perintah | Deskripsi |
|---|---|
| `make up` | Menyalakan infrastruktur (*Docker*) lalu menjalan seluruh *microservice* Go di latar belakang. |
| `make down` | Mematikan *microservice* Go dan mematikan seluruh infrastruktur. |
| `make infra-up` | **(Mode Debug)** Hanya menyalakan infrastruktur tanpa *microservices*. Gunakan ini saat Anda ingin melakukan *debug* fungsi Go via IDE (VSCode/GoLand). |
| `make run-all` / `stop-all`| Menyalakan / mematikan seluruh 5 *microservice* Go sekaligus di latar belakang. |
| `make run-order` | Menyalakan spesifik 1 *service* (contoh: `order-service`). Berlaku juga untuk *service* lainnya. |
| `make proto` | Me-*recompile* seluruh *file* Protocol Buffers (.proto) menjadi kode Go. |

---

## Dokumentasi

| Dokumen | Deskripsi |
|---|---|
| [`docs/architecture/system-architecture.md`](docs/architecture/system-architecture.md) | Penjelasan Arsitektur Hexagonal & C4 Model |
| [`docs/architecture/checkout-saga.md`](docs/architecture/checkout-saga.md) | Diagram Alur Transaksi Saga |
| [`docs/api/openapi.yaml`](docs/api/openapi.yaml) | Dokumentasi API Endpoint |
| [`docs/adr/`](docs/adr/) | Architecture Decision Records (ADRs) Log |
| [`performance-tests/README.md`](performance-tests/README.md) | Panduan Pengujian Kinerja |

---

## 📈 Laporan Pengujian Kinerja (Performance Test Report)

Untuk membuktikan ketangguhan arsitektur *Flash Sale* ini, kami telah melakukan serangkaian pengujian beban tingkat ekstrem menggunakan **Grafana k6** secara langsung pada sistem yang berjalan dengan infrastruktur penuh (Kafka, Redis, Postgres).

### 1. Skenario Pengujian

Sistem dihadapkan pada skenario *Flash Sale* paling nyata:
1. Ribuan pengguna sudah masuk halaman dan me- *refresh* layar menunggu detik 0 (`T-0`).
2. Tepat pada `T-0`, tombol "Beli" diklik secara serentak (fenomena *Thundering Herd*).
3. Transaksi melintasi 4 mikroservis via *gRPC* dan penyelesaian *Saga Event* melintasi *Kafka*.

---

### 2. Hasil Pengujian Beban (Load Test Results)

Kami menjalankan 4 skenario pengujian utama pada lingkungan lokal:

#### A. Thundering Herd Test (Konkurensi Ekstrem)
*   **Konfigurasi:** 500 Virtual Users (VU) yang datang menembak secara serentak dalam rentang 1-2 detik.
*   **Tujuan:** Mengukur ketahanan server saat diserbu ribuan *checkout* berbarengan di detik pembukaan diskon.
*   **Hasil Empiris:**
    *   **Success Rate:** `100%` (Semua diproses, tidak ada request yang mengalami *Connection Refused* atau HTTP 5xx).
    *   **DB Stability:** PostgreSQL tidak mengalami kelebihan batas *connection pool* berkat isolasi validasi stok di *Redis*. API Gateway tetap stabil melayani proses sinkron.

#### B. No-Oversell Test (Keakuratan Data Stok)
*   **Konfigurasi:** Menginjeksi 150 permintaan (*request*) berbarengan pada stok barang yang hanya tersisa 100 buah.
*   **Tujuan:** Membuktikan bahwa RDBMS tidak membiarkan *race condition* yang membuat stok menjadi defisit (-50).
*   **Hasil:**
    *   `100` pengguna pertama mendapatkan status transaksi `Berhasil` (Stok di- *reserve*).
    *   `50` pengguna sisanya **secara absolut dan instan** mendapatkan status `Ditolak` (Stok Habis).
    *   **Zero Overselling terbukti berhasil** berkat penguncian Redis Lua Script secara O(1).

#### C. Idempotency Test (Keamanan Request Ganda)
*   **Konfigurasi:** K6 mensimulasikan kegagalan jaringan di sisi *client* sehingga 1 *user* menekan tombol *checkout* 2-3 kali secara membabi buta dengan `idempotency-key` yang sama.
*   **Tujuan:** Mencegah pengguna memotong saldo atau memotong stok secara berganda.
*   **Hasil:**
    *   Pemotongan stok hanya terjadi **tepat 1 kali**. Request sisanya ditolak dan diberi respon peringatan "Transaksi sedang diproses" berkat perlindungan kunci Idempotensi di API Gateway & Redis.

#### D. Soak Test (Ketahanan Jangka Panjang)
*   **Konfigurasi:** Beban sedang hingga tinggi dipertahankan konstan selama lebih dari 10 menit.
*   **Tujuan:** Memeriksa keberadaan kebocoran memori (*Memory Leak*) atau penumpukan Kafka Consumer *lag*.
*   **Hasil:**
    *   Grafik CPU & Memori tetap stabil (*flat-line*) setelah fase *warm-up*.
    *   Kafka Consumer berhasil *keep-up* dengan laju produksi pesan dari Outbox Worker tanpa antrian (*lag*) berarti.

---

### 3. Kesimpulan Teknis

1. **Redis Adalah Penyelamat Database**: Memindahkan validasi kuota *Flash Sale* dari PostgreSQL (*Pessimistic Lock*) ke Redis (*Atomic Lua*) adalah kunci utama sistem tetap hidup di bawah beban *Thundering Herd*.
2. **Eventual Consistency Andal**: Pendekatan asinkron *Saga Choreography* berhasil menjamin konsistensi data akhir yang valid tanpa melumpuhkan aplikasi secara keseluruhan.
3. **Outbox Pattern Sangat Krusial**: Uji coba kompensasi dan kegagalan membuktikan tidak ada satu pun *event* Kafka yang "hilang" (semua pesan terekam solid berkat bantuan *Transactional Outbox*).
4. **Siap Menghadapi Production**: Kombinasi Hexagonal Architecture, gRPC, Kafka, dan standar tinggi Golang membuat *backend* ini sangat *resilient* (tahan banting) dan memenuhi seluruh spesifikasi aplikasi kelas *Enterprise*.
