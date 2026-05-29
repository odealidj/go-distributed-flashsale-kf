.PHONY: infra-up infra-down infra-logs proto
.PHONY: run-api-gateway run-product run-inventory run-order run-payment
.PHONY: stop-api-gateway stop-product stop-inventory stop-order stop-payment
.PHONY: run-all stop-all up down

# ==============================================================================
# INFRASTRUCTURE (Docker Compose)
# ==============================================================================

# Menjalankan semua infrastruktur pendukung (Postgres, Redis, Kafka, Jaeger)
infra-up:
	docker-compose up -d

# Mematikan infrastruktur pendukung
infra-down:
	docker-compose down -v

# Melihat log infrastruktur
infra-logs:
	docker-compose logs -f


# ==============================================================================
# MICROSERVICES (Go Run Background)
# ==============================================================================

run-api-gateway:
	@echo "Starting API Gateway..."
	@nohup go run api-gateway/cmd/api-gateway/main.go > api-gateway.log 2>&1 & echo $$! > api-gateway.pid
	@echo "API Gateway started."

stop-api-gateway:
	@echo "Stopping API Gateway..."
	@-kill `cat api-gateway.pid 2>/dev/null` 2>/dev/null || true
	@rm -f api-gateway.pid

run-product:
	@echo "Starting Product Service..."
	@nohup go run product-service/cmd/product-service/main.go > product.log 2>&1 & echo $$! > product.pid
	@echo "Product Service started."

stop-product:
	@echo "Stopping Product Service..."
	@-kill `cat product.pid 2>/dev/null` 2>/dev/null || true
	@rm -f product.pid

run-inventory:
	@echo "Starting Inventory Service..."
	@nohup go run inventory-service/cmd/inventory-service/main.go > inventory.log 2>&1 & echo $$! > inventory.pid
	@echo "Inventory Service started."

stop-inventory:
	@echo "Stopping Inventory Service..."
	@-kill `cat inventory.pid 2>/dev/null` 2>/dev/null || true
	@rm -f inventory.pid

run-order:
	@echo "Starting Order Service..."
	@nohup go run order-service/cmd/order-service/main.go > order.log 2>&1 & echo $$! > order.pid
	@echo "Order Service started."

stop-order:
	@echo "Stopping Order Service..."
	@-kill `cat order.pid 2>/dev/null` 2>/dev/null || true
	@rm -f order.pid

run-payment:
	@echo "Starting Payment Service..."
	@nohup go run payment-service/cmd/payment-service/main.go > payment.log 2>&1 & echo $$! > payment.pid
	@echo "Payment Service started."

stop-payment:
	@echo "Stopping Payment Service..."
	@-kill `cat payment.pid 2>/dev/null` 2>/dev/null || true
	@rm -f payment.pid


# ==============================================================================
# BATCH COMMANDS
# ==============================================================================

# Menjalankan seluruh microservices
run-all: run-product run-inventory run-payment run-order run-api-gateway
	@echo "All microservices started."

# Mematikan seluruh microservices
stop-all: stop-api-gateway stop-order stop-payment stop-inventory stop-product
	@echo "All microservices stopped."

# Menyalakan keseluruhan sistem (Infra + Microservices)
up: infra-up
	@echo "Waiting for infra to be ready..."
	@sleep 5
	@$(MAKE) run-all

# Mematikan keseluruhan sistem (Microservices + Infra)
down: stop-all infra-down


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
