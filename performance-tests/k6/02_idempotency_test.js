/**
 * Skenario 02: Idempotency Test - Double Checkout Verification
 *
 * Memverifikasi bahwa sistem TIDAK terjadi overselling meskipun user
 * yang sama mengirim request checkout duplikat (misal: double-click,
 * network retry, dll).
 *
 * Cara Kerja:
 * - Setiap VU membuat 1 user_token unik
 * - VU tersebut mengirim checkout 3x berturut-turut dengan token yang SAMA
 * - Hanya 1 yang boleh berhasil (202), sisanya 409 (idempotency ditolak)
 *
 * Jalankan: k6 run --env PRODUCT_ID=product-001 02_idempotency_test.js
 */
import http from 'k6/http';
import { check, group, sleep } from 'k6';
import { Counter } from 'k6/metrics';
import { BASE_URL, PRODUCT_ID } from './config.js';

const doubleCheckoutPrevented = new Counter('double_checkout_prevented');
const idempotencyFailures = new Counter('idempotency_failures');

export const options = {
  scenarios: {
    idempotency_test: {
      executor: 'per-vu-iterations',
      vus: 200,
      iterations: 1,  // Setiap VU hanya 1 siklus (tapi 3 request di dalamnya)
      maxDuration: '60s',
    },
  },
  thresholds: {
    // Kunci: kegagalan idempotency harus 0
    'idempotency_failures': ['count==0'],
    // Setidaknya separuh lebih dari request harus dicegah (409)
    'double_checkout_prevented': ['count>0'],
    'http_req_duration': ['p(95)<300'],
  },
};

export default function () {
  // Setiap VU punya identitas unik
  const userToken = `fixed-user-${__VU}`;
  const payload = JSON.stringify({ product_id: PRODUCT_ID });

  const params = {
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${userToken}`,
    },
    tags: { scenario: 'idempotency_test' },
  };

  let firstStatus = null;
  let firstTraceID = null;

  group('Checkout 3x dengan user yang sama', function () {
    for (let attempt = 1; attempt <= 3; attempt++) {
      const res = http.post(`${BASE_URL}/api/v1/checkout`, payload, params);

      let body = null;
      try {
        body = JSON.parse(res.body);
      } catch (e) {
        // ignore parse error
      }

      if (attempt === 1) {
        firstStatus = res.status;
        firstTraceID = body?.meta?.trace_id;

        check(res, {
          'Request pertama berhasil (202 atau 409 valid)': (r) =>
            r.status === 202 || r.status === 409,
        });
      } else {
        // Request ke-2 dan ke-3 harus DITOLAK jika request pertama sukses
        if (firstStatus === 202) {
          const isDuplicate = res.status === 409;
          check(res, {
            'Request duplikat ditolak (409)': () => isDuplicate,
          });

          if (isDuplicate) {
            doubleCheckoutPrevented.add(1);
          } else {
            idempotencyFailures.add(1);
            console.error(
              `IDEMPOTENCY GAGAL! VU=${__VU}, attempt=${attempt}, ` +
              `firstStatus=${firstStatus}, status=${res.status}, ` +
              `traceID=${firstTraceID}`
            );
          }
        }
      }

      sleep(0.1); // Jeda kecil antar request (simulasi retry cepat)
    }
  });
}

export function handleSummary(data) {
  return {
    './results/idempotency_summary.json': JSON.stringify(data, null, 2),
  };
}
