# Strategi Testing: Flash Sale Microservices

Dalam sistem yang kompleks, *testing* berlapis adalah keharusan. Kita akan menggunakan piramida *testing*.

## 1. Unit Testing (Level Usecase & Domain)
- **Fokus**: Logika bisnis (Hexagonal Inner Core).
- **Tools**: `testing` bawaan Go dan `github.com/stretchr/testify` untuk *mocking*.
- **Contoh**: Mengetes `OrderUsecase` dengan me-*mock* `InventoryPort`. Kita memverifikasi bahwa `OrderUsecase` tidak memanggil `CreateOrder` di database jika `InventoryPort` mengembalikan error "Stok Habis".

## 2. Integration Testing (Level Adapter / Repository)
- **Fokus**: Interaksi dengan infrastruktur eksternal.
- **Tools**: **Testcontainers-Go**.
- **Contoh 1 (Redis)**: Menggulirkan *container* Redis dinamis, mengeksekusi *Lua Script* pemotongan stok berulang kali secara *goroutine*, dan memverifikasi hasilnya tidak pernah minus.
- **Contoh 2 (PostgreSQL)**: Mengetes kueri SQL di *container* Postgres dinamis.

## 3. End-to-End (E2E) Testing (Level Sistem)
- **Fokus**: Memastikan *Saga Pattern* berjalan dari ujung ke ujung.
- **Skenario**: 
  1. Panggil `/checkout` -> Verifikasi respons HTTP 202.
  2. Tunggu beberapa detik. Panggil `/order/{id}` -> Verifikasi status menjadi `PENDING_PAYMENT`.
  3. Panggil `/pay` -> Verifikasi pesanan berubah menjadi `PAID`.
- **Tools**: Go HTTP client biasa yang mengeksekusi skenario layaknya *user* sesungguhnya.
