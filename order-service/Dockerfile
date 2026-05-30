# Stage 1: Build binary using standard Golang environment
FROM docker.io/library/golang:1.26-alpine AS builder
RUN apk add --no-cache git
WORKDIR /app

# Copy the entire workspace into compilation context
COPY . .

# Compile the binary with optimizations (strip debug symbols for smaller size)
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o bin/order-service ./order-service/cmd/order-service

# Stage 2: Final production image
FROM docker.io/library/alpine:3.18
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app

# Copy compiled binary from builder stage
COPY --from=builder /app/bin/order-service .

# Run service (No EXPOSE port needed as order-service is a Kafka consumer/worker only)
CMD ["./order-service"]
