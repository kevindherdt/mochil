// scripts/capture-screenshots.mjs — regenerate the pictures in README.md from the running app.
//
// The screenshots in the repository go stale every time the interface moves, and stale pictures are worse than none:
// they describe an app that no longer exists. This walks the pages the README shows, at the two widths it shows them,
// in the two themes it shows, and writes them over screenshots/.
//
// It is meant to be pointed at the disposable demo (`./scripts/ui-demo.sh`), never at a real library: the covers in a
// real library are other people's artwork, and a repository is the last place that should be republished. The demo's
// books are invented and its covers are drawn, so nothing in the pictures belongs to anyone.
//
//   ./scripts/ui-demo.sh                                   # invented books, port 18399
//   node web/scripts/capture-screenshots.mjs               # writes screenshots/*.png
//   ./scripts/ui-demo.sh --stop
//
// MOCHIL_BASE_URL, MOCHIL_USERNAME and MOCHIL_PASSWORD override the demo's defaults.
import { chromium } from "playwright";
import { mkdir } from "node:fs/promises";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const outDir = join(here, "..", "..", "screenshots");
const baseURL = (process.env.MOCHIL_BASE_URL ?? "http://127.0.0.1:18399").replace(/\/$/, "");
const username = process.env.MOCHIL_USERNAME ?? "ui";
const password = process.env.MOCHIL_PASSWORD ?? "ui-sweep-password-1";

const desktop = { width: 1440, height: 900 };
const phone = { width: 390, height: 844 };

// name, path, viewport, theme, mode. The two themes are shown in opposite modes on purpose: it is the clearest single
// way to say that a theme is a palette and the light/dark scheme is separate from it.
const shots = [
  { name: "web-library", path: "/library", viewport: desktop, theme: "kiln", mode: "dark" },
  { name: "web-mobile-library", path: "/library", viewport: phone, theme: "kiln", mode: "dark" },
  { name: "web-settings", path: "/settings", viewport: desktop, theme: "kiln", mode: "dark" },
  { name: "theme-magenta-cream", path: "/library", viewport: desktop, theme: "magenta-cream", mode: "light" },
  { name: "theme-magenta-cream-phone", path: "/library", viewport: phone, theme: "magenta-cream", mode: "light" },
];

await mkdir(outDir, { recursive: true });
const browser = await chromium.launch({
  executablePath: process.env.CHROMIUM_PATH ?? "/usr/bin/chromium",
  headless: true,
  args: ["--no-sandbox"],
});

try {
  for (const shot of shots) {
    const context = await browser.newContext({ viewport: shot.viewport, deviceScaleFactor: 1, colorScheme: "dark" });
    // The theme lives as one JSON value (`{ palette, mode }`) under "mochil:theme"; writing a bare string is what the
    // app did before themes had a light/dark mode, and it is silently ignored now.
    await context.addInitScript(
      ([palette, mode]) => localStorage.setItem("mochil:theme", JSON.stringify({ palette, mode })),
      [shot.theme, shot.mode],
    );
    const page = await context.newPage();
    page.setDefaultTimeout(15000);
    await page.goto(baseURL + "/login", { waitUntil: "networkidle" });
    if (new URL(page.url()).pathname.startsWith("/login")) {
      await page.locator('input[name="username"]').fill(username);
      await page.locator('input[name="password"]').fill(password);
      await Promise.all([
        page.waitForURL((url) => !url.pathname.startsWith("/login")),
        page.locator('button[type="submit"]').click(),
      ]);
    }
    await page.goto(baseURL + shot.path, { waitUntil: "networkidle" });
    // Covers fade in; waiting for the count to settle beats a fixed sleep, which either wastes time or catches a
    // half-drawn grid.
    await page.waitForFunction(() => {
      const images = [...document.querySelectorAll("img")];
      return images.length > 0 && images.every((image) => image.complete);
    }, null, { timeout: 15000 }).catch(() => {});
    await page.waitForTimeout(700);
    await page.screenshot({ path: join(outDir, `${shot.name}.png`) });
    console.log(`captured ${shot.name}.png`);
    await context.close();
  }
} finally {
  await browser.close();
}
