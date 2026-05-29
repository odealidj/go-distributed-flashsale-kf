# Arsitektur Docker Compose (Local & Demo)

## 1. Tujuan
Dokumen ini merancang topologi jaringan lokal (*Docker Compose*) untuk menjalankan seluruh 5 *microservices*, *Reverse Proxy*, dan infrastruktur pendukung dalam satu perintah `docker-compose up`. Ini sangat berguna untuk pengembangan (*local development*) dan presentasi demo proyek.

## 2. Topologi Jaringan & Komponen

Semua komponen akan tergabung dalam satu Docker *bridge network* internal bernama `flashsale_net`. Hanya *Reverse Proxy* dan *port* infrastruktur tertentu yang diekspos ke mesin *host* (laptop Anda).

```text
[Browser / K6] 
      │ (Port 80 / 443)
      ▼
+---------------------+
|   Reverse Proxy     | (Traefik / NGINX) - Mengurus SSL, Rate Limiting
+---------------------+
      │ (Rute /api/v1/*)
      ▼
+---------------------+
|    API Gateway      | (BFF Go) - Validasi Auth, Agregasi, Transform HTTP -> gRPC
+---------------------+
      │ 
      ├───────────────── (gRPC) ────────────────┐
      ▼                                         ▼
+---------------------+                   +---------------------+
|  Inventory Service  |                   |   Product Service   | 
+---------------------+                   +---------------------+
      │                                         │
      │ (Lua Script Atomic)                     │ (Read Cache)
      ▼                                         ▼
+---------------------------------------------------------------+
|                      REDIS CACHE                              | (Port 6379 - Exposed for debug)
+---------------------------------------------------------------+
      
      [Sistem Asinkron & Database di Bawah Layar]

      (Outbox Publisher Worker dari Inventory)
      │
      ▼
+---------------------------------------------------------------+
|                      APACHE KAFKA                             | (Port 9092) -> [Kafka UI: 8080]
+---------------------------------------------------------------+
      │ (Consume)                               │ (Consume)
      ▼                                         ▼
+---------------------+                   +---------------------+
|   Order Service     |                   |   Payment Service   | 
+---------------------+                   +---------------------+

+---------------------------------------------------------------+
|                      POSTGRESQL                               | (Port 5432 - Exposed for DataGrip/DBeaver)
+---------------------------------------------------------------+
(Setiap service punya logical database sendiri: db_order, db_inventory, dll)
```

## 3. Daftar Service Docker

| Container Name | Port Eksternal (Host) | Peran |
| :--- | :--- | :--- |
| `reverse-proxy` | `80`, `443` | NGINX / Traefik. Pintu masuk tunggal dunia luar. |
| `api-gateway` | *(Internal 8080)* | BFF untuk *frontend*. |
| `inventory-service` | *(Internal 9000)* | gRPC Server, mengatur stok di Redis. |
| `order-service` | *(Internal 9001)* | Kafka Consumer, mengelola saga transaksi. |
| `payment-service` | *(Internal 9002)* | gRPC Server untuk webhook bank / proses pembayaran. |
| `product-service` | *(Internal 9003)* | gRPC Server untuk katalog. |
| `postgres` | `5432` | RDBMS utama. |
| `redis` | `6379` | *In-memory store* untuk *Atomic Counters* (Stok) & Cache. |
| `kafka` | `9092` | *Message Broker* utama. |
| `kafka-ui` | `8080` | Web UI (Prove/Kafka-UI) untuk melihat isi *topic* dan *messages*. |
| `jaeger` | `16686` | Web UI untuk melihat visualisasi *Distributed Tracing*. |
| `otel-collector` | *(Internal 4317)* | Penerima *trace* gRPC dari aplikasi Go sebelum dikirim ke Jaeger. |

## 4. Cara Menjalankan

Melalui *Makefile* di Root:

1. **Jalankan Infrastruktur Saja (Untuk *Development* di Terminal)**
   ```bash
   make infra-up
   ```
   *(Hanya menyalakan Postgres, Redis, Kafka, Jaeger, Otel). Aplikasi Go dijalankan manual lewat terminal IDE agar mudah di-debug.*

2. **Jalankan Semuanya (Mode Demo / K6 Load Test)**
   ```bash
   make demo-up
   ```
   *(Membangun image Go dan menjalankan semua 12 kontainer sekaligus).*
