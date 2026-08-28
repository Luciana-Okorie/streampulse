import http from 'k6/http';
import { check, sleep } from 'k6';

// Run with: k6 run --vus 100 --duration 60s loadtest/k6-script.js
// 100 VUs firing roughly every 100ms ≈ 1,000 events/sec.

const API_URL = __ENV.API_URL || 'http://localhost:4002/events';

const EVENT_TYPES = [
  'user.login', 'user.logout', 'order.created',
  'payment.success', 'payment.failed', 'api.request', 'api.error',
];

export const options = {
  scenarios: {
    steady_load: {
      executor: 'constant-arrival-rate',
      rate: 1000,
      timeUnit: '1s',
      duration: '60s',
      preAllocatedVUs: 150,
      maxVUs: 300,
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<200'], // 95% of requests under 200ms
    http_req_failed: ['rate<0.01'],   // less than 1% failures
  },
};

export default function () {
  const eventType = EVENT_TYPES[Math.floor(Math.random() * EVENT_TYPES.length)];
  const payload = JSON.stringify({
    event_type: eventType,
    user_id: `user_${Math.floor(Math.random() * 5000)}`,
    source: 'web',
    timestamp: new Date().toISOString(),
    metadata: { order_id: `order_${Math.floor(Math.random() * 100000)}`, amount: Math.floor(Math.random() * 100000) },
  });

  const res = http.post(API_URL, payload, {
    headers: { 'Content-Type': 'application/json' },
  });

  check(res, {
    'status is 202': (r) => r.status === 202,
  });
}
