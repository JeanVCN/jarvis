import assert from "node:assert/strict";
import test from "node:test";
import { sameOriginRequest } from "./request-origin.ts";

test("accepts same-origin browser mutations on localhost", () => {
  const request = new Request("http://127.0.0.1:13000/api/opencode/sessions", {
    method: "POST",
    headers: { host: "127.0.0.1:13000", origin: "http://127.0.0.1:13000" },
  });
  assert.equal(sameOriginRequest(request), true);
});

test("accepts HTTPS tailnet origin when reverse proxy marks the scheme", () => {
  const request = new Request("http://jarvis.internal/api/opencode/sessions", {
    method: "POST",
    headers: { host: "jarvis.example.ts.net", origin: "https://jarvis.example.ts.net", "x-forwarded-proto": "https" },
  });
  assert.equal(sameOriginRequest(request), true);
});

test("rejects cross-origin, missing-origin, and forwarded-host spoof attempts", () => {
  const crossOrigin = new Request("http://127.0.0.1:13000/api/opencode/sessions", {
    method: "POST",
    headers: { host: "127.0.0.1:13000", origin: "https://attacker.example", "x-forwarded-host": "attacker.example", "x-forwarded-proto": "https" },
  });
  const missing = new Request("http://127.0.0.1:13000/api/opencode/sessions", {
    method: "POST",
    headers: { host: "127.0.0.1:13000" },
  });
  assert.equal(sameOriginRequest(crossOrigin), false);
  assert.equal(sameOriginRequest(missing), false);
});
