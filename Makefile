.PHONY: infra-up infra-down infra-logs

# Menjalankan semua infrastruktur pendukung (Postgres, Redis, Kafka, Jaeger)
infra-up:
	docker-compose up -d

# Mematikan infrastruktur pendukung
infra-down:
	docker-compose down -v

# Melihat log infrastruktur
infra-logs:
	docker-compose logs -f
