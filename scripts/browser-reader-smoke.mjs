// Opens one EPUB in both reading engines and checks the core reading flow:
// it renders, pages turn from the edge and the keyboard, the scrubber seeks,
// and the position survives a reload. Set BOBOBOOKS_READER_BOOK=<library>/<id>
// (or let it pick the first EPUB of BOBOBOOKS_READER_LIBRARY).
import assert from "node:assert/strict";
import { chromium } from "playwright";

const baseURL = process.env.BOBOBOOKS_BASE_URL || "http://127.0.0.1:8099";
const executablePath = process.env.CHROMIUM_PATH || "/usr/bin/chromium";

async function pickBook() {
  if (process.env.BOBOBOOKS_READER_BOOK) return process.env.BOBOBOOKS_READER_BOOK;
  const library = process.env.BOBOBOOKS_READER_LIBRARY || "books";
  const response = await fetch(`${baseURL}/api/books?library=${encodeURIComponent(library)}&limit=300`);
  assert.equal(response.status, 200, "book list must load");
  const book = (await response.json()).find((item) => item.format === "epub" && Number(item.size) > 300000 && Number(item.size) < 8000000);
  assert.ok(book, "no mid-sized EPUB found; set BOBOBOOKS_READER_BOOK");
  return `${library}/${book.id}`;
}

const book = await pickBook();
const browser = await chromium.launch({ executablePath, headless: true });
const problems = [];

async function session(engine, run) {
  const context = await browser.newContext({ viewport: { width: 1100, height: 800 } });
  const page = await context.newPage();
  page.on("pageerror", (error) => problems.push(`${engine}: ${error.message}`));
  await page.goto(`${baseURL}/reader/${book}?engine=${engine}`, { waitUntil: "domcontentloaded" });
  try {
    await run(page);
  } finally {
    await context.close();
  }
}

const settle = (page, ms = 1100) => page.waitForTimeout(ms);

try {
  await session("classic", async (page) => {
    const fraction = () => page.evaluate(() => document.querySelector("#reader").lastLocation?.fraction);
    await page.waitForFunction(() => document.querySelector("#reader")?.lastLocation?.fraction != null, null, { timeout: 30000 });
    await page.evaluate(() => document.querySelector("#reader").goToFraction(0.4));
    await settle(page, 1800);
    const start = await fraction();
    await page.mouse.click(1050, 400);
    await settle(page);
    const forward = await fraction();
    assert.ok(forward > start, "classic: a right-edge click turns the page forward");
    await page.mouse.click(40, 400);
    await settle(page);
    assert.ok((await fraction()) < forward, "classic: a left-edge click turns back");
    await page.evaluate(() => { const seek = document.querySelector("#seek"); seek.value = 700; seek.dispatchEvent(new Event("change", { bubbles: true })); });
    await settle(page, 1800);
    assert.ok(Math.abs((await fraction()) - 0.7) < 0.03, "classic: the scrubber seeks");
  });

  await session("readium", async (page) => {
    await page.waitForFunction(() => window.__readium?.navigator, null, { timeout: 30000 });
    const fraction = () => page.evaluate(() => window.__readium.navigator.currentLocator.locations.totalProgression);
    await page.evaluate(() => { const seek = document.querySelector("#seek"); seek.value = 400; seek.dispatchEvent(new Event("change", { bubbles: true })); });
    await settle(page, 1800);
    const start = await fraction();
    await page.keyboard.press("ArrowRight");
    await settle(page);
    assert.ok((await fraction()) > start, "readium: the arrow key turns the page");
    const saved = await fraction();
    await page.reload({ waitUntil: "domcontentloaded" });
    await page.waitForFunction(() => window.__readium?.navigator, null, { timeout: 30000 });
    await settle(page, 1500);
    assert.ok(Math.abs((await fraction()) - saved) < 0.01, "readium: the position survives a reload");
    assert.ok((await page.locator("#toc-list button").count()) > 0, "readium: the contents list is filled");
  });
} finally {
  await browser.close();
}

assert.deepEqual(problems, [], "the readers raised browser errors");
console.log(`reader smoke passed for ${book}`);
