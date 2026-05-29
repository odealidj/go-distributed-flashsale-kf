# Kafka Operational Design

Untuk menunjang *thundering herd* 10.000 RPS, arsitektur Kafka harus dikonfigurasi dengan benar agar tidak menjadi *bottleneck* tunggal.

## 1. Partisi Topik (Topic Partitions)

Partisi adalah kunci konkurensi di Kafka. Jika topik `flashsale.inventory.events` hanya memiliki 1 partisi, maka Order Service hanya bisa memproses pesanan dengan 1 *worker* secara konkuren.

- **Rekomendasi Partisi:** Minimal **10 partisi** per topik utama.
- **Keuntungan:** Memungkinkan hingga 10 instance Order Service (*consumer group members*) membaca event secara bersamaan (secara paralel).

## 2. Partition Key

Sangat penting agar pesanan untuk *user_id* yang sama selalu jatuh ke partisi yang sama untuk menjamin urutan eksekusi (*ordering*).
- **Key yang digunakan:** `user_id` atau `order_id` (tergantung event).

## 3. Retention Policy

Data *Flash Sale* bersifat transaksional jangka pendek. Kita tidak membutuhkan riwayat tak terbatas di Kafka (biarkan database yang menyimpannya).
- **log.retention.hours:** 24 jam (Sangat cukup untuk durasi Flash Sale + investigasi bug sehari setelahnya).

## 4. Consumer Group

- Setiap service harus memiliki `group_id` yang spesifik dan konstan (misal: `order-service-group`).
- Skala: Jumlah pods/instance dari `order-service` idealnya sama dengan jumlah partisi Kafka (10 Pods untuk 10 Partitions) untuk mencapai performa maksimal. Jika Pods lebih banyak dari partisi, Pod sisa akan *idle*.
