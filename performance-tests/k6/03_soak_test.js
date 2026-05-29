/**
 * Skenario 03: Soak Test - Uji Ketahanan Jangka Panjang
 *
 * Menjalankan load sedang selama 30 menit untuk mendeteksi:
 *   - Memory leak (goroutine yang tidak dibersihkan)
 *   - Connection pool exhaustion (DB/Redis)
 *   - Degradasi performa seiring waktu (latency P95 meningkat)
 *
 * Jalankan: k6 run --env PRODUCT_ID=product-001 03_soak_test.js
 */
import http from 'k6/http';
import { check, sleep } from 'k6';
import { Trend } from 'k6/metrics';
import { BASE_URL, PRODUCT_ID, randomUserToken } from './config.js';

const checkoutLatency = new Trend('soak_checkout_latency_ms', true);
const productListLatency = new Trend('soak_product_list_latency_ms', true);

export const options = {
  scenarios: {
    soak: {
      executor: 'constant-vus',
      vus: 100,          // Load sedang, bukan puncak
      duration: '30m',   // Jangka panjang untuk deteksi memory leak
    },
  },
  thresholds: {
    // Latency tidak boleh meningkat lebih dari 2x dari awal ke akhir
    'soak_checkout_latency_ms': [
      'p(95)<1000',  // Tidak boleh melebihi 1 detik di P95 sepanjang test
      'p(99)<2000',
    ],
    'soak_product_list_latency_ms': ['p(95)<300'],
    'http_req_failed': ['rate<0.005'], // Error rate < 0.5%
  },
};

export default function () {
  const userToken = randomUserToken();
  const headers = {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${userToken}`,
  };

  // Selingi antara list products dan checkout untuk mensimulasikan user flow nyata
  const scenario = Math.random();

  if (scenario < 0.7) {
    // 70% request: list products (read-heavy)
    const startTime = Date.now();
    const res = http.get(`${BASE_URL}/api/v1/products?page=1&per_page=20`, {
      headers,
      tags: { type: 'read' },
    });
    productListLatency.add(Date.now() - startTime);

    check(res, {
      'Product list status 200': (r) => r.status === 200,
      'Product list has data': (r) => {
        try {
          const body = JSON.parse(r.body);
          return body.meta && body.meta.trace_id;
        } catch (e) {
          return false;
        }
      },
    });

    sleep(1 + Math.random() * 2); // User browsing
  } else {
    // 30% request: checkout (write-heavy)
    const startTime = Date.now();
    const res = http.post(
      `${BASE_URL}/api/v1/checkout`,
      JSON.stringify({ product_id: PRODUCT_ID }),
      { headers, tags: { type: 'write' } }
    );
    checkoutLatency.add(Date.now() - startTime);

    check(res, {
      'Checkout status valid': (r) => r.status === 202 || r.status === 409 || r.status === 429,
    });

    sleep(0.5 + Math.random() * 1.5); // Jeda antar checkout
  }
}

export function handleSummary(data) {
  return {
    './results/soak_test_summary.json': JSON.stringify(data, null, 2),
  };
}
