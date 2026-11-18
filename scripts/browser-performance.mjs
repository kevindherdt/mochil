import { chromium } from "playwright";

const baseURL = process.env.MOCHIL_BASE_URL || "http://127.0.0.1:8099";
const executablePath = process.env.CHROMIUM_PATH || "/usr/bin/chromium";
const username = process.env.OPDSHELF_USERNAME || "";
const password = process.env.OPDSHELF_PASSWORD || "";
const browser = await chromium.launch({ executablePath, headless: true });
const page = await browser.newPage({ viewport: { width: 1440, height: 900 } });
await page.addInitScript(() => {
  window.__mochilLayoutShifts = [];
  new PerformanceObserver(list => {
    window.__mochilLayoutShifts.push(...list.getEntries().map(entry => ({ value: entry.value, recentInput: entry.hadRecentInput })));
  }).observe({ type: "layout-shift", buffered: true });
});

try {
  await page.goto(baseURL, { waitUntil: "networkidle" });
  if (new URL(page.url()).pathname.startsWith("/login")) {
    if (!username || !password) throw new Error("Set OPDSHELF_USERNAME and OPDSHELF_PASSWORD for an authenticated benchmark.");
    await page.locator('input[name="username"]').fill(username);
    await page.locator('input[name="password"]').fill(password);
    await Promise.all([
      page.waitForURL(url => !url.pathname.startsWith("/login")),
      page.locator('button[type="submit"]').click(),
    ]);
    await page.waitForLoadState("networkidle");
  }

  const initial = await resourceSummary(page);
  const music = page.locator('#filters option').filter({ hasText: /music/i }).first();
  if (await music.count()) {
    const started = performance.now();
    await page.locator("#filters").selectOption(await music.getAttribute("value"));
    await page.locator("#music-destinations:not([hidden])").waitFor();
    await page.waitForFunction(() => !/opening|loading/i.test(document.querySelector("#status")?.textContent || ""));
    await page.waitForLoadState("networkidle");
    initial.musicNavigationMs = Math.round(performance.now() - started);
    initial.afterMusic = await resourceSummary(page);
  }
  console.log(JSON.stringify(initial, null, 2));
  enforceBudgets(initial);
} finally {
  await browser.close();
}

function enforceBudgets(summary) {
  const budgets = {
    homeScriptBytes: budget("MOCHIL_BUDGET_HOME_JS", 200_000),
    homeStyleBytes: budget("MOCHIL_BUDGET_HOME_CSS", 130_000),
    homeDomNodes: budget("MOCHIL_BUDGET_HOME_DOM", 800),
    musicNavigationMs: budget("MOCHIL_BUDGET_MUSIC_MS", 800),
    musicScriptBytes: budget("MOCHIL_BUDGET_MUSIC_JS", 400_000),
    musicStyleBytes: budget("MOCHIL_BUDGET_MUSIC_CSS", 180_000),
    musicDomNodes: budget("MOCHIL_BUDGET_MUSIC_DOM", 1_600),
    cls: budget("MOCHIL_BUDGET_CLS", 0.1),
  };
  const failures = [];
  check(failures, "Home script transfer", summary.scriptTransferBytes, budgets.homeScriptBytes, "bytes");
  check(failures, "Home style transfer", summary.styleTransferBytes, budgets.homeStyleBytes, "bytes");
  check(failures, "Home DOM", summary.domNodes, budgets.homeDomNodes, "nodes");
  check(failures, "Home CLS", summary.cls, budgets.cls, "");
  if (summary.afterMusic) {
    check(failures, "Music navigation", summary.musicNavigationMs, budgets.musicNavigationMs, "ms");
    check(failures, "Music script transfer", summary.afterMusic.scriptTransferBytes, budgets.musicScriptBytes, "bytes");
    check(failures, "Music style transfer", summary.afterMusic.styleTransferBytes, budgets.musicStyleBytes, "bytes");
    check(failures, "Music DOM", summary.afterMusic.domNodes, budgets.musicDomNodes, "nodes");
    check(failures, "Music CLS", summary.afterMusic.cls, budgets.cls, "");
  }
  if (failures.length) throw new Error(`Browser performance budget exceeded:\n${failures.join("\n")}`);
}

function budget(name, fallback) {
  const value = Number(process.env[name]);
  return Number.isFinite(value) && value >= 0 ? value : fallback;
}

function check(failures, label, actual, maximum, unit) {
  if (actual > maximum) failures.push(`- ${label}: ${actual}${unit} > ${maximum}${unit}`);
}

async function resourceSummary(currentPage) {
  return currentPage.evaluate(() => {
    const resources = performance.getEntriesByType("resource");
    const scripts = resources.filter(entry => entry.initiatorType === "script");
    const styles = resources.filter(entry => entry.initiatorType === "link" && /\.css(?:\?|$)/.test(entry.name));
    const bytes = entries => entries.reduce((total, entry) => total + (entry.transferSize || 0), 0);
    return {
      resources: resources.length,
      scripts: scripts.length,
      scriptTransferBytes: bytes(scripts),
      styles: styles.length,
      styleTransferBytes: bytes(styles),
      domNodes: document.getElementsByTagName("*").length,
      cls: Number((window.__mochilLayoutShifts || []).filter(entry => !entry.recentInput).reduce((total, entry) => total + entry.value, 0).toFixed(4)),
    };
  });
}
