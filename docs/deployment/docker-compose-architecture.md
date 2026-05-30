# Arsitektur Docker Compose (Local & Demo)

## 1. Tujuan
Dokumen ini merancang topologi jaringan lokal (*Docker Compose*) untuk menjalankan infrastruktur pendukung dalam satu perintah `docker-compose up`. Ini sangat berguna untuk pengembangan (*local development*) dan presentasi demo proyek. 

**Penting:** Aplikasi Go (Microservices) **TIDAK** dijalankan di dalam Docker Compose. Mereka dijalankan langsung di mesin *host* via `go run` (di-orkestrasi melalui `Makefile`) agar mudah di-debug.

## 2. Topologi Jaringan & Komponen

Semua komponen infrastruktur tergabung dalam satu Docker *bridge network* internal bernama `flashsale-network`.

```text
[Browser / K6] 
      │ (Port 80)
      ▼
+---------------------+
|   Reverse Proxy     | (NGINX) - Mengurus Rate Limiting
+---------------------+
      │ (Rute /api/v1/*)
      ▼
+-------------------------------------------------------------+
|    API Gateway (Go Process di Host - Port 8000)             |
+-------------------------------------------------------------+
      │ 
      ├───────────────── (gRPC) ────────────────┐
      ▼                                         ▼
+---------------------+                   +---------------------+
|  Inventory Service  |                   |   Product Service   | 
|  (Host Port: 9002)  |                   |  (Host Port: 9001)  |
+---------------------+                   +---------------------+
      │                                         │
      │ (Lua Script Atomic)                     │ (Read Cache)
      ▼                                         ▼
+---------------------------------------------------------------+
|                      REDIS CACHE                              | (Port 6379)
+---------------------------------------------------------------+
      
      [Sistem Asinkron & Database di Bawah Layar]

      (Outbox Publisher Worker dari Inventory)
      │
      ▼
+---------------------------------------------------------------+
|                      APACHE KAFKA                             | (Port 9092, 9094) -> [Kafka UI: 8080]
+---------------------------------------------------------------+
      │ (Consume)                               │ (Consume)
      ▼                                         ▼
+---------------------+                   +---------------------+
|   Order Service     |                   |   Payment Service   | 
| (Host, no gRPC port)|                   |  (Host Port: 9003)  |
+---------------------+                   +---------------------+

+---------------------------------------------------------------+
|                      POSTGRESQL                               | (Port 5432)
+---------------------------------------------------------------+
(Satu database `flashsale` digunakan bersama-sama untuk scaffold)
```

## 3. Daftar Service Docker (Infrastruktur)

| Container Name | Port Eksternal (Host) | Peran |
| :--- | :--- | :--- |
| `nginx` | `80` | Reverse Proxy masuk untuk API. |
| `postgres` | `5432` | RDBMS utama (menggunakan `init.sql`). |
| `redis` | `6379` | *In-memory store* untuk *Atomic Counters* (Stok) & Cache. |
| `kafka` | `9092`, `9094` | *Message Broker* utama. |
| `kafka-ui` | `8080` | Web UI untuk melihat isi *topic* dan *messages*. |
| `jaeger` | `16686`, `4317`, `4318` | Tracing backend + UI (Menerima OTLP via 4317). |

*(Tidak ada container untuk otel-collector, Jaeger langsung menangani OTLP)*.

## 4. Cara Menjalankan

Melalui *Makefile* di Root:

1. **Jalankan Infrastruktur Docker**
   ```bash
   make infra-up
   ```
   *(Hanya menyalakan Nginx, Postgres, Redis, Kafka, Jaeger).*

2. **Jalankan Semua Aplikasi (Microservices)**
   ```bash
   make up
   ```
   *(Menjalankan semua aplikasi Go secara paralel menggunakan script `run_all.sh` atau Makefile command di host).*

## 5. Pertimbangan Skala Produksi (Scaling & PgBouncer)

Di lingkungan produksi yang skalanya jauh lebih besar (misalnya saat dideploy ke Alibaba Cloud dengan replikasi microservices/autoscaling):

*   **Tantangan Koneksi Database:** Setiap instance microservices Go yang berjalan akan membuka *connection pool* ke PostgreSQL. Jika sistem melakukan *autoscaling* menjadi puluhan atau ratusan kontainer, jumlah koneksi fisik ke Postgres akan melonjak ekstrem dan melampaui `max_connections` di Postgres, yang menyebabkan degradasi performa atau penolakan koneksi.
*   **Solusi PgBouncer:** Sangat direkomendasikan untuk menempatkan **PgBouncer** (sebuah *lightweight connection pooler*) di depan PostgreSQL.
    *   Microservices Go akan diarahkan ke port PgBouncer (default: `6432`).
    *   PgBouncer akan mengelola ribuan koneksi masuk dari aplikasi Go dan memetakan koneksi tersebut (*connection multiplexing*) ke jumlah koneksi fisik yang jauh lebih sedikit ke PostgreSQL server secara efisien.

