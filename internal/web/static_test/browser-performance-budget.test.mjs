import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

test("authenticated browser benchmark enforces Home and Music performance budgets", async () => {
  const source = await readFile(new URL("../../../scripts/browser-performance.mjs", import.meta.url), "utf8");
  assert.match(source, /enforceBudgets\(initial\)/);
  for (const budget of [
    "MOCHIL_BUDGET_HOME_JS",
    "MOCHIL_BUDGET_HOME_CSS",
    "MOCHIL_BUDGET_HOME_DOM",
    "MOCHIL_BUDGET_MUSIC_MS",
    "MOCHIL_BUDGET_MUSIC_JS",
    "MOCHIL_BUDGET_MUSIC_CSS",
    "MOCHIL_BUDGET_MUSIC_DOM",
    "MOCHIL_BUDGET_CLS",
  ]) assert.match(source, new RegExp(budget));
  assert.match(source, /Browser performance budget exceeded/);
});
