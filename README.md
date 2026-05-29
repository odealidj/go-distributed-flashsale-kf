# ⚡ Go Distributed Flash Sale System

![Go Version](https://img.shields.io/badge/Go-1.21-00ADD8?style=flat-square&logo=go)
![Kafka](https://img.shields.io/badge/Kafka-Event%20Driven-E23528?style=flat-square&logo=apache-kafka)
![Redis](https://img.shields.io/badge/Redis-Atomic%20Locking-DC382D?style=flat-square&logo=redis)
![gRPC](https://img.shields.io/badge/gRPC-Internal%20RPC-244C5A?style=flat-square&logo=google)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-Outbox%20Pattern-336791?style=flat-square&logo=postgresql)

A production-grade, highly concurrent, and resilient distributed system designed to handle the **Thundering Herd** problem during E-commerce Flash Sales. Built entirely in Go, this project serves as an architectural showcase of modern distributed systems patterns, making it extremely resilient against massive sudden traffic spikes, network partitions, and database bottlenecks.

## 🌟 Architectural Highlights (Why this stands out)

This system is engineered with the rigor expected of senior-level engineering, avoiding naive CRUD implementations in favor of robust distributed patterns:

### 1. Hexagonal Architecture (Ports and Adapters)
Every microservice strictly follows Hexagonal Architecture, decoupling the core domain logic from infrastructural concerns (databases, message brokers, external APIs). This enables highly isolated unit testing and effortless dependency swapping.

### 2. Distributed Transactions (Saga Choreography)
Since Flash Sales span across Inventory, Order, and Payment services, maintaining ACID properties across microservices is impossible. This system uses an **Event-Driven Saga Pattern (Choreography)** via Kafka to maintain eventual consistency.
*   **Happy Path**: `Inventory Reserved` ➔ `Order Created` ➔ `Payment Processed`.
*   **Unhappy Path (Compensation)**: If payment fails or times out, an `OrderCancelledEvent` is triggered to atomically refund the stock via Redis Lua.

### 3. Absolute Data Consistency (Outbox Pattern)
To prevent the classic "Dual-Write Problem" (e.g., saving to Postgres succeeds but publishing to Kafka fails), this project implements the **Transactional Outbox Pattern**. Domain events are persisted atomically alongside business data in PostgreSQL, and an asynchronous Relay Worker reliably pushes them to Kafka, guaranteeing *at-least-once* delivery.

### 4. Zero Overselling (Atomic Redis Lua Scripts)
During a Flash Sale, traditional RDBMS row-level locking (`SELECT FOR UPDATE`) causes massive bottlenecks. Instead, inventory deduction is pushed to Redis using **Atomic Lua Scripts**. The script checks limits, deduplicates requests, and decrements stock in a single O(1) atomic operation.

### 5. High Resilience & Fault Tolerance
*   **Circuit Breakers**: Implemented via `sony/gobreaker` at the API Gateway to fast-fail internal gRPC requests and prevent cascading failures.
*   **Idempotency Mechanisms**: Strict idempotency keys are enforced both at the Redis cache layer and the PostgreSQL database layer (preventing double-charges or double-stock-deductions during network retries).
*   **Dead Letter Queues (DLQ) & Exponential Backoff**: Kafka Consumers are built with exponential backoff retries. Unresolvable events are routed to a DLQ for manual inspection, ensuring zero message loss.
*   **Pessimistic Locking for Workers**: Cron workers use PostgreSQL `FOR UPDATE SKIP LOCKED` to safely process expired orders concurrently across multiple pods without race conditions.

### 6. Deep Observability
Integrated with **OpenTelemetry** and **Jaeger**. Distributed Trace IDs are propagated synchronously via gRPC metadata and asynchronously via Kafka Headers (and bridged through the Outbox table), providing end-to-end visibility of a request's journey across the entire microservice ecosystem.

### 7. Performance & Load Tested
Comes with a comprehensive `k6` performance testing suite simulating real-world chaos:
*   **Thundering Herd**: Thousands of concurrent users checking out at exactly `T-0`.
*   **Soak Testing**: Sustained high load over a long duration to detect memory leaks.
*   **No-Oversell Validation**: Mathematically verifying that 150 requests for 100 items exactly result in 100 successful orders and 50 rejections.

---

## 🏗️ System Architecture

```text
User ➔ [ NGINX Rate Limiter ] ➔ [ API Gateway ]
                                      │
                 (gRPC Sync)          │          (gRPC Sync)
              ┌───────────────────────┴───────────────────────┐
              ▼                                               ▼
     [ Product Service ]                             [ Inventory Service ] ──(Lua)──➔ [ Redis ]
                                                              │
    (Outbox Relay)                                      (Outbox Relay)
          │                                                   │
          └─────────────────────➔ [ Kafka ] 🡄─────────────────┘
                                      │
               ┌──────────────────────┴──────────────────────┐
               ▼                                             ▼
       [ Order Service ]                            [ Payment Service ]
```

## 🛠️ Tech Stack

*   **Language**: Go 1.21
*   **Framework**: Go-Kratos (Microservice Framework)
*   **RPC**: gRPC & Protocol Buffers
*   **Broker**: Apache Kafka (franz-go)
*   **Database**: PostgreSQL (sqlx)
*   **Cache & Concurrency Control**: Redis (go-redis)
*   **Observability**: OpenTelemetry & Jaeger
*   **Testing**: Testify (Unit), Mockery (Mocks), k6 (Load Testing)
*   **CI/CD**: GitHub Actions

## 🚀 Getting Started

### Prerequisites
*   Docker & Docker Compose
*   Go 1.21+
*   `make`

### Running the Infrastructure
Start Postgres, Redis, Kafka, Zookeeper, and Jaeger:
```bash
docker-compose up -d
```

### Running the Microservices
*(Run these in separate terminal tabs)*
```bash
cd api-gateway && go run cmd/api-gateway/main.go
cd product-service && go run cmd/product-service/main.go
cd inventory-service && go run cmd/inventory-service/main.go
cd order-service && go run cmd/order-service/main.go
cd payment-service && go run cmd/payment-service/main.go
```

### Running Load Tests
```bash
k6 run k6/01_thundering_herd_test.js
```

---

*This project was developed to demonstrate mastery in backend engineering, distributed system design, and Go programming.*
