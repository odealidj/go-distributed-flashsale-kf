# Metrics & Resource Dashboard (Prometheus + Grafana)

Selain *Distributed Tracing* (Jaeger) untuk melacak *request* spesifik, kita juga wajib memiliki visibilitas terhadap **Kesehatan Sistem Secara Keseluruhan (System Health & Metrics)**. Untuk ini, kita memadukan OpenTelemetry Metrics, Prometheus, dan Grafana.

## 1. Arsitektur Pengumpulan Metrik

```text
[ Go Services (5 pcs) ]
          │ (Push OTLP Metrics)
          ▼
[ OpenTelemetry Collector ]
          │ (Expose /metrics)
          ▼
    [ Prometheus ] (Mengekstrak data secara berkala - Pull)
          │
          ▼
      [ Grafana ] (Visualisasi & Alerting)
```

## 2. Metrik Kunci (RED Metrics)

Setiap *microservice* akan secara otomatis mengekspos metrik **RED**:
1.  **Rate**: Jumlah total *request* HTTP/gRPC per detik (RPS).
2.  **Errors**: Jumlah *request* yang menghasilkan HTTP 5xx atau gRPC *Internal Error*.
3.  **Duration**: Waktu yang dibutuhkan untuk merespons (Latency P50, P90, P99).

## 3. Metrik Spesifik Flash Sale (Custom Metrics)

Selain metrik standar, kita akan menginjeksi beberapa **Custom Metrics** di Go untuk memantau jantung bisnis:
*   `flashsale_stock_remaining_total`: Sisa stok di Redis secara *real-time*.
*   `flashsale_checkout_success_total`: Jumlah orang yang berhasil mendapat alokasi stok.
*   `flashsale_checkout_failed_total`: Jumlah orang yang ditolak (karena kehabisan stok atau melanggar *rate limit*).
*   `kafka_consumer_lag`: Jumlah pesanan yang sedang mengantre di Kafka namun belum diproses oleh *Order Service*. Jika *lag* terus menumpuk, artinya *Order Service* kewalahan.

## 4. Visualisasi di Grafana

Kita akan menyediakan *Dashboard Grafana* yang *pre-configured* (di-*provisioning* otomatis saat `docker-compose up`).

**Panel yang tersedia:**
*   **API Gateway Traffic**: Grafik batang menunjukkan lonjakan 10.000 RPS.
*   **Redis Hit/Miss & Latency**: Menunjukkan kestabilan Redis Lua Script di bawah 5ms.
*   **Saga Completion Rate**: Perbandingan antara *Checkout Created* vs *Order Confirmed* vs *Order Cancelled*.

## 5. Lokasi Akses Lokal

Saat sistem berjalan:
*   **Grafana**: `http://localhost:3000` (User/Pass: `admin/admin`)
*   **Prometheus**: `http://localhost:9090` (Untuk query mentah PromQL)
