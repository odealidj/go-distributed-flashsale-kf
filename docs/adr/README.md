# Architecture Decision Records (ADR)

Folder ini berisi log keputusan arsitektural penting yang diambil selama fase desain dan implementasi proyek Flash Sale.

| ID | Keputusan | Tanggal | Status | Ringkasan |
|---|---|---|---|---|
| ADR-001 | [Gunakan Redis Lua Script untuk Stok](adr-001-redis-lua-stock.md) | 2026-05-29 | Diterima | Hindari RDBMS lock contention; gunakan Redis sebagai *source of truth* sementara. |
| ADR-002 | [Gunakan Kafka untuk Saga Choreography](adr-002-kafka-saga.md) | 2026-05-29 | Diterima | Kurangi latensi pembuatan pesanan dengan asinkronisasi proses checkout. |
| ADR-003 | [Go Hexagonal Architecture](adr-003-hexagonal-go.md) | 2026-05-29 | Diterima | Memastikan *business rules* terisolasi dan mudah di-test tanpa infrastruktur nyata. |
| ADR-004 | [Idempotency Key via Database Unique Constraint](adr-004-idempotency.md) | 2026-05-29 | Diterima | Cegah pesanan ganda/duplikat Kafka *events* dengan tabel `inbox_events`. |
