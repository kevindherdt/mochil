// e2e/shell.spec.ts — the layout laws, falsified in three real engines rather than argued about in reviews.
//
// Every assertion here is one of the shell's promises:
//   * the dock is the last row and stays on screen while the shelf scrolls (the "scrolling down loses the upper
//     pills / the dock" bug class);
//   * the chrome band stays put (law L1: the shell owns positioning, screens flow);
//   * nothing overflows sideways at 390px (the pill row + trailing control bug);
//   * the content can always be scrolled clear of the dock (the last card must not live under it).
//
// The API is fulfilled locally: `contracts/api/*.json` is the *shape* source of truth (the Go contract test checks
// those files against the real handlers), but they hold two books — too few for a scroll test. So the shelf is
// generated from that template: real shapes, enough rows that `main` really scrolls.
import { expect, test, type Page } from "@playwright/test";
import { readFileSync } from "node:fs";
import path from "node:path";

const fixtures = path.resolve(import.meta.dirname, "../../contracts/api");

function fixture<T>(name: string): T {
  return JSON.parse(readFileSync(path.join(fixtures, `${name}.json`), "utf8")) as T;
}

/** 60 books cloned from the contract fixture, so the grid is taller than the viewport. */
function shelf(): unknown[] {
  const [template] = fixture<Record<string, unknown>[]>("books");
  return Array.from({ length: 60 }, (_, index) => ({
    ...template,
    id: `book-${index}`,
    title: `Book ${index}`,
    path: `Book ${index}.epub`,
    cover_url: `/covers/books/book-${index}`,
  }));
}

const DOCK_HEIGHT = 54; // design/tokens.json → chrome.dockHeightPhone; a regression here is a token regression
const PHONE_HEIGHT = 844;

async function signedIn(page: Page): Promise<void> {
  // Only the endpoints the library screen needs, plus modules (the shell asks for them to decide the AI button).
  await page.route("**/api/libraries", (route) => route.fulfill({ json: fixture("libraries") }));
  await page.route("**/api/books**", (route) =>
    route.fulfill({ json: shelf(), headers: { "X-Total-Count": "60" } }),
  );
  await page.route("**/api/modules", (route) => route.fulfill({ json: fixture("modules") }));
  // /api/version has no contract fixture (it reports the running build), and the shell only needs it to answer.
  await page.route("**/api/version", (route) =>
    route.fulfill({ json: { product: "Mochil", version: "0.0.0", commit: "e2e", go: "1", started: "2026-01-01T00:00:00Z" } }),
  );
  await page.goto("/library");
  await expect(page.getByRole("navigation", { name: "Primary" })).toBeVisible();
  // The grid has painted: a card is a button whose accessible name is its title.
  await expect(page.getByRole("button", { name: /^Book 0\b/ })).toBeVisible();
}

test.describe("phone shell", () => {
  test("the dock stays on screen while the shelf scrolls", async ({ page }) => {
    await signedIn(page);
    const dock = page.getByRole("navigation", { name: "Primary" });
    const before = await dock.boundingBox();
    expect(before).not.toBeNull();
    // Bottom edge flush with the viewport: a dock with a gap under it is the Firefox-Android symptom.
    expect(Math.round((before?.y ?? 0) + (before?.height ?? 0))).toBe(PHONE_HEIGHT);

    await page.locator("main").evaluate((main) => main.scrollTo(0, 800));
    await page.waitForTimeout(150); // let the scroll settle; nothing is animated, so this is not a flake guard

    const after = await dock.boundingBox();
    expect(after?.y).toBe(before?.y);

    // The document itself must never scroll: if it did, the dock would ride away with it.
    const docScrolls = await page.evaluate(
      () =>
        document.scrollingElement !== null &&
        document.scrollingElement.scrollHeight > document.scrollingElement.clientHeight + 1,
    );
    expect(docScrolls).toBe(false);
    const mainScrolled = await page.locator("main").evaluate((main) => main.scrollTop);
    expect(mainScrolled).toBeGreaterThan(0);
  });

  test("the chrome band stays put while the shelf scrolls", async ({ page }) => {
    await signedIn(page);
    // The pill's label, not the button: the accessible name of a pill includes its glyph, and what this test
    // cares about is where the chrome sits, not how the pill names itself.
    const pills = page.getByText("Books", { exact: true }).first();
    const before = await pills.boundingBox();
    await page.locator("main").evaluate((main) => main.scrollTo(0, 600));
    await page.waitForTimeout(150);
    const after = await pills.boundingBox();
    expect(after?.y).toBe(before?.y);
    expect(before?.y).toBeLessThan(200); // still up top, not pushed below the fold by a growing band
  });

  test("nothing overflows sideways at 390px", async ({ page }) => {
    await signedIn(page);
    const overflow = await page.evaluate(() => ({
      scrollWidth: document.documentElement.scrollWidth,
      innerWidth: window.innerWidth,
    }));
    expect(overflow.scrollWidth).toBeLessThanOrEqual(overflow.innerWidth);
  });

  test("the dock is the token's height, not a per-engine accident", async ({ page }) => {
    await signedIn(page);
    const dock = await page.getByRole("navigation", { name: "Primary" }).boundingBox();
    expect(Math.round(dock?.height ?? 0)).toBe(DOCK_HEIGHT);
  });

  test("the end of the shelf can be scrolled clear of the dock", async ({ page }) => {
    await signedIn(page);
    const main = page.locator("main");
    await main.evaluate((element) => element.scrollTo(0, element.scrollHeight));
    await page.waitForTimeout(150);
    const dockTop = (await page.getByRole("navigation", { name: "Primary" }).boundingBox())?.y ?? PHONE_HEIGHT;
    const lastCard = await page.getByRole("button", { name: /^Book 59\b/ }).boundingBox();
    expect(lastCard).not.toBeNull();
    // A card may sit above the dock, never underneath it: `main` must reserve the dock's height.
    expect((lastCard?.y ?? 0) + (lastCard?.height ?? 0)).toBeLessThanOrEqual(dockTop + 1);
  });

  test("the dock does not fly up when the on-screen keyboard opens", async ({ page }) => {
    // Android Chrome auto-scrolls the layout viewport to clear a focused input from the keyboard:
    // visualViewport shrinks AND gets a nonzero offsetTop at the same time. A fake visualViewport reproduces
    // that exact shape deterministically, without a real keyboard — the same technique the legacy
    // scripts/layout-gate.mjs already used for the unrelated Firefox-toolbar case (there, offsetTop stays 0).
    // Root cause this guards: useVisibleViewport's hiddenBottom math used to be computed the same way
    // regardless of offsetTop, so a keyboard-triggered scroll was mistaken for permanent browser chrome and
    // double-compensated — on top of `.shell`'s own (already-correct) height+transform, the sticky dock's
    // `bottom: var(--dock-bottom)` yanked it further up, away from the shell's true bottom edge (reported
    // live, 22 Nov 2026: "the dock flies to the top" the moment the keyboard shows up).
    await page.addInitScript(() => {
      const listeners: Record<string, Array<() => void>> = { resize: [], scroll: [] };
      const fake = {
        height: 544,
        width: 390,
        offsetTop: 120,
        addEventListener: (type: string, fn: () => void) => listeners[type]?.push(fn),
        removeEventListener: (type: string, fn: () => void) => {
          listeners[type] = (listeners[type] ?? []).filter((f) => f !== fn);
        },
      };
      Object.defineProperty(window, "visualViewport", { value: fake, configurable: true });
    });
    await signedIn(page);
    const box = await page.getByRole("navigation", { name: "Primary" }).boundingBox();
    expect(box).not.toBeNull();
    // Flush with the shell's own (translated + shrunk) bottom edge — offsetTop + height — not yanked further
    // up by hiddenBottom, which must stay 0 whenever offsetTop > 0.
    expect(Math.round((box?.y ?? 0) + (box?.height ?? 0))).toBe(120 + 544);
  });
});
