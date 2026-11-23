// e2e/gallery.spec.ts — the visual baseline the tracker kept listing as "nothing has taken the shots yet".
//
// `/__gallery` is the living contract for the primitives (buttons, chips, cards, type scale, colour roles, the three
// card families, empty/error/async states). It had real content and no evidence behind it: a design change could
// alter every screen's look with nothing to diff against.
//
// Hermetic like the rest of this suite — `vite preview` serves the built app and no API is touched, because the
// gallery renders from local fixtures. Playwright stores one baseline per engine, so Chromium/Firefox/WebKit
// differences are expectations rather than failures.
//
// Both viewports on purpose: the same primitives are drawn by the phone dock and the desktop sidebar, and a card
// that only breaks at 1440px is exactly the regression this is for.
import { expect, test } from "@playwright/test";

const VIEWPORTS = [
  { name: "phone", width: 390, height: 844 },
  { name: "desktop", width: 1440, height: 900 },
] as const;

/** The sections worth their own shot — a full-page image makes a one-pixel spacing change look like a rewrite. */
const SECTIONS = ["colors", "scale", "buttons", "media-grid-books", "media-grid-music", "states"] as const;

for (const viewport of VIEWPORTS) {
  // Tagged @visual and excluded from `npm run test:e2e` (see package.json): baselines are per-environment, and CI's
  // font stack is not this machine's, so running them there would fail on rendering rather than on a regression —
  // a gate that cries wolf gets ignored, which is worse than no gate. `npm run test:visual` runs them where the
  // baselines were taken; a real design change shows up there as a diff on the section that changed.
  test.describe(`gallery at ${viewport.name} @visual`, () => {
    test.use({ viewport: { width: viewport.width, height: viewport.height } });

    test.beforeEach(async ({ page }) => {
      await page.goto("/__gallery");
      // Animations and transitions make screenshots flaky without catching anything a human would see twice.
      await page.addStyleTag({
        content: "*,*::before,*::after{animation:none!important;transition:none!important;caret-color:transparent!important}",
      });
      await page.waitForSelector("[data-gallery-section]");
      // Covers are inline SVG data URLs (no network), but let layout settle before measuring anything.
      await page.waitForTimeout(250);
    });

    for (const section of SECTIONS) {
      test(`section: ${section}`, async ({ page }) => {
        const target = page.locator(`[data-gallery-section="${section}"]`);
        await expect(target).toBeVisible();
        await expect(target).toHaveScreenshot(`gallery-${viewport.name}-${section}.png`, {
          // Small anti-aliasing differences between runs are noise; a real spacing or colour change is not.
          maxDiffPixelRatio: 0.02,
        });
      });
    }

    test("whole page", async ({ page }) => {
      await expect(page).toHaveScreenshot(`gallery-${viewport.name}-full.png`, {
        fullPage: true,
        maxDiffPixelRatio: 0.02,
      });
    });
  });
}
