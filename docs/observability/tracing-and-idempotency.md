# Strategi Observability & Idempotency

Sistem terdistribusi (terutama dengan antrean asinkron Kafka) memiliki tingkat kompleksitas tinggi saat proses *debugging*. Dokumen ini mendefinisikan standar **Distributed Tracing (OpenTelemetry)** dan **Idempotensi** untuk mencegah efek ganda akibat pengiriman ulang pesan (*retries*).

---

## 1. OpenTelemetry & Context Propagation

Kita menggunakan standar **OpenTelemetry (OTel)** untuk *tracing*, yang nantinya diekspor ke **Jaeger**.

### Aliran Trace ID (End-to-End)
1. **Reverse Proxy (Lapis 1)**: NGINX/Traefik menginjeksi header `X-Request-Id` jika belum ada.
2. **API Gateway (HTTP -> gRPC)**: 
   * Menerima `X-Request-Id`.
   * Memulai *Span* baru dari OTel. OTel secara otomatis membuat `TraceID` (ID keseluruhan transaksi) dan `SpanID` (ID langkah saat ini).
   * Menyelipkan `TraceID` dan `SpanID` ke dalam **gRPC Metadata**.
3. **Backend Service (gRPC Server)**: 
   * `go-kratos` OTel Middleware otomatis membaca gRPC Metadata dan menyambung *trace* tersebut ke *logger* (`slog`).
4. **Kafka Publisher (Outbox Worker)**:
   * Saat mempublikasikan *event* (misal: `StockReservedEvent`), *worker* mengambil `TraceID` dari context dan menyuntikkannya ke dalam **Kafka Record Headers** (bernama `traceparent`).
5. **Kafka Consumer**:
   * Membaca Kafka Header `traceparent`, mengekstrak `TraceID`, lalu melanjutkannya saat melakukan proses simpan ke Database (menggunakan SQL komentar yang mengandung Trace ID jika memungkinkan).

### Standar Logging
Semua log aplikasi (via `log/slog`) WAJIB menyertakan field terstruktur berikut untuk kemudahan filter di ELK/Kibana:
```json
{
  "level": "INFO",
  "msg": "Menerima notifikasi webhook dari bank",
  "trace_id": "8934kj2h34k23j4h23",
  "span_id": "234jh234",
  "order_id": "ord_123456",
  "service": "payment-service"
}
```

---

## 2. Idempotency (Pencegah Operasi Ganda)

Di *Flash Sale*, satu pengguna bisa saja tidak sabar dan menekan tombol "Beli" 5 kali dalam sedetik. Atau, Kafka mengalami *network jitter* dan mengirim ulang pesan yang sama 2 kali (*At-Least-Once Delivery*).

Kita mengatasi ini di 2 lapis: **Synchronous API** dan **Asynchronous Worker**.

### A. API Gateway / Synchronous (Idempotency Key Header)
Klien (Mobile/Web) **DIWAJIBKAN** mengirim HTTP Header `Idempotency-Key: <UUID>` saat melakukan aksi POST ke `/checkout`.
1. API Gateway mengoper `Idempotency-Key` ini ke *Inventory Service* via gRPC.
2. *Inventory Service* mengecek nilai kunci ini di **Redis**. 
   * Jika kunci sudah ada, itu berarti *request* ini duplikat dari *request* sebelumnya.
   * *Inventory* langsung merespons "Sukses" mengembalikan ID Pesanan yang sama, **TANPA** memotong stok lagi.
   * TTL (Time-to-Live) kunci di Redis diset sekitar 24 jam.

### B. Kafka Consumer (Inbox Pattern)
Untuk *service* yang mendengarkan pesan Kafka (Order & Payment), kita menggunakan pola **Inbox Pattern** untuk mencapai Idempotensi *Exactly-Once Processing* secara logika bisnis.

Setiap *event* Kafka (*JSON Payload*) memiliki `event_id` yang unik (UUID).

**Alur Konsumsi Pesan di Service Tujuan (Contoh di Order Service):**
1. Menerima *Event* `PaymentCompleted` (event_id: `evt_999`).
2. Membuka koneksi Transaksi SQL ( `BEGIN;` ).
3. Mencoba menyimpan `evt_999` ke dalam tabel `inbox_messages`:
   ```sql
   INSERT INTO inbox_messages (id, event_type, payload, status, created_at)
   VALUES ('evt_999', 'PaymentCompleted', '...', 'PROCESSED', NOW());
   ```
4. **Jika INSERT GAGAL** karena melanggar *Primary Key Constraint* (artinya ID ini sudah pernah diproses):
   * *Rollback* transaksi.
   * Jangan kembalikan *error* ke Kafka. Lakukan `commit offset` Kafka agar pesan dianggap sukses, lalu keluar.
5. **Jika INSERT SUKSES**:
   * Lanjutkan mengeksekusi logika bisnis (Ubah status pesanan menjadi `PAID`).
   * Lakukan `COMMIT;` pada transaksi database.
   * Lakukan `commit offset` Kafka.

Pola ini mengamankan sistem agar mutasi data bisnis (update status, pemotongan uang, alokasi inventaris) hanya pernah terjadi tepat **satu kali**.
