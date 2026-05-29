# Standar REST Response (API Gateway)

## 1. Tujuan
Semua REST API yang menghadap *frontend* (dari API Gateway / Reverse Proxy) harus mengembalikan *JSON envelope* yang konsisten untuk *success response* dan *error response*. Standar ini memudahkan klien (Mobile/Web) untuk mem-*parsing* data dan melacak *error* lintas *microservices*.

## 2. Success Response: Single Object

```json
{
  "success": true,
  "data": {
    "order_id": "ord_123",
    "status": "PENDING_PAYMENT"
  },
  "meta": {
    "request_id": "req_abc123",
    "trace_id": "7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e",
    "timestamp": "2026-05-29T10:00:00Z"
  }
}
```

**Aturan:**
*   `success` harus bernilai `true`.
*   `data` berisi *response payload* utama. **WAJIB** menggunakan DTO (Data Transfer Object) berupa *struct* Golang yang eksplisit (contoh: `CheckoutResponse`). **DILARANG** menggunakan *anonymous struct* atau `map[string]any` untuk menjaga keamanan tipe data (*type-safety*) dan dokumentasi OpenAPI yang akurat.
*   `meta` wajib ada (berguna untuk *Tracing* di Jaeger/ELK).
*   Kunci `error` tidak boleh ada di respons sukses.

## 3. Success Response: List (Pagination)

Untuk mengambil daftar produk Flash Sale:

```json
{
  "success": true,
  "data": [
    {
      "id": "prod_sepatu_01",
      "name": "Sepatu Lari X",
      "flashsale_price": 150000
    }
  ],
  "pagination": {
    "page": 1,
    "per_page": 20,
    "total_items": 100,
    "total_pages": 5,
    "has_next": true,
    "has_prev": false
  },
  "meta": {
    "request_id": "req_def456",
    "trace_id": "8a9b0c1d2e3f4a5b6c7d8e9f0a1b2c3d",
    "timestamp": "2026-05-29T10:05:00Z"
  }
}
```

## 4. Error Response (Bisnis / General)

Bila terjadi kegagalan bisnis (misal: stok habis) atau kegagalan internal (500).

```json
{
  "success": false,
  "error": {
    "code": "INSUFFICIENT_STOCK",
    "message": "Mohon maaf, stok sepatu lari sudah habis direbut orang lain.",
    "details": {
      "product_id": "prod_sepatu_01",
      "available_qty": 0
    }
  },
  "meta": {
    "request_id": "req_err999",
    "trace_id": "9f0a1b2c3d4e5f6a7b8c9d0e1f2a3b4c",
    "timestamp": "2026-05-29T10:06:00Z"
  }
}
```

**Aturan:**
*   `success` harus bernilai `false`.
*   `error.code` harus spesifik dan berformat *UPPERCASE_SNAKE_CASE* agar dikenali mesin/kode frontend.
*   `error.message` harus *human-readable* dan ramah pengguna.
*   *Internal stack trace* Golang **DILARANG KERAS** bocor ke dalam `details`.

## 5. Validation Error Response

Bila klien mengirim parameter yang salah (HTTP 400 Bad Request).

```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Validasi request gagal.",
    "fields": [
      {
        "field": "quantity",
        "message": "Jumlah pesanan maksimal 1 per pengguna saat Flash Sale."
      }
    ]
  },
  "meta": {
    "request_id": "req_val111",
    "trace_id": "0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b",
    "timestamp": "2026-05-29T10:07:00Z"
  }
}
```

## 6. Metadata Spesification

Setiap *response* harus menyertakan blok `meta`:

| Field | Deskripsi |
| :--- | :--- |
| `request_id` | ID unik per HTTP request (dari NGINX/Client). Berguna untuk investigasi *traffic* di lapis terluar. |
| `trace_id` | ID 128-bit unik (dari **OpenTelemetry**). Ini adalah *single source of truth* untuk mencari seluruh perjalanan *request* di **Jaeger**. |
| `timestamp` | Waktu selesai diproses dalam format RFC3339/ISO-8601. |

## 7. Standar Error Codes

| HTTP Status | Code | Keterangan |
| :--- | :--- | :--- |
| 400 | `VALIDATION_ERROR` | Input salah |
| 401 | `UNAUTHORIZED` | Token JWT tidak valid/hilang |
| 404 | `RESOURCE_NOT_FOUND` | Data tidak ada |
| 409 | `INSUFFICIENT_STOCK` | Stok Redis kosong |
| 409 | `USER_ALREADY_PURCHASED` | Aturan 1 user 1 barang dilanggar |
| 500 | `INTERNAL_ERROR` | Error tak terduga (misal DB down) |
| 503 | `SERVICE_UNAVAILABLE` | Trafik terlalu tinggi, menembus limit |
