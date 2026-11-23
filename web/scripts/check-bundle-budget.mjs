// scripts/check-bundle-budget.mjs — the bundle budget from docs/rewrite-plan.md §9 used to be a number written in
// prose and re-measured by hand after every deploy; it is now a real check that fails. Hand-tracking drifted: three
// separate slices ended with "149.5 kB, 220 bytes of headroom" notes and nothing stopped the next one from landing
// over the line without anyone noticing until it was already shipped.
//
// Measures the *gzip* size of the built JS under internal/web/static/next/assets — what actually crosses the wire —
// not the on-disk size, which says nothing useful about load time.
//
// **What is budgeted changed on 23 Nov 2026: the entry graph, not the sum of every chunk.** The gate was written when
// everything was one file, so "total JS" and "what a screen pays" were the same number. Once the player became its own
// lazily-loaded route they stopped being the same question: the sum grows every time a screen is split out, which
// meant the gate would fail *because* the app got faster to open. The budget now applies to the entry chunk — the
// code every screen pays for before the user asks for anything — and each lazily-loaded chunk is listed and checked
// against a looser per-chunk cap so nothing can quietly dump a dependency into a lazy route either.
//
// Budget history: 150 kB (first real app) → 170 kB on 22 Nov 2026 (Kevin: "you can raise it if it's ok to do so"),
// because 150 predated real media browsing, and it is left at 170 on purpose now that the entry is measured: the entry
// is 157.83 kB, so the room exists without moving the number again.
import { readdirSync, readFileSync } from "node:fs";
import { gzipSync } from "node:zlib";
import { join } from "node:path";

const ENTRY_BUDGET_KB = Number(process.env.BUNDLE_BUDGET_KB ?? 170);
/** A single lazy chunk bigger than this is a screen that should itself be split, or a dependency that belongs nowhere. */
const CHUNK_BUDGET_KB = Number(process.env.CHUNK_BUDGET_KB ?? 60);
const ASSETS = join(import.meta.dirname, "..", "..", "internal", "web", "static", "next", "assets");

let files;
try {
  files = readdirSync(ASSETS).filter((name) => name.endsWith(".js"));
} catch {
  console.error(`check-bundle-budget: no build output at ${ASSETS} — run \`npm run build\` first.`);
  process.exit(1);
}

if (files.length === 0) {
  console.error(`check-bundle-budget: ${ASSETS} has no .js chunks — the build produced nothing.`);
  process.exit(1);
}

// zlib's own default gzip level, applied to the emitted bytes: independent of whatever the bundler printed, so the
// check can't drift when Vite's size reporter changes how it rounds or which level it uses.
const measured = files
  .map((name) => ({ name, gz: gzipSync(readFileSync(join(ASSETS, name))).length }))
  .sort((a, b) => b.gz - a.gz);

// Vite names the chunk that index.html loads after the input (`index-<hash>.js`); everything else in assets is either
// a chunk it imports statically, or a lazily-imported route. Assert exactly one entry rather than silently measuring
// nothing if the naming ever changes.
const entryChunks = measured.filter((file) => file.name.startsWith("index-"));
if (entryChunks.length !== 1) {
  console.error(
    `check-bundle-budget: expected exactly one entry chunk named index-*.js, found ${entryChunks.length}. ` +
      `The build's chunk naming changed — fix this check rather than trusting its number.`,
  );
  process.exit(1);
}
const [entry] = entryChunks;
const lazy = measured.filter((file) => file !== entry);

const kb = (bytes) => (bytes / 1024).toFixed(2).padStart(8);
const entryKb = entry.gz / 1024;
const lazyKb = lazy.reduce((sum, file) => sum + file.gz, 0) / 1024;

console.log(`  ${kb(entry.gz)} kB gz   ${entry.name}   (entry — every screen pays this)`);
for (const file of lazy) console.log(`  ${kb(file.gz)} kB gz   ${file.name}   (lazy)`);
console.log(
  `\nentry ${entryKb.toFixed(2)} kB gz of ${ENTRY_BUDGET_KB} kB · ` +
    `lazy ${lazyKb.toFixed(2)} kB in ${lazy.length} chunk${lazy.length === 1 ? "" : "s"} · ` +
    `total ${(entryKb + lazyKb).toFixed(2)} kB`,
);

let failed = false;
if (entryKb > ENTRY_BUDGET_KB) {
  console.error(
    `\nFAIL: the entry chunk is ${entryKb.toFixed(2)} kB gz, ${(entryKb - ENTRY_BUDGET_KB).toFixed(2)} kB over the ` +
      `${ENTRY_BUDGET_KB} kB budget.`,
  );
  console.error("Something every screen pays for has grown: drop the dependency, or move the screen to a lazy route.");
  console.error("Raise the budget in this file AND docs/rewrite-plan.md §9 together — never only one of them.");
  failed = true;
}

for (const file of lazy) {
  const fileKb = file.gz / 1024;
  if (fileKb > CHUNK_BUDGET_KB) {
    console.error(`\nFAIL: lazy chunk ${file.name} is ${fileKb.toFixed(2)} kB gz, over the ${CHUNK_BUDGET_KB} kB per-chunk cap.`);
    failed = true;
  }
}

if (failed) process.exit(1);

console.log(`\nok — ${(ENTRY_BUDGET_KB - entryKb).toFixed(2)} kB of entry headroom left.`);
