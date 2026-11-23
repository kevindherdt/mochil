#!/usr/bin/env node
// Proves the eslint and stylelint rules in eslint.config.js / .stylelintrc.json (laws L1/L2/L5/L7, plan
// docs/rewrite-plan.md §4.2) actually fire on a real violation, not just on paper. Each fixture under
// src/features/__lint_fixtures__ is globally ignored by the main configs (so it never fails `npm run lint`);
// this script lints each one individually with that one ignore overridden, and fails loudly if a fixture does
// NOT produce the rule id it's supposed to.
import { ESLint } from "eslint";
import stylelint from "stylelint";

let allOk = true;

// --- eslint: L1, L2, L5 -----------------------------------------------------------------------------------
const eslintChecks = [
  { file: "src/features/__lint_fixtures__/L1Violation.tsx", rule: "no-restricted-syntax", law: "L1" },
  { file: "src/features/__lint_fixtures__/screens/L2Violation.tsx", rule: "no-restricted-imports", law: "L2" },
  { file: "src/features/__lint_fixtures__/screens/L5Violation.tsx", rule: "no-restricted-syntax", law: "L5" },
];

const eslint = new ESLint({ ignore: false });
for (const check of eslintChecks) {
  const results = await eslint.lintFiles([check.file]);
  const messages = results.flatMap((r) => r.messages);
  const matched = messages.filter((m) => m.ruleId === check.rule);
  if (matched.length === 0) {
    allOk = false;
    console.error(`FAIL  ${check.law}: ${check.file} did not trigger ${check.rule} — the rule does not work.`);
    for (const m of messages) console.error(`        (got instead: ${m.ruleId}: ${m.message})`);
  } else {
    console.log(`ok    ${check.law}: ${check.file} correctly failed (${check.rule})`);
  }
}

// --- stylelint: L1, L3, L7 ----------------------------------------------------------------------------------
const styleChecks = [
  { file: "src/features/__lint_fixtures__/StyleViolation.module.css", rule: "declaration-property-value-disallowed-list", law: "L1" },
  { file: "src/features/__lint_fixtures__/StyleViolation.module.css", rule: "at-rule-disallowed-list", law: "L7" },
  { file: "src/features/__lint_fixtures__/L3Violation.module.css", rule: "scale-unlimited/declaration-strict-value", law: "L3" },
];

// stylelint's config `ignoreFiles` entry for this directory is what keeps the fixtures out of the project's own
// `npm run lint:css` — so it must be stripped here, on an explicitly-resolved copy of the real config, to prove
// the *rules themselves* (not just the fact that the files are skipped) actually fire.
const styleFiles = [...new Set(styleChecks.map((c) => c.file))];
const resolvedConfig = await stylelint.resolveConfig(styleFiles[0]);
delete resolvedConfig.ignoreFiles;
const styleResult = await stylelint.lint({ files: styleFiles, config: resolvedConfig });
const warningsByFile = new Map(styleResult.results.map((r) => [r.source, r.warnings]));
for (const check of styleChecks) {
  const warnings = [...warningsByFile.entries()].find(([source]) => source.endsWith(check.file))?.[1] ?? [];
  const matched = warnings.filter((w) => w.rule === check.rule);
  if (matched.length === 0) {
    allOk = false;
    console.error(`FAIL  ${check.law}: ${check.file} did not trigger ${check.rule} — the rule does not work.`);
  } else {
    console.log(`ok    ${check.law}: ${check.file} correctly failed (${check.rule})`);
  }
}

if (!allOk) {
  console.error("\nOne or more layout-law lint rules failed to catch their own deliberate violation.");
  process.exit(1);
}
console.log("\nAll layout-law lint rules proven to fire on a real violation.");
