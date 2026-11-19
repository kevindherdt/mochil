import assert from "node:assert/strict";
import test from "node:test";
import { createRemoteZipFile } from "../static/remote-zip-file.js";

function server(bytes) {
  const calls = [];
  const fetchImpl = async (_url, { headers }) => {
    const [, start, end] = /bytes=(\d+)-(\d+)/.exec(headers.Range);
    calls.push([Number(start), Number(end)]);
    const body = bytes.slice(Number(start), Number(end) + 1);
    return { status: 206, arrayBuffer: async () => body.buffer.slice(body.byteOffset, body.byteOffset + body.byteLength) };
  };
  return { calls, fetchImpl };
}

const data = Uint8Array.from({ length: 5000 }, (_, i) => i % 251);

test("range file serves slices, spans blocks, and caches repeated reads", async () => {
  const { calls, fetchImpl } = server(data);
  const file = createRemoteZipFile({ url: "/x", name: "x.epub", size: data.length, fetchImpl, blockSize: 1000 });
  const head = new Uint8Array(await file.slice(0, 4).arrayBuffer());
  assert.deepEqual([...head], [...data.slice(0, 4)]);
  const spanning = await file.rangeReader.readUint8Array(900, 300);
  assert.deepEqual([...spanning], [...data.slice(900, 1200)]);
  const before = calls.length;
  await file.rangeReader.readUint8Array(950, 20);
  assert.equal(calls.length, before, "cached block is reused");
});

test("large reads bypass the cache and the last block is clamped to the file size", async () => {
  const { calls, fetchImpl } = server(data);
  const file = createRemoteZipFile({ url: "/x", name: "x.epub", size: data.length, fetchImpl, blockSize: 1000 });
  const big = await file.rangeReader.readUint8Array(100, 4000);
  assert.equal(big.length, 4000);
  assert.deepEqual(calls.at(-1), [100, 4099]);
  const tail = await file.rangeReader.readUint8Array(4900, 500);
  assert.equal(tail.length, 100);
});

test("a failed block is not cached", async () => {
  let fail = true;
  const fetchImpl = async () => (fail ? { status: 500 } : { status: 206, arrayBuffer: async () => new Uint8Array(10).buffer });
  const file = createRemoteZipFile({ url: "/x", name: "x", size: 10, fetchImpl, blockSize: 10 });
  await assert.rejects(() => file.rangeReader.readUint8Array(0, 5));
  fail = false;
  assert.equal((await file.rangeReader.readUint8Array(0, 5)).length, 5);
});
