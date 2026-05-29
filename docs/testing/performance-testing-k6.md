# Strategi Performance Testing (K6)

Untuk sistem yang diklaim sebagai **Flash Sale Real-Time**, tes fungsional saja tidak cukup. Kita harus membuktikan secara ilmiah bahwa arsitektur mikroservis ini tahan terhadap fenomena *Thundering Herd* (ribuan pengguna mengakses secara serentak di detik yang sama).

## 1. Alat yang Digunakan: K6 (Grafana Labs)
Kita menggunakan **K6** karena skripnya berbasis JavaScript, sangat ringan, dan didesain khusus untuk *load testing* modern di level API/gRPC.

## 2. Skenario Uji Flash Sale

Uji performa akan dibagi menjadi 3 skenario yang masing-masing memiliki tujuan validasi berbeda:

### A. Smoke Test
*   **Tujuan**: Memastikan konfigurasi *routing*, koneksi database, dan Kafka nyala semua tanpa error di kondisi normal.
*   **Load**: 5 Virtual Users (VUs) selama 10 detik.
*   **Target KPI**: 0% error, P95 latency < 50ms.

### B. Load Test (Kondisi Ramai)
*   **Tujuan**: Mengetahui kinerja sistem dalam kondisi sibuk namun belum masuk fase Flash Sale yang ekstrem.
*   **Load**: Naik bertahap hingga 500 VUs selama 3 menit.
*   **Target KPI**: P99 latency < 200ms.

### C. Spike Test / Flash Sale Simulation (Extreme)
*   **Tujuan**: Menyimulasikan jam 12:00:00 teng! Ribuan pengguna mengklik "Beli" di waktu yang hampir bersamaan.
*   **Load**: Langsung menembak ke **10.000 VUs** dalam waktu 5 detik.
*   **Target KPI**:
    *   Sistem tidak boleh *crash* (OOM).
    *   **TIDAK BOLEH ADA OVERSELLING** (Stok minus). Jika stok awal 100, persis 100 pesanan sukses (`HTTP 202`), dan 9.900 lainnya mendapatkan `HTTP 409 INSUFFICIENT_STOCK`.
    *   Waktu respons Redis Lua Script harus tetap stabil di bawah 10ms.

## 3. Direktori Skrip k6

Semua skrip performa akan disimpan di folder `scripts/k6/`.
Contoh *flow* skrip k6:
1. VU melakukan HTTP GET ke API Gateway `/api/v1/products/flashsale`.
2. VU mengambil ID Produk.
3. VU melakukan HTTP POST ke `/api/v1/checkout` dengan `Idempotency-Key` berupa UUID.

## 4. Eksekusi Lokal

Anda bisa menjalankannya lewat instruksi `Makefile` yang akan kita buat:
```bash
make perf-smoke
make perf-spike
```
