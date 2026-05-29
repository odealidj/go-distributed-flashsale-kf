# Product Requirements Document (PRD): Flash Sale Microservices

## 1. Ringkasan Eksekutif
Proyek ini bertujuan untuk membangun sistem *backend* *Flash Sale* yang mampu menangani lonjakan *traffic* secara tiba-tiba (*thundering herd*) tanpa mengalami *downtime* atau *overselling* (stok terjual melebihi persediaan fisik).

## 2. Asumsi Traffic & Skala (Target NFR)
- **Concurrent Users (Peak):** 10.000 user menekan tombol "Beli" secara serentak di detik ke-1.
- **Stock Available:** 500 unit (sangat terbatas).
- **Read/Write Ratio:** 99% Read (cek produk/stok), 1% Write (pemotongan stok).
- **Latency Target:** API `POST /order` merespons dalam < 100ms (HTTP 202 Accepted).

## 3. Aturan Bisnis (Business Rules)
1. **No Overselling (Paling Kritis):** Jumlah pesanan yang berhasil (status PAID atau PENDING_PAYMENT) tidak boleh lebih dari total inventaris awal. Inventory >= 0 di semua kondisi.
2. **Quota Per User:** Satu User ID hanya diperbolehkan membeli maksimal 1 unit barang per sesi *flash sale*.
3. **Payment Window:** Pengguna memiliki waktu tepat **15 menit** untuk menyelesaikan pembayaran setelah pesanan dibuat. Jika lewat, pesanan batal otomatis dan stok dikembalikan.
4. **Fairness:** Siapa cepat dia dapat (First Come First Serve).
5. **Idempotency:** Jika klien mengirim request ganda akibat koneksi buruk, sistem harus memprosesnya sebagai 1 request tunggal.

## 4. Metrik Sukses
- *Zero Overselling* terbukti melalui *Load Test*.
- API Gateway sanggup menerima 10k RPS tanpa putus (*rate limiter* diaktifkan untuk *excess traffic*).
- Observability lengkap (Trace ID dapat diikuti dari Gateway hingga database).

## 5. Scope
**In-Scope:**
- Registrasi dan login user sederhana (atau *mock auth*).
- Menampilkan produk *flash sale*.
- Menahan (reserve) stok dan membuat pesanan.
- Simulasi pembayaran sukses/gagal.
- Pengembalian stok (*compensation*) jika batas waktu terlewati.

**Out-of-Scope:**
- Integrasi Payment Gateway sungguhan (akan dimock).
- CMS Admin untuk menambah produk (bisa di-*seed* via script).
- UI/Frontend lengkap (API First approach).
