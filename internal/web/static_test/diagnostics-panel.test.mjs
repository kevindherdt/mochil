import assert from "node:assert/strict";
import test from "node:test";
import { ago, diagnosticsHTML, formatUptime, mountDiagnostics } from "../static/diagnostics-panel.js";

test("uptime reads as the largest useful units", () => {
  assert.equal(formatUptime(30), "0m");
  assert.equal(formatUptime(3 * 3600 + 5 * 60), "3h 5m");
  assert.equal(formatUptime(2 * 86400 + 4 * 3600), "2d 4h");
});

test("the summary escapes request paths and explains empty lists", () => {
  const html = diagnosticsHTML({ uptime_seconds: 90, heap_mb: 33.4, requests: 12, failures: 0, libraries: 2, goroutines: 18,
    slow: [{ method: "GET", path: "/<b>x</b>", status: 200, duration_ms: 900, at: 0 }], errors: [] });
  assert.match(html, /&lt;b&gt;x&lt;\/b&gt;/);
  assert.doesNotMatch(html, /<b>x/);
  assert.match(html, /No server errors since the server started\./);
  assert.match(html, /33 MB/);
});

test("the panel loads only when opened and survives a failed request", async () => {
  let toggle, body = { textContent: "", innerHTML: "" };
  const details = { open: false, addEventListener: (type, fn) => (toggle = fn) };
  const panel = {
    querySelector: (selector) => (selector === "#diagnostics" ? (panel.mounted ? details : null) : selector === "#diagnostics-body" ? body : null),
    insertAdjacentHTML() { panel.mounted = true; },
  };
  let calls = 0;
  mountDiagnostics({ querySelector: () => panel }, async () => { calls++; throw new Error("503"); });
  assert.equal(calls, 0);
  details.open = true;
  await toggle();
  assert.equal(calls, 1);
  assert.match(body.textContent, /unavailable/);
});

test("background failures say what stopped, when and how often, and are escaped", () => {
  const html = diagnosticsHTML({ uptime_seconds: 1, heap_mb: 1, requests: 0, failures: 0, libraries: 1, goroutines: 1, slow: [], errors: [],
    background: [{ what: "narration <warm>", at: Date.now() - 10 * 60000, message: "nil map", count: 3, restarts: 2 }] });
  assert.match(html, /narration &lt;warm&gt;/);
  assert.match(html, /10 min ago · 3 times · restarted · nil map/);
  assert.doesNotMatch(html, /<warm>/);
});

test("with nothing failed the section says so", () => {
  const html = diagnosticsHTML({ uptime_seconds: 1, heap_mb: 1, requests: 0, failures: 0, libraries: 1, goroutines: 1, slow: [], errors: [] });
  assert.match(html, /Nothing in the background has failed/);
});

test("ago reads in minutes, hours and days", () => {
  const now = 1_000_000_000_000;
  assert.equal(ago(now - 20_000, now), "just now");
  assert.equal(ago(now - 5 * 60000, now), "5 min ago");
  assert.equal(ago(now - 3 * 3600_000, now), "3 h ago");
  assert.equal(ago(now - 72 * 3600_000, now), "3 d ago");
});
