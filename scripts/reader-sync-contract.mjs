// scripts/reader-sync-contract.mjs — the automated test a ported reader would have to pass.
//
// The reason the reader stayed legacy was not its UI: it was that position sync "did not work right" and nobody could say so
// faster than by hand. This script turns that complaint into assertions. It drives the real reader in a real browser and
// checks the four things a replacement must do:
//
//   1. the reader writes its position to /api/reading/<library>/<book>;
//   2. the record carries what the other clients need (a locator, and a KoSync document when the server has one);
//   3. a second, independent session sees that position and resumes there;
//   4. the position survives a reload.
//
// It restores the position it found before it changed anything, so it is safe to run against a real library.
//
// Auth: pass MOCHIL_COOKIE (a ready cookie header) or MOCHIL_E2E_USER / MOCHIL_E2E_PASS for a form login. With neither, the
// script prints why and exits 0 — a missing credential is not a failing test, but it is never a passing one either.
import assert from "node:assert/strict";
import { createRequire } from "node:module";
import { readFileSync } from "node:fs";
// Playwright is installed for the web workspace, not at the repository root; resolving it from there means this script runs
// from a plain checkout without a second install.
const { chromium } = createRequire(new URL("../web/package.json", import.meta.url))("playwright");

const baseURL = process.env.MOCHIL_BASE_URL || "http://127.0.0.1:8099";
const executablePath = process.env.CHROMIUM_PATH || "/usr/bin/chromium";
// A cookie header can come straight from the environment or from a file (0600), which keeps it out of `ps`.
const cookieFile = process.env.MOCHIL_COOKIE_FILE || "";
const cookie = process.env.MOCHIL_COOKIE || (cookieFile ? readFileSync(cookieFile, "utf8").trim() : "");
const user = process.env.MOCHIL_E2E_USER || "";
const pass = process.env.MOCHIL_E2E_PASS || "";

if (!cookie && !(user && pass)) {
  console.log("reader-sync-contract: no MOCHIL_COOKIE or MOCHIL_E2E_USER/PASS, so nothing was proven. Set one to run it.");
  process.exit(0);
}

const cookieHeader = () => {
  if (!cookie) throw new Error("cookie mode expected");
  return cookie;
};

async function authedFetch(path, init = {}, cookieJar) {
  const headers = { ...(init.headers || {}) };
  if (cookieJar) headers.cookie = cookieJar;
  const response = await fetch(`${baseURL}${path}`, { ...init, headers, redirect: "manual" });
  return response;
}

async function pickBook(cookieJar) {
  if (process.env.MOCHIL_READER_BOOK) return process.env.MOCHIL_READER_BOOK;
  const library = process.env.MOCHIL_READER_LIBRARY || "books";
  const response = await authedFetch(`/api/books?library=${encodeURIComponent(library)}&limit=300`, {}, cookieJar);
  assert.equal(response.status, 200, "the book list must load with the credentials given");
  const books = await response.json();
  const book = books.find((item) => item.format === "epub" && Number(item.size) > 300000 && Number(item.size) < 8000000);
  assert.ok(book, "no mid-sized EPUB found; set MOCHIL_READER_BOOK=<library>/<id>");
  return `${library}/${book.id}`;
}

const readPosition = async (cookieJar, book) => {
  const response = await authedFetch(`/api/reading/${book}`, {}, cookieJar);
  if (response.status === 404) return null;
  assert.equal(response.status, 200, `reading position must load (got ${response.status})`);
  return response.json();
};

const turnPages = async (page, times) => {
  for (let index = 0; index < times; index += 1) {
    await page.keyboard.press("ArrowRight");
    await page.waitForTimeout(700);
  }
};

/** Waits for a condition the server has to confirm, rather than trusting what the page says about itself. */
async function until(what, probe, timeoutMs = 20000) {
  const deadline = Date.now() + timeoutMs;
  let last;
  while (Date.now() < deadline) {
    last = await probe();
    if (last !== undefined && last !== null && last !== false) return last;
    await new Promise((resolve) => setTimeout(resolve, 500));
  }
  throw new Error(`timed out waiting for ${what}`);
}

let browser;
try {
  browser = await chromium.launch({ executablePath, headless: true });
  const problems = [];

  // ---- One session, used to log in when only a username and password were given.
  let cookieJar = cookie;
  if (!cookieJar) {
    const loginContext = await browser.newContext();
    const login = await loginContext.newPage();
    await login.goto(`${baseURL}/login`, { waitUntil: "domcontentloaded" });
    await login.fill('input[name="username"]', user);
    await login.fill('input[name="password"]', pass);
    await Promise.all([login.waitForLoadState("networkidle"), login.click('button[type="submit"], button:has-text("Sign in")')]);
    const cookies = await loginContext.cookies();
    cookieJar = cookies.map((entry) => `${entry.name}=${entry.value}`).join("; ");
    await loginContext.close();
    assert.ok(cookieJar, "the form login produced no session cookie");
  }

  const book = await pickBook(cookieJar);
  const before = await readPosition(cookieJar, book);
  console.log(`reader-sync-contract: ${book}, starts at ${before ? `fraction ${Number(before.fraction).toFixed(4)} (revision ${before.revision})` : "no saved position"}`);

  const context = await browser.newContext({ viewport: { width: 1100, height: 800 } });
  await context.addCookies(
    cookieJar.split("; ").map((pair) => {
      const [name, ...rest] = pair.split("=");
      return { name, value: rest.join("="), url: baseURL };
    }),
  );
  const page = await context.newPage();
  page.on("pageerror", (error) => problems.push(error.message));
  await page.goto(`${baseURL}/reader/${book}`, { waitUntil: "domcontentloaded" });

  // The reader mounts the engine inside a frame; wait for a page of text rather than a fixed sleep.
  await until("the reader to render a page", async () => {
    const frames = page.frames().filter((frame) => frame !== page.mainFrame());
    for (const frame of frames) {
      const text = await frame.evaluate(() => document.body?.innerText?.length ?? 0).catch(() => 0);
      if (text > 200) return true;
    }
    return false;
  }, 30000);

  await turnPages(page, 4);

  // (1) The reader wrote, and (2) the record carries a locator the other clients can use.
  const written = await until("the reader to write its position to /api/reading", async () => {
    const current = await readPosition(cookieJar, book);
    if (!current) return false;
    const advanced = !before || Number(current.fraction) > Number(before.fraction) + 1e-6 || current.revision !== before.revision;
    return advanced ? current : false;
  });
  console.log(`reader-sync-contract: reader wrote fraction ${Number(written.fraction).toFixed(4)} (revision ${written.revision})`);
  // The shape the server actually stores: a text anchor (prefix/exact/suffix plus the spine index) and the engine's own
  // locator. Looking for a single "locator" field is what this assertion did first, and it failed a record that had both.
  const anchor = written.web_anchor?.exact ? written.web_anchor : undefined;
  const engineLocator = written.locators?.readium || written.koreader_locator;
  assert.ok(
    anchor || engineLocator,
    `the saved position must carry a locator, or no other client can resume from it (got ${JSON.stringify(Object.keys(written))})`,
  );
  console.log(
    `reader-sync-contract: locator present (${anchor ? `text anchor, spine ${anchor.spine_index}` : "engine locator"}${engineLocator ? ", plus the engine's own" : ""})`,
  );
  if (written.kosync_document) {
    console.log(`reader-sync-contract: linked to KoSync document ${written.kosync_document}`);
  } else {
    console.log("reader-sync-contract: no KoSync document on this book yet (that is a link step, not a sync failure)");
  }

  // (3) A second, independent session sees the same position and resumes there.
  const otherContext = await browser.newContext({ viewport: { width: 1100, height: 800 } });
  await otherContext.addCookies(
    cookieJar.split("; ").map((pair) => {
      const [name, ...rest] = pair.split("=");
      return { name, value: rest.join("="), url: baseURL };
    }),
  );
  const other = await otherContext.newPage();
  await other.goto(`${baseURL}/reader/${book}`, { waitUntil: "domcontentloaded" });
  const resumed = await until("the second session to open at the shared position", async () => {
    const inside = other.frames().filter((frame) => frame !== other.mainFrame());
    for (const frame of inside) {
      // The reader marks the live position on the document element as --progress; anything near the stored fraction is a
      // resume, and a fresh open would be at zero.
      const progress = await frame.evaluate(() => {
        const own = Number(getComputedStyle(document.documentElement).getPropertyValue("--progress"));
        const parent = Number(
          window.parent?.document?.documentElement
            ? getComputedStyle(window.parent.document.documentElement).getPropertyValue("--progress")
            : NaN,
        );
        return Number.isFinite(parent) ? parent : own;
      }).catch(() => NaN);
      if (Number.isFinite(progress) && progress > 0.01) return progress;
    }
    return false;
  }, 30000);
  console.log(`reader-sync-contract: a second session resumed at ${(Number(resumed) * 100).toFixed(1)}%`);
  await otherContext.close();

  // (4) The position survives a reload of the first session.
  await page.reload({ waitUntil: "domcontentloaded" });
  const afterReload = await until("the position to survive a reload", async () => {
    const current = await readPosition(cookieJar, book);
    return current && Math.abs(Number(current.fraction) - Number(written.fraction)) < 0.05 ? current : false;
  });
  assert.ok(afterReload, "the position must survive a reload");

  assert.deepEqual(problems, [], `the reader raised page errors: ${problems.join("; ")}`);

  // Put back what was there, so running this against a real library does not move a reading position.
  if (before) {
    const restore = await authedFetch(`/api/reading/${book}`, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        cfi: before.cfi,
        fraction: before.fraction,
        device_id: before.device_id,
        device_name: before.device_name,
        base_revision: written.revision,
        web_anchor: before.web_anchor,
        locators: before.locators,
      }),
    }, cookieJar);
    assert.ok(restore.ok, `restoring the previous position failed with ${restore.status}`);
    console.log("reader-sync-contract: previous position restored");
  }

  await context.close();
  console.log("reader-sync-contract: passed (write, locator, second session, reload)");
} finally {
  await browser?.close();
}
