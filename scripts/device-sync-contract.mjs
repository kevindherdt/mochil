// scripts/device-sync-contract.mjs — the other half of the reader sync test.
//
// reader-sync-contract.mjs drives the web reader through a browser. This one speaks as a *device*: a paired token, the
// /api/android/v1/reading path the phone and the plugin use, and the compare-and-set that decides which position wins. Between
// them, both ends of the ledger are covered without a phone in someone's hand.
//
// The four assertions:
//   1. a device token can read the position for a book;
//   2. a push is accepted and moves the record (fraction and revision);
//   3. a push built on a stale revision is refused with 409, so two devices cannot silently overwrite each other;
//   4. the restored position is what the server answers with afterwards.
//
// It restores what it found, so it is safe against a real library. Token: MOCHIL_DEVICE_TOKEN or MOCHIL_DEVICE_TOKEN_FILE;
// with neither it says so and exits 0, because a missing credential is not a passing test.
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";

const baseURL = process.env.MOCHIL_BASE_URL || "http://127.0.0.1:8099";
const tokenFile = process.env.MOCHIL_DEVICE_TOKEN_FILE || "";
const token = process.env.MOCHIL_DEVICE_TOKEN || (tokenFile ? readFileSync(tokenFile, "utf8").trim() : "");

if (!token) {
  console.log("device-sync-contract: no MOCHIL_DEVICE_TOKEN or MOCHIL_DEVICE_TOKEN_FILE, so nothing was proven.");
  process.exit(0);
}

const headers = { Authorization: `Bearer ${token}` };
const call = (path, init = {}) => fetch(`${baseURL}${path}`, { ...init, headers: { ...headers, ...(init.headers || {}) } });

const library = process.env.MOCHIL_READER_LIBRARY || "books";

async function pickBook() {
  if (process.env.MOCHIL_READER_BOOK) return process.env.MOCHIL_READER_BOOK;
  const response = await call(`/api/android/v1/books?library=${encodeURIComponent(library)}&limit=100`);
  assert.equal(response.status, 200, `the device book list must load (got ${response.status})`);
  const books = await response.json();
  const book = books.find((item) => item.format === "epub" && Number(item.size) > 300000 && Number(item.size) < 8000000);
  assert.ok(book, "no mid-sized EPUB found; set MOCHIL_READER_BOOK=<library>/<id>");
  return `${library}/${book.id}`;
}

const readPosition = async (book) => {
  const response = await call(`/api/android/v1/reading/${book}`);
  if (response.status === 404) return null;
  assert.equal(response.status, 200, `a device must be able to read the position (got ${response.status})`);
  return response.json();
};

const push = (book, body) =>
  call(`/api/android/v1/reading/${book}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });

const book = await pickBook();
const before = await readPosition(book);
console.log(
  `device-sync-contract: ${book}, starts at ${before ? `fraction ${Number(before.fraction).toFixed(4)} (revision ${before.revision})` : "no saved position"}`,
);

const startFraction = before ? Number(before.fraction) : 0;
const target = Math.min(0.9, startFraction + 0.05);

// (1) + (2) A push moves the record.
const written = await push(book, {
  fraction: target,
  device_id: "device-sync-contract",
  device_name: "contract test",
  base_revision: before?.revision ?? 0,
});
assert.equal(written.status, 200, `the push must be accepted (got ${written.status}: ${await written.text()})`);
const afterPush = await readPosition(book);
assert.ok(afterPush, "the record must exist after a push");
assert.ok(Math.abs(Number(afterPush.fraction) - target) < 1e-6, `the stored fraction must be the pushed one (${afterPush.fraction} vs ${target})`);
assert.ok(!before || afterPush.revision > before.revision, "a write must move the revision forward");
console.log(`device-sync-contract: push accepted, fraction ${Number(afterPush.fraction).toFixed(4)} (revision ${afterPush.revision})`);

// (3) A push built on the revision it saw before the write is refused, which is what keeps two devices from overwriting
// each other silently.
const stale = await push(book, {
  fraction: Math.min(0.9, target + 0.01),
  device_id: "device-sync-contract",
  device_name: "contract test",
  base_revision: before?.revision ?? 0,
});
assert.equal(stale.status, 409, `a stale push must be refused with 409, got ${stale.status}`);
console.log("device-sync-contract: a stale push is refused (409)");

// (4) Put back what was there, on top of the revision the push produced.
const restore = await push(book, {
  fraction: startFraction,
  device_id: before?.device_id || "device-sync-contract",
  device_name: before?.device_name || "contract test",
  base_revision: afterPush.revision,
  ...(before?.web_anchor ? { web_anchor: before.web_anchor } : {}),
  ...(before?.locators ? { locators: before.locators } : {}),
});
assert.equal(restore.status, 200, `restoring the previous position failed (${restore.status})`);
const restored = await readPosition(book);
assert.ok(Math.abs(Number(restored.fraction) - startFraction) < 1e-6, "the restored fraction must be the one that was there before");
console.log("device-sync-contract: previous position restored");
console.log("device-sync-contract: passed (read, push, stale-refused, restore)");
