import assert from "node:assert/strict";
import { readFileSync, readdirSync } from "node:fs";
import test from "node:test";

// A helper a module calls but never binds throws only when that exact line runs, which is how
// "Could not save: withdrawRestart is not defined" reached a person: every non-module save failed and the
// page looked fine until Save was pressed. So every sibling helper a file calls must be bound in it —
// imported, declared, or taken from a parameter or an object it destructures.
const dir = new URL("../static/", import.meta.url);
const files = readdirSync(dir).filter((name) => name.endsWith(".js"));

const exported = new Map(); // helper name -> the file that exports it
for (const name of files) {
  const source = readFileSync(new URL(name, dir), "utf8");
  for (const match of source.matchAll(/export\s+(?:async\s+)?function\s+([A-Za-z_$][\w$]*)/g)) exported.set(match[1], name);
  for (const match of source.matchAll(/export\s+(?:const|let|var)\s+([A-Za-z_$][\w$]*)/g)) exported.set(match[1], name);
}

// Every identifier that appears inside a {...} block counts as bound. Object literals and destructuring
// patterns are not worth telling apart for this question: being generous here can only miss a bug, never
// invent one.
function boundNames(source) {
  const names = new Set();
  for (const match of source.matchAll(/import\s*\{([^}]*)\}/g)) {
    for (const part of match[1].split(",")) {
      const name = part.trim().split(/\s+as\s+/).pop().trim();
      if (name) names.add(name);
    }
  }
  for (const match of source.matchAll(/import\s+([A-Za-z_$][\w$]*)\s*(?:,|from)/g)) names.add(match[1]);
  const patterns = [
    /(?:^|\n)\s*(?:export\s+)?(?:async\s+)?function\s+([A-Za-z_$][\w$]*)/g,
    /(?:^|\n)\s*(?:export\s+)?(?:const|let|var)\s+([A-Za-z_$][\w$]*)/g,
    /(?:^|\n)\s*(?:export\s+)?class\s+([A-Za-z_$][\w$]*)/g,
  ];
  for (const pattern of patterns) {
    for (const match of source.matchAll(pattern)) names.add(match[1]);
  }
  // any identifier sitting in a parameter position anywhere: function (a, b, requestJSON)
  for (const match of source.matchAll(/\(([^()]{0,400})\)/g)) {
    for (const part of match[1].split(",")) {
      const name = part.trim().split("=")[0].trim();
      if (/^[A-Za-z_$][\w$]*$/.test(name)) names.add(name);
    }
  }
  // depth-aware walk of every brace block
  for (let i = 0; i < source.length; i++) {
    if (source[i] !== "{") continue;
    let depth = 0;
    let end = i;
    for (; end < source.length; end++) {
      if (source[end] === "{") depth++;
      else if (source[end] === "}" && --depth === 0) break;
    }
    for (const match of source.slice(i, end + 1).matchAll(/([A-Za-z_$][\w$]*)\s*(?::|,|=|\})/g)) names.add(match[1]);
  }
  return names;
}

// Prose in a comment or a string is not a call: "the request (HTTP 500)" once flagged request().
function stripNoise(source) {
  let out = "";
  let quote = null;
  let line = false;
  let block = false;
  for (let i = 0; i < source.length; i++) {
    const c = source[i];
    const next = source[i + 1];
    if (line) {
      if (c === "\n") { line = false; out += c; } else out += " ";
      continue;
    }
    if (block) {
      if (c === "*" && next === "/") { block = false; out += "  "; i++; } else out += c === "\n" ? c : " ";
      continue;
    }
    if (quote) {
      if (c === "\\") { out += "  "; i++; continue; }
      if (c === quote) { quote = null; out += c; } else out += c === "\n" ? c : " ";
      continue;
    }
    if (c === "/" && next === "/") { line = true; out += "  "; i++; continue; }
    if (c === "/" && next === "*") { block = true; out += "  "; i++; continue; }
    if (c === '"' || c === "'" || c === "`") { quote = c; out += c; continue; }
    out += c;
  }
  return out;
}

test("no module calls a sibling helper it never bound", () => {
  const problems = [];
  for (const name of files) {
    const source = readFileSync(new URL(name, dir), "utf8");
    const available = boundNames(source);
    for (const [helper, owner] of exported) {
      if (owner === name || available.has(helper)) continue;
      // called as a bare function, not as a property of something
      if (new RegExp(`(^|[^.\\w$])${helper}\\s*\\(`, "m").test(stripNoise(source))) {
        problems.push(`${name} calls ${helper}() from ${owner} without importing it`);
      }
    }
  }
  assert.deepEqual(problems, []);
});
