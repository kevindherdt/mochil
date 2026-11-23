#!/usr/bin/env node
// Checks every theme in design/tokens.json against WCAG 2.1 AA contrast minimums (plan docs/rewrite-plan.md §4.1,
// law L6: "themes are tokens only; contrast is checked"). A theme is free to pick any colours it likes, but it must
// clear these ratios, so a new or retuned theme can't quietly ship text nobody can read.
//
// Thresholds used (WCAG 2.1 §1.4.3 / §1.4.11):
//   4.5:1  — normal text pairs: text/bg, text/surface, mutedText/bg, mutedText/surface, accentInk/accent (a button's
//            label), error/bg (error text is still text, read at body size).
//   3.0:1  — non-text UI components and graphical objects: accent/bg (borders, icons, focus rings, the accent used
//            as a stroke rather than as a text colour).
//
// No dependencies beyond Node's stdlib, matching scripts/gen-tokens.mjs.

import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import path from "node:path";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const tokens = JSON.parse(readFileSync(path.join(root, "design/tokens.json"), "utf8"));

function relativeLuminance(hex) {
  const value = hex.replace("#", "");
  const channel = (hexPair) => {
    const c = parseInt(hexPair, 16) / 255;
    return c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4;
  };
  const [r, g, b] = [value.slice(0, 2), value.slice(2, 4), value.slice(4, 6)].map(channel);
  return 0.2126 * r + 0.7152 * g + 0.0722 * b;
}

function contrastRatio(hexA, hexB) {
  const [lighter, darker] = [relativeLuminance(hexA), relativeLuminance(hexB)].sort((a, b) => b - a);
  return (lighter + 0.05) / (darker + 0.05);
}

// Each check: a human label, which two roles to compare, and the WCAG minimum that pair must clear.
const CHECKS = [
  { label: "text on bg (body text)", a: "text", b: "bg", min: 4.5 },
  { label: "text on surface (body text)", a: "text", b: "surface", min: 4.5 },
  { label: "muted on bg (secondary text)", a: "muted", b: "bg", min: 4.5 },
  { label: "muted on surface (secondary text)", a: "muted", b: "surface", min: 4.5 },
  { label: "accentInk on accent (button label)", a: "accentInk", b: "accent", min: 4.5 },
  { label: "error on bg (error text)", a: "error", b: "bg", min: 4.5 },
  { label: "success on bg (success icon / badge fill)", a: "success", b: "bg", min: 3.0 },
  { label: "accent on bg (focus ring / icon stroke)", a: "accent", b: "bg", min: 3.0 },
];

let failures = [];
let checked = 0;

for (const themeId of tokens.canonicalThemes) {
  const theme = tokens.themes[themeId];
  for (const mode of ["dark", "light"]) {
    const colors = theme[mode];
    for (const check of CHECKS) {
      checked += 1;
      const ratio = contrastRatio(colors[check.a], colors[check.b]);
      if (ratio < check.min) {
        failures.push(
          `${theme.label} (${themeId}) ${mode}: ${check.label} is ${ratio.toFixed(2)}:1, needs ${check.min}:1 ` +
          `(${check.a}=${colors[check.a]} vs ${check.b}=${colors[check.b]})`
        );
      }
    }
  }
}

console.log(`Checked ${checked} contrast pairs across ${tokens.canonicalThemes.length} themes × 2 modes.`);
if (failures.length > 0) {
  console.error(`\n${failures.length} contrast failure(s):\n`);
  failures.forEach((f) => console.error(`  FAIL ${f}`));
  process.exit(1);
}
console.log("All pairs clear WCAG 2.1 AA.");
