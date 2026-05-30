ifneq (,$(wildcard ./.env))
    include .env
    export
endif

.PHONY: infra-up infra-down infra-logs proto
.PHONY: run-api-gateway run-product run-product-service run-inventory run-inventory-service run-order run-order-service run-payment run-payment-service
.PHONY: stop-api-gateway stop-product stop-product-service stop-inventory stop-inventory-service stop-order stop-order-service stop-payment stop-payment-service
.PHONY: run-all stop-all up down

# ==============================================================================
# INFRASTRUCTURE (Docker Compose)
# ==============================================================================

# Menjalankan HANYA infrastruktur pendukung (Postgres, Redis, Kafka, Jaeger, Nginx)
# Catatan: Kontainer aplikasi Go diabaikan karena berada di bawah profil 'app'
infra-up:
	@echo "Menyalakan HANYA kontainer infrastruktur pendukung..."
	docker-compose up -d

# Mematikan HANYA infrastruktur pendukung
infra-down:
	@echo "Mematikan kontainer infrastruktur pendukung..."
	docker-compose down -v

# Melihat log infrastruktur
infra-logs:
	docker-compose logs -f


# ==============================================================================
# MICROSERVICES (Docker Container Orchestration)
# ==============================================================================

run-api-gateway:
	@echo "Membangun & menjalankan kontainer API Gateway..."
	docker-compose --profile app up -d --build api-gateway

stop-api-gateway:
	@echo "Mematikan & menghapus kontainer API Gateway..."
	docker-compose --profile app rm -fs api-gateway

run-product run-product-service:
	@echo "Membangun & menjalankan kontainer Product Service..."
	docker-compose --profile app up -d --build product-service

stop-product stop-product-service:
	@echo "Mematikan & menghapus kontainer Product Service..."
	docker-compose --profile app rm -fs product-service

run-inventory run-inventory-service:
	@echo "Membangun & menjalankan kontainer Inventory Service..."
	docker-compose --profile app up -d --build inventory-service

stop-inventory stop-inventory-service:
	@echo "Mematikan & menghapus kontainer Inventory Service..."
	docker-compose --profile app rm -fs inventory-service

run-order run-order-service:
	@echo "Membangun & menjalankan kontainer Order Service..."
	docker-compose --profile app up -d --build order-service

stop-order stop-order-service:
	@echo "Mematikan & menghapus kontainer Order Service..."
	docker-compose --profile app rm -fs order-service

run-payment run-payment-service:
	@echo "Membangun & menjalankan kontainer Payment Service..."
	docker-compose --profile app up -d --build payment-service

stop-payment stop-payment-service:
	@echo "Mematikan & menghapus kontainer Payment Service..."
	docker-compose --profile app rm -fs payment-service


# ==============================================================================
# BATCH COMMANDS
# ==============================================================================

# Menjalankan seluruh microservices Go sebagai kontainer
run-all:
	@echo "Membangun & menjalankan SELURUH kontainer microservices Go..."
	docker-compose --profile app up -d --build api-gateway product-service inventory-service order-service payment-service

# Mematikan seluruh microservices Go kontainer
stop-all:
	@echo "Mematikan & menghapus SELURUH kontainer microservices Go..."
	docker-compose --profile app rm -fs api-gateway product-service inventory-service order-service payment-service

# Menyalakan keseluruhan sistem (Infra + Microservices Kontainer) secara bersih
up:
	@echo "Menyalakan keseluruhan sistem (Infrastruktur + Aplikasi Go Kontainer)..."
	docker-compose --profile app up -d --build

# Mematikan keseluruhan sistem secara bersih dan menghapus volume
down:
	@echo "Mematikan keseluruhan sistem secara bersih..."
	docker-compose --profile app down -v


# ==============================================================================
# UTILS
# ==============================================================================

# Men-generate kode Go dari file .proto
proto:
	cd proto && protoc --go_out=paths=source_relative:. \
	       --go-grpc_out=paths=source_relative:. \
	       inventory/v1/inventory.proto \
	       order/v1/order.proto \
	       payment/v1/payment.proto \
	       product/v1/product.proto
