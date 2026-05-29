#!/bin/bash
for s in api-gateway product-service inventory-service order-service payment-service; do
  path="$s/cmd/$s/main.go"
  if ! grep -q '"context"' "$path"; then
    sed -i 's/import (/import (\n\t"context"/g' "$path"
  fi
  sed -i '/"go.opentelemetry.io\/otel"/d' "$path"
  sed -i '/"go.opentelemetry.io\/otel\/trace"/d' "$path"
  go fmt "$path"
done
