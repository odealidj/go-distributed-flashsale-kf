#!/bin/bash
# =======================================================
# Script: Setup Stok Awal untuk Performance Testing
# Digunakan sebelum menjalankan k6 test
# =======================================================

set -e

REDIS_HOST="${REDIS_HOST:-localhost}"
REDIS_PORT="${REDIS_PORT:-6379}"
PRODUCT_ID="${PRODUCT_ID:-product-flashsale-001}"
INITIAL_STOCK="${INITIAL_STOCK:-100}"

echo "🚀 Setting up Flash Sale Stock..."
echo "   Redis    : ${REDIS_HOST}:${REDIS_PORT}"
echo "   ProductID: ${PRODUCT_ID}"
echo "   Stok Awal: ${INITIAL_STOCK} unit"

# Reset dan set stok di Redis via kontainer Docker
docker exec -i flashsale-redis redis-cli FLUSHALL
docker exec -i flashsale-redis redis-cli SET "stock:${PRODUCT_ID}" "$INITIAL_STOCK"

echo "✅ Stok berhasil di-set: stock:${PRODUCT_ID} = ${INITIAL_STOCK}"
echo ""
echo "📊 Verifikasi stok:"
docker exec -i flashsale-redis redis-cli GET "stock:${PRODUCT_ID}"
