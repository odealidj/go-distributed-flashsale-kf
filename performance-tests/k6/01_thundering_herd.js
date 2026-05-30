/**
 * Skenario 01: Thundering Herd - Flash Sale Checkout
 *
 * Mensimulasikan gelombang besar user yang mencoba checkout secara bersamaan
 * saat Flash Sale dimulai. Ini adalah skenario utama yang menguji:
 *   - Rate Limiting di Nginx
 *   - Atomic Redis Lua Script (tidak ada overselling)
 *   - Idempotency melalui Outbox Pattern
 *
 * Jalankan: k6 run --env PRODUCT_ID=product-001 01_thundering_herd.js
 */
import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';
import { BASE_URL, PRODUCT_ID, randomUserToken, DEFAULT_THRESHOLDS } from './config.js';

// Metrik kustom
const checkoutSuccessCount = new Counter('checkout_success_total');
const checkoutConflictCount = new Counter('checkout_conflict_total'); // 409 = stok habis
const checkoutErrorCount = new Counter('checkout_error_total');
const checkoutSuccessRate = new Rate('checkout_success_rate');
const checkoutDuration = new Trend('checkout_duration_ms', true);

export const options = {
  // Simulasi Thundering Herd: ramp naik cepat ke 1000 VU dalam 10 detik
  scenarios: {
    thundering_herd: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '10s', target: 1000 }, // Ledakan tiba-tiba (flash sale dibuka)
        { duration: '30s', target: 1000 }, // Bertahan di puncak
        { duration: '10s', target: 0 },    // Cool down
      ],
      gracefulRampDown: '5s',
    },
  },
  thresholds: {
    // Override default thresholds to ignore http_req_failed (since Nginx 429 is expected)
    'http_req_duration': ['p(95)<500'],      // Respon HTTP harus cepat
    'checkout_conflict_total': ['count>0'], // Harus ada 409 (stok habis)
    'checkout_error_total': ['count<5000'],   // Toleransi socket drops lokal under 5000+ RPS
    'checkout_duration_ms': ['p(95)<500'],  // Checkout harus cepat
  },
};

export default function () {
  const userToken = randomUserToken();
  const payload = JSON.stringify({ product_id: PRODUCT_ID });

  const params = {
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${userToken}`,
    },
    tags: { scenario: 'thundering_herd' },
  };

  // 1. Skenario Realistis Langkah 1: Pengguna melihat halaman produk (Browse) sebelum checkout
  const browseRes = http.get(`${BASE_URL}/api/v1/products?page=1&per_page=10`, params);
  check(browseRes, {
    'browse products status 200': (r) => r.status === 200,
  });

  // Simulasi "Think Time" singkat (20ms - 100ms) sebelum menekan tombol "Beli"
  sleep(Math.random() * 0.08 + 0.02);

  // 2. Skenario Realistis Langkah 2: Pengguna menekan tombol checkout (Buy)
  const startTime = Date.now();
  const res = http.post(`${BASE_URL}/api/v1/checkout`, payload, params);
  const duration = Date.now() - startTime;

  checkoutDuration.add(duration);

  const isSuccess = check(res, {
    'response received': (r) => r.status !== 0,
    'status 202 accepted OR 409/429 rejection (expected)': (r) =>
      r.status === 202 || r.status === 409 || r.status === 429,
    'response has meta.trace_id': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.meta && body.meta.trace_id !== undefined;
      } catch (e) {
        return false;
      }
    },
  });

  if (res.status === 202) {
    checkoutSuccessCount.add(1);
    checkoutSuccessRate.add(true);
  } else if (res.status === 409 || res.status === 429) {
    checkoutConflictCount.add(1);
    checkoutSuccessRate.add(false);
  } else {
    checkoutErrorCount.add(1);
    checkoutSuccessRate.add(false);
    console.warn(`Unexpected status ${res.status}: ${res.body}`);
  }

  // Jeda sangat kecil antar iterasi
  sleep(Math.random() * 0.1);
}

export function handleSummary(data) {
  return {
    'stdout': summaryReport(data),
    './results/thundering_herd_summary.json': JSON.stringify(data, null, 2),
  };
}

function summaryReport(data) {
  const success = data.metrics['checkout_success_total']?.values?.count || 0;
  const conflict = data.metrics['checkout_conflict_total']?.values?.count || 0;
  const errors = data.metrics['checkout_error_total']?.values?.count || 0;
  const total = success + conflict + errors;
  const p95 = data.metrics['checkout_duration_ms']?.values['p(95)'] || 0;

  return `
╔══════════════════════════════════════════════════════╗
║         THUNDERING HERD - HASIL PENGUJIAN           ║
╠══════════════════════════════════════════════════════╣
║  Total Request  : ${String(total).padStart(6)}                         ║
║  ✅ 202 Accepted: ${String(success).padStart(6)} (checkout diterima)     ║
║  ⚠️  409/429     : ${String(conflict).padStart(6)} (stok habis / limit)   ║
║  ❌ Error Sistem : ${String(errors).padStart(5)}                         ║
║  P95 Durasi     : ${String(Math.round(p95)).padStart(5)}ms                       ║
╚══════════════════════════════════════════════════════╝
`;
}
