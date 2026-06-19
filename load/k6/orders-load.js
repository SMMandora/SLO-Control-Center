import http from "k6/http";
import { check, sleep } from "k6";

// Deterministic synthetic load: fixed arrival rate + VUs, item chosen from the
// iteration counter (no Math.random) so runs are reproducible.
export const options = {
  scenarios: {
    steady: {
      executor: "constant-arrival-rate",
      rate: 50,
      timeUnit: "1s",
      duration: "12h",
      preAllocatedVUs: 20,
      maxVUs: 50,
    },
  },
};

const BASE = __ENV.ORDERS_URL || "http://orders-api:8080";
const items = ["widget", "gadget", "gizmo", "doohickey"];

export default function () {
  const i = __ITER % items.length;
  const res = http.post(
    `${BASE}/orders`,
    JSON.stringify({ item: items[i], qty: (i % 3) + 1 }),
    { headers: { "Content-Type": "application/json" } },
  );
  // 500s are injected chaos and are an expected outcome, not a load-test failure.
  check(res, { "created or chaos": (r) => r.status === 201 || r.status === 500 });
  sleep(0.01);
}
