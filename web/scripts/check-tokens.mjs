#!/usr/bin/env node
// check-tokens.mjs — fail on a `var(--something)` that no CSS file defines.
//
// A misspelled custom property does not throw: the declaration is simply invalid, so the browser drops it. The
// editor's column gap was `var(--space-6)`, and the spacing scale is 4/8/12/16/24/32/48 — so the gap was zero, and
// "#" and "Year" rendered flush against each other for as long as the code existed, reading as one number
// ("12004"). Nothing else catches this: TypeScript never sees CSS, stylelint checks the *values* of tokens it can
// resolve but not whether they exist, and a browser shows it as a layout that is merely slightly wrong.
//
// Runtime tokens are allowed through `ALLOWED` below: they are published by JavaScript at mount (`--vvh`,
// `--dock-bottom`, `--vv-offset-top` by lib/useVisibleViewport) and by container queries.
import { readdirSync, readFileSync, statSync } from "node:fs";
import { join, relative } from "node:path";

const ROOT = new URL("../src", import.meta.url).pathname;
const ALLOWED = new Set(["--vvh", "--dock-bottom", "--vv-offset-top", "--sheet-scrim"]);

function walk(dir) {
  return readdirSync(dir).flatMap((entry) => {
    const path = join(dir, entry);
    return statSync(path).isDirectory() ? walk(path) : [path];
  });
}

const files = walk(ROOT).filter((path) => /\.(css|tsx?|html)$/.test(path));
const defined = new Set();
for (const path of files) {
  const text = readFileSync(path, "utf8");
  for (const match of text.matchAll(/(--[a-z0-9-]+)\s*:/g)) defined.add(match[1]);
}

const problems = [];
for (const path of files) {
  const text = readFileSync(path, "utf8");
  for (const match of text.matchAll(/var\((--[a-z0-9-]+)/g)) {
    const name = match[1];
    // `var(--space-${n})` in a template literal: the name is built at runtime, so the literal prefix is not a token.
    if (text.slice(match.index + match[0].length, match.index + match[0].length + 2) === "${") continue;
    if (defined.has(name) || ALLOWED.has(name)) continue;
    const line = text.slice(0, match.index).split("\n").length;
    problems.push(`${relative(ROOT, path)}:${line}  ${name} is not defined`);
  }
}

if (problems.length > 0) {
  console.error(`✖ ${problems.length} use(s) of a custom property nothing defines — the declaration is silently dropped:\n`);
  for (const problem of [...new Set(problems)]) console.error(`  ${problem}`);
  process.exit(1);
}
console.log(`ok — every var() refers to a token that exists (${defined.size} defined).`);
