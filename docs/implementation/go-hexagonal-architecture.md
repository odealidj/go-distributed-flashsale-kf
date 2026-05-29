# Panduan Go Hexagonal Architecture (Ports and Adapters)

## 1. Konsep Utama
Untuk mengisolasi logika bisnis (*Domain*) dari detail teknis (Database, Kafka, gRPC, HTTP), kita menggunakan *Hexagonal Architecture* (dikenal juga sebagai *Ports and Adapters*). 

Prinsip utamanya adalah **Dependency Inversion**:
Layer terluar bergantung pada layer di dalamnya. Layer dalam (Domain) **tidak boleh tahu menahu** tentang keberadaan layer luar (Postgres, Redis, Kafka, REST).

```text
+-----------------------------------------------------------+
|                        ADAPTERS                           |
|  (HTTP REST, gRPC Server, PostgreSQL, Redis, Kafka)       |
|                                                           |
|    +-------------------------------------------------+    |
|    |                 APPLICATION                     |    |
|    | (Usecase, Saga, Input/Output Ports)             |    |
|    |                                                 |    |
|    |      +-----------------------------------+      |    |
|    |      |             DOMAIN                |      |    |
|    |      | (Entities, Rules, Value Objects)  |      |    |
|    |      +-----------------------------------+      |    |
|    +-------------------------------------------------+    |
+-----------------------------------------------------------+
```

## 2. Bedah Layer (Sesuai Struktur Repository)

### A. Layer Domain (`internal/domain`)
Jantung dari aplikasi. Bebas dari *library* eksternal (tidak ada `sqlx`, `pgx`, `kratos`, atau `franz-go`).

*   **Model**: Berisi struct yang merepresentasikan entitas bisnis murni (contoh: `Order`, `Product`, `Stock`).
*   **Service (Domain Service)**: Logika rumit yang melintasi beberapa entitas (bukan CRUD biasa).

```go
// Contoh di internal/domain/model/order.go
type Order struct {
    ID     string
    UserID string
    Status string
}

func (o *Order) MarkAsPaid() error {
    if o.Status != "PENDING_PAYMENT" {
        return errors.New("hanya pesanan pending yang bisa dibayar")
    }
    o.Status = "PAID"
    return nil
}
```

### B. Layer Application (`internal/application`)
Menjembatani antara Dunia Luar (Adapter) dan Dunia Dalam (Domain).

*   **Ports**: Berisi definisi **Interface**. *Inbound Port* (didefinisikan secara implisit oleh signature *usecase*) dipanggil oleh *Controller* HTTP. *Outbound Port* dipanggil oleh *usecase* untuk berbicara dengan DB/Kafka.
*   **Usecase**: Mengorkestrasi logika. Memanggil fungsi Domain, lalu menyimpan hasilnya ke DB melalui Port.
*   **Saga**: Logika khusus untuk transisi State Machine (misal: bereaksi saat *Payment Failed*).

```go
// Contoh di internal/application/port/order_repository.go
type OrderRepository interface {
    Save(ctx context.Context, order *model.Order) error
    FindByID(ctx context.Context, id string) (*model.Order, error)
}

// Contoh di internal/application/usecase/checkout.go
type CheckoutUsecase struct {
    orderRepo port.OrderRepository
    kafkaPort port.EventPublisher
}

func (uc *CheckoutUsecase) Execute(ctx context.Context, req CheckoutRequest) error {
    // 1. Buat entity
    order := model.NewOrder(req.UserID)
    // 2. Simpan ke DB lewat Interface (TIDAK PEDULI apakah Postgres/MySQL/Mongo)
    err := uc.orderRepo.Save(ctx, order)
    return err
}
```

### C. Layer Adapter (`internal/adapter`)
Menghubungkan aplikasi kita dengan dunia nyata.

*   **Inbound (Driver)**: Menerima *request* dari luar. Contoh: `rest/handler.go` (go-kratos), `grpc/server.go`, `kafka/consumer.go`. *Inbound* memanggil *Usecase*.
*   **Outbound (Driven)**: Implementasi nyata dari *Interface* di layer Application.
    *   `postgres/order_repo_sqlx.go` (Menggunakan `sqlx` dan *raw* SQL).
    *   `redis/inventory_repo.go` (Menggunakan *Lua Script*).
    *   `kafka/producer.go` (Menggunakan `franz-go`).

```go
// Contoh implementasi di internal/adapter/outbound/postgres/order_repo_sqlx.go
type orderRepoSQLX struct {
    db *sqlx.DB
}

func NewOrderRepoSQLX(db *sqlx.DB) port.OrderRepository {
    return &orderRepoSQLX{db: db}
}

func (r *orderRepoSQLX) Save(ctx context.Context, order *model.Order) error {
    query := `INSERT INTO orders (id, user_id, status) VALUES ($1, $2, $3)`
    _, err := r.db.ExecContext(ctx, query, order.ID, order.UserID, order.Status)
    return err
}
```

## 3. Aturan Emas (DOs and DON'Ts)

✅ **LAKUKAN:**
*   Taruh definisi *interface* untuk database/messaging di *Application Port*.
*   Ubah DTO (*Data Transfer Object*) dari JSON (Inbound Adapter) menjadi parameter/struct biasa sebelum mengopernya ke Usecase.
*   Kembalikan *Domain Model* dari Usecase, biarkan Inbound Adapter mengubahnya menjadi JSON/gRPC Response sesuai `response-standard.md`.

❌ **JANGAN LAKUKAN:**
*   Mengembalikan *HTTP Status Code* (seperti 400, 404) dari *Usecase*. Usecase harus mengembalikan error Go (`errors.New("stok habis")`). Inbound Adapter yang bertugas menerjemahkannya ke HTTP 409.
*   Menggunakan *struct* yang digenerate oleh protobuf (gRPC) di dalam layer *Domain* atau *Application*. Struct gRPC hanya boleh hidup di *Inbound/Outbound Adapter*.
*   Membawa *database transaction* (seperti `*sqlx.Tx`) ke layer *Application*. Jika butuh transaksi panjang (seperti *Outbox Pattern*), gunakan fungsi pembungkus seperti *Transaction Runner Interface*.
