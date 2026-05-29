.PHONY: infra-up infra-down infra-logs proto

# Men-generate kode Go dari file .proto
proto:
	cd proto && protoc --go_out=paths=source_relative:. \
	       --go-grpc_out=paths=source_relative:. \
	       inventory/v1/inventory.proto \
	       order/v1/order.proto \
	       payment/v1/payment.proto \
	       product/v1/product.proto


# Menjalankan semua infrastruktur pendukung (Postgres, Redis, Kafka, Jaeger)
infra-up:
	docker-compose up -d

# Mematikan infrastruktur pendukung
infra-down:
	docker-compose down -v

# Melihat log infrastruktur
infra-logs:
	docker-compose logs -f
