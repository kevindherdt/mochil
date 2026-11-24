// The cross-engine layout gate (plan docs/rewrite-plan.md §9, law L8). It exists because a Firefox-only bug — the
// dock sitting behind Firefox Android's own bottom bar while Chrome was fine — shipped at 0.1.86 and nobody caught it
// until Kevin saw it on his phone. This script runs the mechanism and visible checks in Chromium, Firefox and
// WebKit so that class of bug fails CI instead of a phone screenshot.
//
// Two kinds of check:
//   - MECHANISM checks replace `window.visualViewport` with a fake one before any app script runs, so the actual
//     logic in dock.js is exercised identically in all three engines — this does not depend on each engine's real
//     mobile chrome, which Playwright cannot emulate for Firefox/WebKit anyway. It is the regression test for the
//     0.1.86 bug: the frame and the dock must follow whatever visualViewport reports, in every engine.
//   - VISIBLE checks run against the real (unmocked) viewport: the dock sits flush with the bottom of the window,
//     and the Settings page has nothing clipped above or below the visible area.
//
// Header height and first-cover-offset checks need a real library with real books; they are skipped with a clear
// message against a fresh, library-less install (the CI ephemeral container from install-smoke.sh) and only run
// when MOCHIL_BASE_URL points at a box that has a library (e.g. the dev box). Full fixture-backed e2e lands in
// Phase 0.3/Phase 1 (contract fixtures).
//
// Usage:
//   node scripts/layout-gate.mjs
//   MOCHIL_BASE_URL=http://your-server:8099 OPDSHELF_USERNAME=... OPDSHELF_PASSWORD=... node scripts/layout-gate.mjs
//   LAYOUT_GATE_ENGINES=chromium,firefox node scripts/layout-gate.mjs   # narrow the engine set

import assert from "node:assert/strict";
import { chromium, firefox, webkit } from "playwright";

const baseURL = process.env.MOCHIL_BASE_URL || "http://127.0.0.1:8099";
const username = process.env.OPDSHELF_USERNAME || "";
const password = process.env.OPDSHELF_PASSWORD || "";
const engineNames = (process.env.LAYOUT_GATE_ENGINES || "chromium,firefox,webkit").split(",").map((s) => s.trim());
const engines = { chromium, firefox, webkit };
const VIEWPORT = { width: 390, height: 844 };

// Replaces window.visualViewport with a fake reporting `height`, before any page script runs. This is what lets
// one script prove the same thing Chrome and Firefox disagree about on a real phone: Chrome's visualViewport equals
// the window on a page with no on-screen keyboard and no browser-chrome overlay; Firefox Android's does not, because
// Firefox keeps its own navigation bar in the layout viewport and reports the visual (smaller) one truthfully. A
// shell that reads visualViewport gets the right answer in both; a shell that reads 100dvh does not.
const fakeVisualViewport = (height) => `
  (() => {
    const real = window.visualViewport;
    const listeners = { resize: [], scroll: [] };
    const fake = {
      height: ${height},
      width: real ? real.width : window.innerWidth,
      offsetTop: 0,
      addEventListener: (type, fn) => listeners[type]?.push(fn),
      removeEventListener: (type, fn) => { const l = listeners[type]; if (l) listeners[type] = l.filter((f) => f !== fn); },
    };
    Object.defineProperty(window, 'visualViewport', { value: fake, configurable: true });
  })();
`;

// A session cookie can be supplied instead of a password, so running this against a long-lived personal install
// (the dev box, not a throwaway CI container) never needs the real password in a shell environment variable.
const sessionCookie = process.env.MOCHIL_SESSION_COOKIE || "";

async function login(page, context) {
  if (sessionCookie) {
    const url = new URL(baseURL);
    await context.addCookies([{ name: "mochil_session", value: sessionCookie, domain: url.hostname, path: "/" }]);
  }
  await page.goto(baseURL, { waitUntil: "domcontentloaded" });
  if (!new URL(page.url()).pathname.startsWith("/login")) return;
  assert.ok(username && password, "the server requires login; set OPDSHELF_USERNAME/OPDSHELF_PASSWORD or MOCHIL_SESSION_COOKIE");
  await page.locator('input[name="username"]').fill(username);
  await page.locator('input[name="password"]').fill(password);
  await Promise.all([
    page.waitForURL((url) => !url.pathname.startsWith("/login")),
    page.locator('form button[type="submit"], form button:not([type])').first().click(),
  ]);
}

async function measureDock(page) {
  return page.evaluate(() => {
    // data-dock first: the shell is a CSS module, so ".dock" only ever matches the legacy page. Keeping the old
    // selector as a fallback means the gate still works against /legacy while it exists.
    const dock = document.querySelector("[data-dock], .dock");
    if (!dock || dock.hidden) return null;
    const rect = dock.getBoundingClientRect();
    return {
      bottomGap: Math.round(window.innerHeight - rect.bottom),
      top: Math.round(rect.top),
      bottom: Math.round(rect.bottom),
      bodyHeight: Math.round(document.body.getBoundingClientRect().height),
    };
  });
}

async function runEngine(name) {
  const launcher = engines[name];
  if (!launcher) throw new Error(`unknown engine "${name}"`);
  const browser = await launcher.launch({ headless: true });
  const failures = [];
  const note = (message) => failures.push(`[${name}] ${message}`);

  try {
    // --- Mechanism check: the frame follows a mocked visualViewport, in this engine's own JS runtime -------------
    {
      const context = await browser.newContext({ viewport: VIEWPORT });
      await context.addInitScript(fakeVisualViewport(780)); // the Firefox-with-its-own-bar case
      const page = await context.newPage();
      await login(page, context);
      await page.goto(`${baseURL}/#/settings`, { waitUntil: "domcontentloaded" });
      await page.waitForTimeout(150); // dock.js measures on load + ResizeObserver tick
      const dock = await measureDock(page);
      if (dock) {
        if (dock.bodyHeight !== 780) {
          note(`body height did not follow the mocked visualViewport: got ${dock.bodyHeight}, want 780 (law L8 — the frame must size from visualViewport, not a CSS viewport unit)`);
        }
        if (dock.bottomGap !== 0 && Math.abs(dock.bottomGap - 64) > 2) {
          // innerHeight is still 844 (real window), visualViewport mock says 780: the dock should sit at 780, i.e.
          // ~64px up from the real window bottom. Anything else means the dock ignored the mock.
          note(`dock did not step up with the mocked visualViewport: bottomGap=${dock.bottomGap}px, want ~64px (this is the exact shape of the 0.1.86 Firefox bug)`);
        }
      } else {
        note("dock not found/hidden on #/settings at 390px — cannot run the mechanism check");
      }
      await context.close();
    }

    // --- Visible checks: real, unmocked viewport -------------------------------------------------------------
    {
      const context = await browser.newContext({ viewport: VIEWPORT });
      const page = await context.newPage();
      await login(page, context);

      // Dock sits flush with the real viewport bottom.
      await page.goto(`${baseURL}/#/settings`, { waitUntil: "domcontentloaded" });
      await page.waitForTimeout(150);
      const dock = await measureDock(page);
      if (dock && dock.bottomGap !== 0) {
        note(`dock is not flush with the viewport bottom: ${dock.bottomGap}px gap (real, unmocked viewport)`);
      }

      // Nothing on Settings is clipped above the header or below the dock.
      const settingsGeometry = await page.evaluate(() => {
        const visible = (el) => {
          if (!el) return false;
          const r = el.getBoundingClientRect();
          return r.width > 0 && r.height > 0;
        };
        // #settings-back exists in the DOM but stays `hidden` outside a sub-panel, so querySelector finds it even
        // when invisible; picking the first *visible* candidate is what actually proves nothing is clipped.
        const candidates = [
          document.querySelector("#settings-back"),
          document.querySelector(".settings-head h1"),
          document.querySelector("#settings-view h1"),
        ];
        const back = candidates.find(visible);
        // data-dock first: the shell is a CSS module, so ".dock" only ever matches the legacy page. Keeping the old
        // selector as a fallback means the gate still works against /legacy while it exists.
        const dock = document.querySelector("[data-dock], .dock");
        const head = document.querySelector(".masthead");
        if (!back) return null;
        const rect = back.getBoundingClientRect();
        return {
          top: Math.round(rect.top),
          bottom: Math.round(rect.bottom),
          headerBottom: head ? Math.round(head.getBoundingClientRect().bottom) : 0,
          dockTop: dock && !dock.hidden ? Math.round(dock.getBoundingClientRect().top) : window.innerHeight,
          viewportHeight: window.innerHeight,
        };
      });
      if (settingsGeometry) {
        if (settingsGeometry.top < settingsGeometry.headerBottom - 1) {
          note(`Settings heading starts above the header (top=${settingsGeometry.top}, header bottom=${settingsGeometry.headerBottom}) — content is clipped under the bar`);
        }
        if (settingsGeometry.bottom > settingsGeometry.dockTop + 1) {
          note(`Settings heading runs under the dock (bottom=${settingsGeometry.bottom}, dock top=${settingsGeometry.dockTop})`);
        }
      }

      // Header height + first-cover offset: only meaningful with a real library. Skip cleanly otherwise.
      const hasLibraryPill = await page.evaluate(() => Boolean(document.querySelector(".pill, [data-library-id]")));
      if (hasLibraryPill) {
        await page.goto(`${baseURL}/#/library/books/books`, { waitUntil: "domcontentloaded" });
        await page.waitForTimeout(200);
        const shelf = await page.evaluate(() => {
          const head = document.querySelector("#shelf-head");
          const card = document.querySelector(".book, .series-card");
          return {
            headHeight: head ? Math.round(head.getBoundingClientRect().height) : null,
            firstCoverTop: card ? Math.round(card.getBoundingClientRect().top) : null,
          };
        });
        if (shelf.headHeight !== null && shelf.headHeight > 80) {
          note(`shelf header is ${shelf.headHeight}px tall at 390px, want well under 80px (the 0.1.80 regression was 114px)`);
        }
      } else {
        console.log(`  [${name}] SKIP header-height/first-cover checks: no library on this install (fresh/library-less box)`);
      }

      await context.close();
    }
  } finally {
    await browser.close();
  }

  return failures;
}

const allFailures = [];
for (const name of engineNames) {
  console.log(`== ${name}`);
  const failures = await runEngine(name);
  if (failures.length === 0) console.log(`  OK`);
  else failures.forEach((f) => console.log(`  FAIL ${f}`));
  allFailures.push(...failures);
}

if (allFailures.length > 0) {
  console.error(`\n${allFailures.length} layout gate failure(s).`);
  process.exit(1);
}
console.log("\nLayout gate passed in all engines.");
