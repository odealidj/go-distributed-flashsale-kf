// Konfigurasi global untuk semua skenario k6
export const BASE_URL = __ENV.BASE_URL || 'http://localhost:8000';

// Produk Flash Sale ID yang akan diuji
export const PRODUCT_ID = __ENV.PRODUCT_ID || 'product-flashsale-001';

// Header simulasi user
export function randomUserToken() {
  return `user-${Math.floor(Math.random() * 100000)}`;
}

// Threshold standar: error rate < 1%, P95 < 500ms
export const DEFAULT_THRESHOLDS = {
  http_req_failed: ['rate<0.01'],
  http_req_duration: ['p(95)<500'],
};

// Tag untuk grouping di Grafana
export const TAGS = {
  service: 'api-gateway',
  scenario: 'flash-sale',
};
