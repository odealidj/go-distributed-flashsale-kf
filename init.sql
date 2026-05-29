-- init.sql
-- Ini adalah script inisialisasi awal database (Hanya untuk demo/scaffold).
-- Di production, sebaiknya menggunakan alat migrasi (seperti golang-migrate).

CREATE TABLE IF NOT EXISTS products (
    id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    original_price BIGINT NOT NULL,
    flash_sale_price BIGINT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_by VARCHAR(100),
    version INTEGER DEFAULT 1
);

CREATE TABLE IF NOT EXISTS inventories (
    product_id VARCHAR(50) PRIMARY KEY,
    stock BIGINT NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_by VARCHAR(100),
    version INTEGER DEFAULT 1
);

CREATE TABLE IF NOT EXISTS outbox_messages (
    id SERIAL PRIMARY KEY,
    aggregate_id VARCHAR(255) NOT NULL,
    aggregate_type VARCHAR(255) NOT NULL,
    event_type VARCHAR(255) NOT NULL,
    payload JSONB NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Masukkan dummy data awal jika tabel kosong
INSERT INTO products (id, name, original_price, flash_sale_price, updated_by)
VALUES 
    ('prod_1', 'Sepatu Lari X', 500000, 150000, 'system'),
    ('prod_2', 'Tas Ransel Y', 300000, 99000, 'system')
ON CONFLICT (id) DO NOTHING;

INSERT INTO inventories (product_id, stock, updated_by)
VALUES 
    ('prod_1', 100, 'system'),
    ('prod_2', 50, 'system')
ON CONFLICT (product_id) DO NOTHING;
