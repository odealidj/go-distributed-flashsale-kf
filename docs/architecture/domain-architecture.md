# Domain Context & Architecture Diagram

Dokumen ini memvisualisasikan batas domain (*domain boundaries*), aliran komunikasi, dan dependensi antar *microservices* dalam sistem Flash Sale. Pendekatan ini terinspirasi dari C4 Model (Container Level) dan Domain-Driven Design (DDD).

---

## 1. Top-Level System Architecture
Ini adalah pandangan "helikopter" dari seluruh sistem, menunjukkan bagaimana klien berinteraksi dengan API Gateway yang kemudian mendistribusikan beban ke layanan *backend*.

```mermaid
flowchart LR
    %% Aktor
    Cust((Mobile/Web \n Customer))
    
    %% Sistem Utama
    subgraph "Flash Sale System"
        GW[API Gateway]
        
        subgraph "Backend Services"
            Prod[Product Service]
            Inv[Inventory Service]
            Ord[Order Service]
            Pay[Payment Service]
        end
        
        %% Infrastruktur
        Kafka[[Kafka Event Broker]]
        Datastore[(Datastores \n Redis & Postgres)]
    end
    
    %% External
    ExtBank[3rd Party \n Payment Gateway]

    %% Koneksi Level Atas
    Cust <-->|HTTPS / REST| GW
    GW <-->|gRPC| Backend Services
    
    Backend Services -.->|Async Pub/Sub| Kafka
    Backend Services <-->|R/W| Datastore
    
    Pay <-->|HTTPS| ExtBank
```

---

## 2. Identifikasi Domain: Inventory Service (The Core)
Inventory Service adalah layanan paling kritis. Diagram ini menunjukkan siapa yang memanggil layanan ini secara langsung (Public/Internal API) dan bagaimana ia berinteraksi dengan *storage*.

```mermaid
flowchart LR
    %% Pemanggil
    GW[API Gateway]
    OrderSvc[Order Service]
    
    %% Domain Inti
    subgraph "Inventory Domain"
        InvSvc[Inventory Service]
        Redis[(Redis Cache \n Atomic Counter)]
        InvDB[(Inventory \n Postgres DB)]
    end
    
    %% Koneksi
    GW -->|gRPC (Internal Call) \n /checkout| InvSvc
    OrderSvc -.->|Kafka Event \n OrderCancelledEvent| InvSvc
    
    InvSvc <-->|Lua Scripts (Microsec)| Redis
    InvSvc <-->|Async Sync / Permanent| InvDB
```
*Catatan:* Gateway melakukan pemanggilan internal gRPC secara sinkron untuk memotong stok di Redis. Jika sukses, Inventory menembakkan event ke Kafka.

---

## 3. Identifikasi Domain: Order Service
Order Service bertindak sebagai "buku catatan". Layanan ini sangat digerakkan oleh *event* (Event-Driven) dan jarang dipanggil untuk melakukan mutasi data secara sinkron.

```mermaid
flowchart LR
    %% Pemanggil
    GW[API Gateway]
    InvSvc[Inventory Service]
    PaySvc[Payment Service]
    
    %% Domain Inti
    subgraph "Order Domain"
        OrdSvc[Order Service]
        OrdDB[(Order \n Postgres DB)]
    end
    
    %% Koneksi
    GW -->|gRPC (Public API via GW) \n /orders/{id}| OrdSvc
    
    InvSvc -.->|Kafka Event \n StockReservedEvent| OrdSvc
    PaySvc -.->|Kafka Event \n PaymentCompletedEvent| OrdSvc
    
    OrdSvc <-->|Read/Write| OrdDB
```
*Catatan:* Order Service membuat baris pesanan `PENDING_PAYMENT` hanya ketika mendengar `StockReservedEvent` dari Inventory. Status berubah menjadi `PAID` ketika mendengar `PaymentCompletedEvent`.

---

## 4. Identifikasi Domain: Payment Service
Payment Service bertugas sebagai jembatan antara sistem internal dan Payment Gateway eksternal.

```mermaid
flowchart LR
    %% Pemanggil
    GW[API Gateway]
    
    %% Domain Inti
    subgraph "Payment Domain"
        PaySvc[Payment Service]
        PayDB[(Payment \n Postgres DB)]
    end
    
    %% External
    ExtBank[External \n Payment Gateway]
    OrderSvc[Order Service]
    
    %% Koneksi
    GW -->|gRPC (Public API via GW) \n /pay| PaySvc
    
    PaySvc <-->|HTTPS API| ExtBank
    PaySvc <-->|Record Log| PayDB
    PaySvc -.->|Kafka Event \n PaymentCompleted| OrderSvc
```

---

## 5. Identifikasi Domain: Product Service
Product Service adalah layanan *read-heavy* yang menampilkan katalog.

```mermaid
flowchart LR
    %% Pemanggil
    GW[API Gateway]
    Admin((Internal Admin))
    
    %% Domain Inti
    subgraph "Product Domain"
        ProdSvc[Product Service]
        Redis[(Redis Cache \n Product Info)]
        ProdDB[(Product \n Postgres DB)]
    end
    
    %% Koneksi
    GW -->|gRPC (Public API via GW) \n /products| ProdSvc
    Admin -->|gRPC (Internal) \n /products/add| ProdSvc
    
    ProdSvc <-->|Cache Read (99%)| Redis
    ProdSvc <-->|DB Read/Write (1%)| ProdDB
```
