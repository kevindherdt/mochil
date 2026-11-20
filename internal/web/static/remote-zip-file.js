// A File-shaped handle for a large publication that is read with HTTP range
// requests, so a 500 MB read-aloud EPUB never has to be downloaded whole.
// The reader library asks for `rangeReader` (zip.js's reader contract) and uses
// `name`, `type`, `size` and `slice().arrayBuffer()` for type sniffing.
export function createRemoteZipFile({
  url,
  name,
  type = "application/epub+zip",
  size,
  lastModified = Date.now(),
  fetchImpl = (...args) => fetch(...args),
  blockSize = 256 * 1024,
  maxBlocks = 48,
}) {
  const blocks = new Map();

  async function fetchRange(start, end) {
    const response = await fetchImpl(url, {
      headers: { Range: `bytes=${start}-${end - 1}` },
    });
    if (response.status !== 206 && !(response.status === 200 && start === 0))
      throw new Error(`Range request returned HTTP ${response.status}`);
    const bytes = new Uint8Array(await response.arrayBuffer());
    return response.status === 200 ? bytes.subarray(start, end) : bytes;
  }

  function block(index) {
    let entry = blocks.get(index);
    if (!entry) {
      const start = index * blockSize;
      entry = fetchRange(start, Math.min(size, start + blockSize));
      entry.catch(() => blocks.delete(index));
      blocks.set(index, entry);
      while (blocks.size > maxBlocks) blocks.delete(blocks.keys().next().value);
    } else {
      blocks.delete(index);
      blocks.set(index, entry);
    }
    return entry;
  }

  async function read(start, end) {
    end = Math.min(end, size);
    if (end <= start) return new Uint8Array(0);
    // Large reads (media, big images) bypass the block cache.
    if (end - start > blockSize * 2) return fetchRange(start, end);
    const first = Math.floor(start / blockSize);
    const last = Math.floor((end - 1) / blockSize);
    const parts = await Promise.all(
      Array.from({ length: last - first + 1 }, (_, i) => block(first + i)),
    );
    const out = new Uint8Array(end - start);
    let offset = 0;
    parts.forEach((part, i) => {
      const partStart = (first + i) * blockSize;
      const from = Math.max(start, partStart) - partStart;
      const to = Math.min(end, partStart + part.length) - partStart;
      out.set(part.subarray(from, to), offset);
      offset += to - from;
    });
    return out;
  }

  return {
    name,
    type,
    size,
    lastModified,
    // Present so the reader library recognises this as a file, not a parsed book.
    arrayBuffer: () => read(0, size).then((bytes) => bytes.buffer),
    slice: (start = 0, end = size) => ({
      arrayBuffer: async () => {
        const bytes = await read(start, end);
        return bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength);
      },
    }),
    rangeReader: {
      size,
      initialized: true,
      async init() {},
      readUint8Array: (index, length) => read(index, index + length),
    },
  };
}
