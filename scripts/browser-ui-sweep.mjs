import { mkdir, writeFile } from "node:fs/promises";
import { chromium } from "playwright";

// Look at every page of Mochil, at three widths and in both themes, and write down what is measurably
// wrong on each one. This exists because "the interface looks off" was being answered by reading CSS
// and guessing: the pages that were actually broken were found by the person using them. A pass that
// walks the whole app, screenshots it and measures it turns that into something repeatable.
//
//   ./scripts/ui-demo.sh                      # a disposable server with invented books
//   node scripts/browser-ui-sweep.mjs         # walk it, screenshot it, measure it
//
// Screenshots and a report land in $UI_SWEEP_OUT (default /tmp/mochil-ui-sweep). The report says what
// was measured; the screenshots are for looking at, which no measurement replaces.
const baseURL = process.env.MOCHIL_BASE_URL || "http://127.0.0.1:18399";
const username = process.env.MOCHIL_USERNAME || "ui";
const password = process.env.MOCHIL_PASSWORD || "ui-sweep-password-1";
const out = process.env.UI_SWEEP_OUT || "/tmp/mochil-ui-sweep";
const executablePath = process.env.CHROMIUM_PATH || "/usr/bin/chromium";
const viewports = [
  { name: "desktop", width: 1440, height: 900 },
  { name: "tablet", width: 834, height: 1112 },
  { name: "phone", width: 390, height: 844 },
];

await mkdir(out, { recursive: true });
const browser = await chromium.launch({ executablePath, headless: true });
const report = { startedAt: new Date().toISOString(), baseURL, surfaces: [] };

try {
  for (const theme of ["dark", "light"]) {
    for (const viewport of viewports) {
      const context = await browser.newContext({ viewport, colorScheme: theme, deviceScaleFactor: 1 });
      await context.addInitScript((value) => localStorage.setItem("mochil-theme", value), theme);
      const page = await context.newPage();
      page.setDefaultTimeout(6000);
      const pageErrors = [];
      page.on("pageerror", (error) => pageErrors.push(error.message));
      try {
        await page.goto(baseURL, { waitUntil: "networkidle" });
        await signIn(page);
        for (const surface of surfaces()) {
          const name = `${viewport.name}-${theme}-${surface.slug}`;
          let surfacePage = page;
          if (surface.fresh) {
            surfacePage = await context.newPage();
            surfacePage.setDefaultTimeout(6000);
            await surfacePage.goto(baseURL, { waitUntil: "networkidle" });
          }
          try {
            if ((await surface.open(surfacePage)) === "absent") {
              report.surfaces.push({ name, theme, viewport: viewport.name, surface: surface.slug, absent: true });
              continue;
            }
            await surfacePage.waitForTimeout(surface.settle ?? 400);
            await surfacePage.screenshot({ path: `${out}/${name}.png`, fullPage: false });
            report.surfaces.push({ name, theme, viewport: viewport.name, surface: surface.slug, ...(await measure(surfacePage, viewport)) });
          } catch (error) {
            report.surfaces.push({ name, theme, viewport: viewport.name, surface: surface.slug, failed: error.message });
          } finally {
            if (surface.fresh && surfacePage !== page) await surfacePage.close();
          }
        }
      } finally {
        report.surfaces.push({ name: `${viewport.name}-${theme}-pageerrors`, pageErrors });
        await context.close();
      }
    }
  }
} finally {
  await browser.close();
}

report.finishedAt = new Date().toISOString();
await writeFile(`${out}/report.json`, JSON.stringify(report, null, 2));
printSummary(report, out);

function surfaces() {
  return [
    { slug: "home", open: (page) => goto(page, "/") },
    { slug: "library-books", open: (page) => openLibrary(page, /library/i) },
    { slug: "library-comics", open: (page) => openLibrary(page, /comics/i) },
    { slug: "search", open: async (page) => { await goto(page, "/"); await runSearch(page, "salt"); } },
    { slug: "settings-index", open: (page) => goto(page, "/#/settings"), settle: 1000 },
    ...settingsPanels(),
    { slug: "import", open: (page) => goto(page, "/import"), settle: 900 },
    { slug: "reader", open: openReader, settle: 1400 },
    { slug: "not-found", open: (page) => goto(page, "/this-page-does-not-exist") },
  ];
}

// Every settings page, by the card title the index shows. A panel that fails to open is reported as a
// finding rather than skipped, because a settings page nobody can reach is a defect too.
function settingsPanels() {
  // The tab id, not the label: the index re-renders when the module list arrives, and a click aimed at
  // a button found by text lands on an element that is being replaced.
  const panels = [
    ["libraries", "Libraries"],
    ["modules", "Modules"],
    ["hermes", "Hermes"],
    ["avatar", "Avatar"],
    ["sync", "Devices & sync"],
    ["playback", "Playback"],
    ["narration", "Narration"],
    ["appearance", "Appearance"],
    ["accounts", "Accounts"],
    ["diagnostics", "Diagnostics"],
  ];
  return panels.map(([tab, title]) => ({
    slug: `settings-${tab}`,
    settle: 800,
    fresh: true,
    open: async (page) => {
      // A full load at /#/settings is the only entry that reliably shows the index: reaching Settings
      // through the library switcher opens the first panel instead, and the index stays hidden.
      await goto(page, "/#/settings");
      await page.waitForTimeout(600);
      const card = page.locator(`.settings-index button[data-settings-tab="${tab}"]`).first();
      if (!(await card.count())) return "absent"; // the module that owns this page is switched off
      await card.click();
      await page.waitForTimeout(500);
    },
  }));
}

async function goto(page, path) {
  await page.goto(baseURL + path, { waitUntil: "domcontentloaded" });
  await page.waitForTimeout(400);
}

async function signIn(page) {
  if (!new URL(page.url()).pathname.startsWith("/login")) return;
  await page.locator('input[name="username"]').fill(username);
  await page.locator('input[name="password"]').fill(password);
  await Promise.all([
    page.waitForURL((url) => !url.pathname.startsWith("/login")),
    page.locator('button[type="submit"]').click(),
  ]);
  await page.waitForLoadState("networkidle");
}

async function openLibrary(page, pattern) {
  await goto(page, "/");
  const option = page.locator("#filters option").filter({ hasText: pattern }).first();
  if (!(await option.count())) throw new Error(`no library matching ${pattern}`);
  await page.locator("#filters").selectOption(await option.getAttribute("value"));
  await page.waitForFunction(() => !/opening|loading/i.test(document.querySelector("#status")?.textContent || ""), null, { timeout: 15000 });
  await page.waitForTimeout(600);
}

async function runSearch(page, query) {
  const box = page.locator('#search input[type="search"], input[type="search"]').first();
  if (!(await box.count())) throw new Error("no search field on the page");
  await box.fill(query);
  await box.press("Enter");
  await page.waitForTimeout(1200);
}

async function openReader(page) {
  await openLibrary(page, /library/i);
  const all = page.locator(".publication-destinations button").filter({ hasText: /^all$/i }).first();
  if (await all.count()) {
    await all.click();
    await page.waitForTimeout(900);
  }
  const book = page.locator("#book-catalog .book-open, #book-catalog .book a").first();
  if (!(await book.count())) throw new Error("no book to open");
  await book.click();
  await page.waitForTimeout(1500);
}

// Measurements a person would otherwise make by eye. Each is conservative on purpose: a false alarm
// costs a wasted look, but a noisy report stops being read at all.
async function measure(page, viewport) {
  return page.evaluate(({ mobile }) => {
    const describe = (element) => {
      const id = element.id ? `#${element.id}` : "";
      const classes = [...(element.classList || [])].slice(0, 2).map((c) => `.${c}`).join("");
      return `${element.tagName.toLowerCase()}${id}${classes}`;
    };
    const visible = (element) => {
      const style = getComputedStyle(element);
      const rect = element.getBoundingClientRect();
      return style.visibility !== "hidden" && style.display !== "none" && rect.width > 0 && rect.height > 0;
    };

    const overflow = Math.max(0, document.documentElement.scrollWidth - window.innerWidth);

    // Controls too small to hit with a thumb, only where a thumb is what is being used.
    const smallTargets = [];
    if (mobile) {
      for (const element of document.querySelectorAll("button, a, [role=button], input[type=checkbox], input[type=radio], select")) {
        if (!visible(element)) continue;
        const rect = element.getBoundingClientRect();
        if (rect.width < 40 || rect.height < 40) {
          if (rect.width * rect.height > 40 * 40 * 4) continue; // wide inline links are not thumb targets
          smallTargets.push(`${describe(element)} ${Math.round(rect.width)}x${Math.round(rect.height)}`);
        }
      }
    }

    // A control whose name is only a glyph: a screen reader and a person new to the app both read nothing.
    const unlabelled = [];
    for (const element of document.querySelectorAll("button, [role=button], a")) {
      if (!visible(element)) continue;
      const text = (element.textContent || "").trim();
      const label = element.getAttribute("aria-label") || element.getAttribute("title") || element.querySelector("img[alt]")?.getAttribute("alt") || "";
      if (!text && !label) unlabelled.push(describe(element));
    }

    // Text that runs past a clipping ancestor is text nobody can read, and it never throws, so it is
    // invisible in every log.
    const clipped = [];
    for (const element of document.querySelectorAll("p, span, h1, h2, h3, li, td, label, .book-title, .book-author")) {
      if (!visible(element) || (element.textContent || "").trim().length < 8) continue;
      let ancestor = element.parentElement;
      while (ancestor && ancestor !== document.body) {
        const style = getComputedStyle(ancestor);
        if (/(hidden|clip)/.test(`${style.overflow}${style.overflowX}${style.overflowY}`) && !/scroll|auto/.test(`${style.overflowX}${style.overflowY}`)) {
          const box = ancestor.getBoundingClientRect();
          const rect = element.getBoundingClientRect();
          if (rect.right > box.right + 2 || rect.bottom > box.bottom + 2) {
            clipped.push(`${describe(element)} past ${describe(ancestor)}`);
            break;
          }
        }
        ancestor = ancestor.parentElement;
      }
    }

    // Contrast of rendered text against what is actually behind it, walked up through transparent
    // backgrounds. WCAG AA for body text is 4.5, for large text 3.
    const parse = (color) => {
      const match = color.match(/rgba?\(([^)]+)\)/);
      if (!match) return null;
      const [r, g, b, a = "1"] = match[1].split(/[ ,/]+/).filter(Boolean);
      return { r: +r, g: +g, b: +b, a: +a };
    };
    const luminance = ({ r, g, b }) => {
      const channel = (value) => {
        const v = value / 255;
        return v <= 0.03928 ? v / 12.92 : Math.pow((v + 0.055) / 1.055, 2.4);
      };
      return 0.2126 * channel(r) + 0.7152 * channel(g) + 0.0722 * channel(b);
    };
    const background = (element) => {
      let node = element;
      while (node) {
        const color = parse(getComputedStyle(node).backgroundColor);
        if (color && color.a > 0.5) return color;
        node = node.parentElement;
      }
      return { r: 0, g: 0, b: 0, a: 1 };
    };
    const lowContrast = [];
    for (const element of document.querySelectorAll("p, span, small, h1, h2, h3, a, button, label, td")) {
      if (!visible(element)) continue;
      if (element.children.length) continue;
      const text = (element.textContent || "").trim();
      if (text.length < 8) continue;
      const style = getComputedStyle(element);
      const ink = parse(style.color);
      if (!ink) continue;
      const back = background(element);
      const ratio = (Math.max(luminance(ink), luminance(back)) + 0.05) / (Math.min(luminance(ink), luminance(back)) + 0.05);
      const size = parseFloat(style.fontSize) || 16;
      const bold = (parseInt(style.fontWeight, 10) || 400) >= 700;
      const large = size >= 24 || (size >= 18.66 && bold);
      if (ratio < (large ? 3 : 4.5)) lowContrast.push(`${describe(element)} ${ratio.toFixed(2)}:1 "${text.slice(0, 40)}"`);
    }

    return {
      overflow,
      smallTargets: [...new Set(smallTargets)].slice(0, 12),
      unlabelled: [...new Set(unlabelled)].slice(0, 12),
      clipped: [...new Set(clipped)].slice(0, 12),
      lowContrast: [...new Set(lowContrast)].slice(0, 12),
      title: document.title,
    };
  }, { mobile: viewport.width < 600 });
}

function printSummary(result, directory) {
  const lines = [];
  const problems = result.surfaces.filter(
    (surface) => !surface.failed && (surface.overflow > 0 || surface.smallTargets?.length || surface.unlabelled?.length || surface.clipped?.length || surface.lowContrast?.length),
  );
  const failures = result.surfaces.filter((surface) => surface.failed);
  const errors = result.surfaces.filter((surface) => surface.pageErrors?.length);

  for (const surface of problems) {
    const parts = [];
    if (surface.overflow > 0) parts.push(`overflow ${surface.overflow}px`);
    if (surface.smallTargets?.length) parts.push(`${surface.smallTargets.length} small targets (${surface.smallTargets[0]})`);
    if (surface.unlabelled?.length) parts.push(`${surface.unlabelled.length} unlabelled (${surface.unlabelled[0]})`);
    if (surface.clipped?.length) parts.push(`${surface.clipped.length} clipped (${surface.clipped[0]})`);
    if (surface.lowContrast?.length) parts.push(`${surface.lowContrast.length} low contrast (${surface.lowContrast[0]})`);
    lines.push(`${surface.name}: ${parts.join("; ")}`);
  }
  for (const failure of failures) lines.push(`${failure.name}: COULD NOT OPEN (${failure.failed})`);
  for (const error of errors) lines.push(`${error.name}: page error (${error.pageErrors[0]})`);

  console.log(`${result.surfaces.filter((s) => !s.failed && s.surface).length} pages captured into ${directory}`);
  console.log(lines.length ? lines.join("\n") : "nothing measurable was wrong");
  if (failures.length) console.log(`\n${failures.length} page(s) could not be opened at all`);
}
