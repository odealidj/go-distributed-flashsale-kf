import http from 'k6/http';
import { check, sleep } from 'k6';
import { uuidv4 } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';

export const options = {
  vus: 1,         // Gunakan 1 VU saja agar mudah membaca log koordinasi Saga
  iterations: 1,  // Hanya jalankan 1 kali untuk satu siklus pengujian
  tags: {
    test_type: 'compensation',
  },
};

const BASE_URL = 'http://localhost:18081'; // Nginx Proxy port dinamis lokal kita
const PRODUCT_ID = 'prod_1';

export default function () {
  const userID = `user-saga-${uuidv4().substring(0, 8)}`;
  
  // Payload REST API Gateway hanya membutuhkan product_id
  const checkoutPayload = JSON.stringify({
    product_id: PRODUCT_ID,
  });

  const checkoutParams = {
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${userID}`,
    },
  };

  console.log(`\n============== [Fase 1: Inisiasi Checkout] ==============`);
  console.log(`Mengirim checkout asinkron ke API Gateway...`);
  console.log(`User ID: ${userID} | Product ID: ${PRODUCT_ID}`);

  const checkoutRes = http.post(`${BASE_URL}/api/v1/checkout`, checkoutPayload, checkoutParams);

  const isAccepted = check(checkoutRes, {
    'Checkout direspon 202 Accepted': (r) => r.status === 202,
    'Pesan respon sesuai spec': (r) => r.json('meta.message') === 'pesanan sedang diproses',
    'Respon memiliki event_id': (r) => r.json('meta.event_id') !== undefined,
  });

  if (!isAccepted) {
    console.error(`Gagal melakukan checkout! Status: ${checkoutRes.status}, Body: ${checkoutRes.body}`);
    return;
  }

  const eventID = checkoutRes.json('meta.event_id');
  console.log(`✅ Checkout diterima! Event ID (Idempotency Key): ${eventID}`);
  console.log(`Saga sedang berjalan di latar belakang:`);
  console.log(`1. API Gateway ──(gRPC)──> Inventory Service (Mengurangi stok Redis)`);
  console.log(`2. Inventory Service ──(Kafka)──> Order Service (Membuat order PENDING)`);

  console.log(`\nMenunggu 3 detik agar Kafka memproses antrean...`);
  sleep(3);

  console.log(`\n============== [Fase 2: Catatan Verifikasi Manual] ==============`);
  console.log(`Karena transaksi checkout bersifat asinkron, order baru dengan ID unik (UUID)`);
  console.log(`telah dibuat di database 'db_order'.`);
  console.log(`\nUntuk menyelesaikan simulasi kegagalan pembayaran (Saga Kompensasi):`);
  console.log(`1. Cari Order ID di database Postgres 'db_order':`);
  console.log(`   SELECT id, total_amount FROM orders WHERE user_id = '${userID}';`);
  console.log(`2. Kirim permintaan pembayaran gagal (Amount berakhiran angka 4, misal: 150004):`);
  console.log(`   POST ${BASE_URL}/api/v1/pay`);
  console.log(`   Payload: { "order_id": "<ORDER_ID_DARI_LANGKAH_1>", "amount": 150004 }`);
  console.log(`3. Periksa log 'inventory-service' atau Redis 'stock:${PRODUCT_ID}'`);
  console.log(`   untuk memverifikasi bahwa stok berhasil dikembalikan secara otomatis (RefundStock).`);
  console.log(`=================================================================\n`);
}
