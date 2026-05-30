# Architecture Decision Records (ADR)

Folder ini berisi log keputusan arsitektural penting yang diambil selama fase desain dan implementasi proyek Flash Sale.

| ID | Keputusan | Tanggal | Status | Ringkasan |
|---|---|---|---|---|
| ADR-001 | Gunakan Redis Lua Script untuk Stok | 2026-05-29 | Diterima | Hindari RDBMS lock contention; gunakan Redis sebagai *source of truth* sementara untuk operasi stok atomik. |
| ADR-002 | Gunakan Kafka untuk Saga Choreography | 2026-05-29 | Diterima | Kurangi latensi pembuatan pesanan dengan asinkronisasi proses checkout via event-driven Kafka. |
| ADR-003 | Go Hexagonal Architecture | 2026-05-29 | Diterima | Memastikan *business rules* terisolasi dan mudah di-test tanpa infrastruktur nyata. |
| ADR-004 | Idempotency Key via Database Unique Constraint | 2026-05-29 | Diterima | Cegah pesanan ganda/duplikat Kafka *events* dengan tabel `processed_events` (PK = `event_id`). |
| ADR-005 | [Resilience Patterns — Circuit Breaker, Retry, DLQ](adr-005-resilience-patterns.md) | 2026-05-29 | Diterima | Cegah *cascading failure* dengan CB per-service, retry+backoff, DLQ, timeout, dan connection pool. |

> **Catatan:** ADR-001 sampai ADR-004 dicatat sebagai ringkasan di tabel ini. ADR-005 memiliki dokumen terperinci karena pola resiliensi melibatkan banyak komponen lintas servis.
