import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { readFileSync, readdirSync, writeFileSync } from "node:fs";
import test from "node:test";

// A stylesheet referenced as `foo.css?v=3` is served `immutable` for a year, so the URL — not the
// bytes — is what the browser keeps. Editing foo.css without changing that number leaves every
// returning visitor on the old sheet, silently, until they clear their cache: the settings page
// looked two columns wide for weeks after it became one. This records what each version number was
// minted from, so a content change without a bump fails here instead of in someone's browser.
const STATIC = new URL("../static/", import.meta.url);
const LOCK = new URL("./asset-versions.json", import.meta.url);

const sha = (file) => createHash("sha256").update(readFileSync(new URL(file, STATIC))).digest("hex").slice(0, 16);

function references() {
  const found = new Map(); // "file.css?v=N" -> Set(pages that ask for it)
  for (const page of readdirSync(STATIC).filter((name) => name.endsWith(".html") || name.endsWith(".js"))) {
    const text = readFileSync(new URL(page, STATIC), "utf8");
    for (const match of text.matchAll(/([A-Za-z0-9._-]+\.css)\?v=(\d+)/g)) {
      const key = `${match[1]}?v=${match[2]}`;
      if (!found.has(key)) found.set(key, new Set());
      found.get(key).add(page);
    }
  }
  return found;
}

test("every versioned stylesheet is recorded, and recorded at its current contents", () => {
  const found = references();
  assert.ok(found.size > 20, "the scan found almost no stylesheets; it is not looking where it thinks");

  if (process.env.RECORD_ASSET_VERSIONS === "1") {
    const lock = {};
    for (const key of [...found.keys()].sort()) lock[key] = sha(key.split("?v=")[0]);
    writeFileSync(LOCK, JSON.stringify(lock, null, 2) + "\n");
    return;
  }

  const lock = JSON.parse(readFileSync(LOCK, "utf8"));
  const problems = [];
  for (const [key, pages] of found) {
    const file = key.split("?v=")[0];
    const recorded = lock[key];
    if (!recorded) {
      problems.push(`${key} (${[...pages].join(", ")}) has no recorded contents: bump the version and run RECORD_ASSET_VERSIONS=1`);
      continue;
    }
    const current = sha(file);
    if (recorded !== current) {
      problems.push(`${key} changed since that version was minted (${recorded} -> ${current}): bump ?v= and re-record, or returning visitors keep the old sheet`);
    }
  }
  assert.deepEqual(problems, []);
});

test("one stylesheet is not served under two version numbers", () => {
  // Two numbers for the same file means one page can be a deploy behind the other, and a bump on one
  // page silently skips the other.
  const byFile = new Map();
  for (const key of references().keys()) {
    const [file, version] = key.split("?v=");
    if (!byFile.has(file)) byFile.set(file, new Set());
    byFile.get(file).add(version);
  }
  const split = [...byFile].filter(([, versions]) => versions.size > 1);
  assert.deepEqual(split, [], "these stylesheets carry more than one ?v= across pages");
});
