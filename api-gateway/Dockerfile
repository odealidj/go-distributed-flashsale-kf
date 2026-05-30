# Stage 1: Build binary using standard Golang environment
FROM docker.io/library/golang:1.26-alpine AS builder
RUN apk add --no-cache git
WORKDIR /app

# Copy the entire workspace into compilation context
COPY . .

# Compile the binary with optimizations (strip debug symbols for smaller size)
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o bin/api-gateway ./api-gateway/cmd/api-gateway

# Stage 2: Final production image
FROM docker.io/library/alpine:3.18
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app

# Copy compiled binary from builder stage
COPY --from=builder /app/bin/api-gateway .

# Expose internal service port
EXPOSE 8000

# Run service
CMD ["./api-gateway"]
