# Domain Context & Architecture Diagram

Dokumen ini memvisualisasikan batas domain (*domain boundaries*), aliran komunikasi, dan dependensi antar *microservices* dalam sistem Flash Sale. Pendekatan ini terinspirasi dari C4 Model (Container Level) dan Domain-Driven Design (DDD).

---

## 1. Top-Level System Architecture
Ini adalah pandangan "helikopter" dari seluruh sistem, menunjukkan bagaimana klien berinteraksi dengan API Gateway yang kemudian mendistribusikan beban ke layanan *backend*.

```mermaid
flowchart LR
    Cust((Mobile/Web<br/>Customer))
    subgraph FlashSaleSystem["Flash Sale System"]
        GW[API Gateway]
        subgraph BackendServices["Backend Services"]
            Prod[Product Service]
            Inv[Inventory Service]
            Ord[Order Service]
            Pay[Payment Service]
        end
        Kafka[[Kafka Event Broker]]
        Datastore[(Datastores<br/>Redis & Postgres)]
    end
    ExtBank[3rd Party<br/>Payment Gateway]
    Cust <-->|HTTPS / REST| GW
    GW <-->|gRPC| Prod
    GW <-->|gRPC| Inv
    GW <-->|gRPC| Ord
    GW <-->|gRPC| Pay
    Prod -.->|Async Pub/Sub| Kafka
    Inv -.->|Async Pub/Sub| Kafka
    Ord -.->|Async Pub/Sub| Kafka
    Pay -.->|Async Pub/Sub| Kafka
    Prod <-->|R/W| Datastore
    Inv <-->|R/W| Datastore
    Ord <-->|R/W| Datastore
    Pay <-->|R/W| Datastore
    Pay <-->|HTTPS| ExtBank
```

---

## 2. Identifikasi Domain: Inventory Service (The Core)
Inventory Service adalah layanan paling kritis. Diagram ini menunjukkan siapa yang memanggil layanan ini secara langsung (Public/Internal API) dan bagaimana ia berinteraksi dengan *storage*.

```mermaid
flowchart LR
    GW[API Gateway]
    OrderSvc[Order Service]
    subgraph InventoryDomain["Inventory Domain"]
        InvSvc[Inventory Service]
        Redis[(Redis Cache<br/>Atomic Counter)]
        InvDB[(Inventory<br/>Postgres DB)]
    end
    GW -->|gRPC Internal Call<br/>/checkout| InvSvc
    OrderSvc -.->|Kafka Event<br/>OrderCancelledEvent| InvSvc
    InvSvc <-->|Lua Scripts Atomic| Redis
    InvSvc <-->|Async Sync / Permanent| InvDB
```
*Catatan:* Gateway melakukan pemanggilan internal gRPC secara sinkron untuk memotong stok di Redis. Jika sukses, Inventory menembakkan event ke Kafka.

---

## 3. Identifikasi Domain: Order Service
Order Service bertindak sebagai "buku catatan". Layanan ini sangat digerakkan oleh *event* (Event-Driven) dan jarang dipanggil untuk melakukan mutasi data secara sinkron.

```mermaid
flowchart LR
    GW[API Gateway]
    InvSvc[Inventory Service]
    PaySvc[Payment Service]
    subgraph OrderDomain["Order Domain"]
        OrdSvc[Order Service]
        OrdDB[(Order<br/>Postgres DB)]
    end
    GW -->|gRPC Public API<br/>/orders id| OrdSvc
    InvSvc -.->|Kafka Event<br/>StockReservedEvent| OrdSvc
    PaySvc -.->|Kafka Event<br/>PaymentCompletedEvent| OrdSvc
    OrdSvc <-->|Read/Write| OrdDB
```
*Catatan:* Order Service membuat baris pesanan `PENDING_PAYMENT` hanya ketika mendengar `StockReservedEvent` dari Inventory. Status berubah menjadi `PAID` ketika mendengar `PaymentCompletedEvent`.

---

## 4. Identifikasi Domain: Payment Service
Payment Service bertugas sebagai jembatan antara sistem internal dan Payment Gateway eksternal.

```mermaid
flowchart LR
    GW[API Gateway]
    subgraph PaymentDomain["Payment Domain"]
        PaySvc[Payment Service]
        PayDB[(Payment<br/>Postgres DB)]
    end
    ExtBank[External<br/>Payment Gateway]
    OrderSvc[Order Service]
    GW -->|gRPC Public API<br/>/pay| PaySvc
    PaySvc <-->|HTTPS API| ExtBank
    PaySvc <-->|Record Log| PayDB
    PaySvc -.->|Kafka Event<br/>PaymentCompleted| OrderSvc
```

---

## 5. Identifikasi Domain: Product Service
Product Service adalah layanan *read-heavy* yang menampilkan katalog.

```mermaid
flowchart LR
    GW[API Gateway]
    Admin((Internal Admin))
    subgraph ProductDomain["Product Domain"]
        ProdSvc[Product Service]
        Redis[(Redis Cache<br/>Product Info)]
        ProdDB[(Product<br/>Postgres DB)]
    end
    GW -->|gRPC Public API<br/>/products| ProdSvc
    Admin -->|gRPC Internal<br/>/products/add| ProdSvc
    ProdSvc <-->|Cache Read 99pct| Redis
    ProdSvc <-->|DB Read/Write 1pct| ProdDB
```
