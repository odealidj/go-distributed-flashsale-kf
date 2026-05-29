import http from 'k6/http';
import { check, sleep } from 'k6';
import { uuidv4 } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';

export const options = {
    vus: 1, // Kita pakai 1 VUs saja untuk mempermudah observasi log
    iterations: 1, // Hanya jalankan 1 kali
    tags: {
        test_type: 'compensation',
    },
};

const BASE_URL = 'http://localhost:8000';

export default function () {
    // Harga sengaja diset agar total amount berakhiran angka 4
    // (misal 994), agar Payment Service men-trigger "FAILED"
    const payload = JSON.stringify({
        user_id: uuidv4(),
        product_id: 'prod_1',
        quantity: 1,
        price: 994, 
    });

    const params = {
        headers: {
            'Content-Type': 'application/json',
        },
    };

    console.log(`Sending checkout request (EXPECT PAYMENT FAIL): ${payload}`);
    const res = http.post(`${BASE_URL}/api/v1/checkout`, payload, params);

    check(res, {
        'status is 200': (r) => r.status === 200,
        'response contains success message': (r) => r.json('message') === 'Checkout process started',
    });

    // Karena proses Saga bersifat asynchronous,
    // Kita berikan waktu sekitar 5 detik untuk memastikan pesan sampai dari Order -> Payment -> Order -> Inventory (Refund)
    sleep(5);
    
    // Idealnya, kita membuat endpoint GET /api/v1/stock/prod_1 untuk mengecek apakah
    // stock kembali ke 100. Namun karena tidak ada endpoint GET stock di spec awal,
    // Pengecekan bisa dilakukan secara manual melalui logs atau Jaeger.
    console.log('Test completed. Please check Inventory Service logs to verify RefundStock was executed.');
}
