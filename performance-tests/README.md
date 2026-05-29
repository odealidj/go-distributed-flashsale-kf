# Performance Testing - Flash Sale k6

Folder ini berisi semua script uji performa untuk sistem Flash Sale.

## Prasyarat

1. **k6** sudah terinstall:
   ```bash
   # Linux
   sudo gpg -k
   sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg \
     --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
   echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" \
     | sudo tee /etc/apt/sources.list.d/k6.list
   sudo apt-get update && sudo apt-get install k6
   ```

2. **Sistem berjalan** (`docker-compose up -d`)

3. **Redis & semua service aktif** (cek `docker ps`)

---

## Deskripsi Skenario

| File | Skenario | VU | Durasi | Tujuan |
|------|----------|----|--------|--------|
| `01_thundering_herd.js` | Thundering Herd | 0→1000 | ~50s | Uji ketahanan lonjakan tiba-tiba |
| `02_idempotency_test.js` | Idempotency | 200 | ~60s | Verifikasi tidak ada double-checkout |
| `03_soak_test.js` | Soak | 100 | 30m | Deteksi memory leak & degradasi |
| `04_no_oversell.js` | No-Oversell | 5000 | ~2m | **Golden test**: jumlah terjual ≤ stok |

---

## Cara Menjalankan

### Setup Stok Awal
```bash
# Set 100 unit stok di Redis
cd performance-tests
PRODUCT_ID=product-flashsale-001 INITIAL_STOCK=100 bash setup_stock.sh
```

### Jalankan 1 Skenario
```bash
cd performance-tests

# Thundering Herd
k6 run --env PRODUCT_ID=product-flashsale-001 k6/01_thundering_herd.js

# Idempotency Test
k6 run --env PRODUCT_ID=product-flashsale-001 k6/02_idempotency_test.js

# Soak Test (30 menit)
k6 run --env PRODUCT_ID=product-flashsale-001 k6/03_soak_test.js

# No-Oversell (golden test)
k6 run --env PRODUCT_ID=product-flashsale-001 --env INITIAL_STOCK=100 k6/04_no_oversell.js
```

### Jalankan Semua (kecuali Soak)
```bash
cd performance-tests
PRODUCT_ID=product-flashsale-001 INITIAL_STOCK=100 bash run_all.sh
```

---

## Interpretasi Hasil

### Thundering Herd
- **202 Accepted**: Checkout diterima sistem
- **409 Conflict**: Stok habis — ini **PERILAKU YANG BENAR**
- **429 Too Many Requests**: Rate limit Nginx aktif — ini **PERILAKU YANG BENAR**
- **5xx**: Error sistem — **harus 0**

### Idempotency Test
- Metric `idempotency_failures` **harus = 0**
- Artinya tidak ada user yang berhasil checkout dua kali

### No-Oversell (Golden Test)
- `successful_checkout_count` **harus ≤ INITIAL_STOCK**
- Laporan otomatis muncul di terminal setelah selesai

---

## Integrasi dengan Observability

Setiap request melewati Nginx → API Gateway → Redis/Postgres.  
Trace ID ada di header dan body response (`meta.trace_id`).  
Buka **Jaeger UI** di `http://localhost:16686` untuk melihat trace selama test berlangsung.
