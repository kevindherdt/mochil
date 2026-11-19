import assert from "node:assert/strict";
import { mkdir } from "node:fs/promises";
import { chromium } from "playwright";
import { primaryNavigationVisible, readPrimaryNavigationGeometry } from "./browser-layout-contract.mjs";

const baseURL = process.env.MOCHIL_BASE_URL || "http://127.0.0.1:8099";
const username = process.env.OPDSHELF_USERNAME || "";
const password = process.env.OPDSHELF_PASSWORD || "";
const artifactDirectory = process.env.BROWSER_SMOKE_ARTIFACTS || "";
const executablePath = process.env.CHROMIUM_PATH || "/usr/bin/chromium";
const requireMediaFixtures = process.env.BROWSER_SMOKE_REQUIRE_MEDIA !== "false";
const viewports = [
  { name: "desktop", width: 1440, height: 900 },
  { name: "mobile", width: 390, height: 844 },
  { name: "compact", width: 320, height: 720 },
];

if (artifactDirectory) await mkdir(artifactDirectory, { recursive: true });

const browser = await chromium.launch({ executablePath, headless: true });
const results = [];

try {
  for (const theme of ["dark", "light"]) {
    for (const viewport of viewports) {
      results.push(await inspectSurface(browser, theme, viewport));
    }
  }
} finally {
  await browser.close();
}

for (const result of results) {
  assert.deepEqual(
    result.pageErrors,
    [],
    `${result.name} raised browser errors`,
  );
  assert.equal(
    result.overflow,
    0,
    `${result.name} overflows horizontally by ${result.overflow}px`,
  );
  if (result.viewport !== "desktop") {
    assert.deepEqual(
      result.smallTargets,
      [],
      `${result.name} has touch targets below 44px`,
    );
    assert.equal(
      result.mobileLibraryControlsContained,
      true,
      `${result.name} library controls escape the mobile shell`,
    );
    assert.equal(result.mobilePrimaryNavigationVisible, true, `${result.name} hides or displaces primary navigation in a library`);
    assert.equal(result.dashboardPrimaryNavigationVisible, true, `${result.name} hides or displaces primary navigation on Home`);
    if (result.dashboardAudioLayout.available) {
      assert.ok(result.dashboardAudioLayout.cardWidth >= (result.viewport === "compact" ? 280 : 320), `${result.name} squeezes Continue Listening into narrow cards`);
      assert.ok(result.dashboardAudioLayout.artworkWidth >= 60, `${result.name} collapses Continue Listening artwork`);
      assert.ok(result.dashboardAudioLayout.columns >= 2, `${result.name} stacks Continue Listening artwork above its copy`);
    }
  }
}

const mobileDark = results.find((result) => result.name === "mobile-dark");
const desktopDark = results.find((result) => result.name === "desktop-dark");
const desktopLight = results.find((result) => result.name === "desktop-light");
assert.equal(
  desktopDark.settingsIsolation,
  true,
  "adding a library reset an unrelated unsaved Settings field",
);
if (requireMediaFixtures) {
  assert.equal(
    desktopLight.videoPlayer.available,
    true,
    "the configured dashboard has no resumable video for the player lifecycle check",
  );
  assert.equal(
    mobileDark.audio.available,
    true,
    "the configured catalogue has no music track for the audio persistence check",
  );
}
if (mobileDark.audio.available) {
  assert.equal(mobileDark.audio.playerVisibleBefore, true, "opening a music track did not reveal the player");
  assert.equal(mobileDark.audio.playerVisibleAfter, true, "navigation removed the persistent audio player");
  assert.equal(mobileDark.audio.sourceAfter, mobileDark.audio.sourceBefore, "navigation changed the active audio source");
}

console.log(`Browser smoke passed: ${results.map((result) => result.name).join(", ")}; ${mobileDark.audio.available ? "audio persisted across navigation" : "media lifecycle checks skipped for the structural fixture"}.`);

async function inspectSurface(browserInstance, theme, viewport) {
  const context = await browserInstance.newContext({
    viewport: { width: viewport.width, height: viewport.height },
    colorScheme: theme,
  });
  await context.addInitScript(
    (selectedTheme) => localStorage.setItem("mochil-theme", selectedTheme),
    theme,
  );
  const page = await context.newPage();
  const pageErrors = [];
  page.on("pageerror", (error) => pageErrors.push(error.message));

  try {
    await page.goto(baseURL, { waitUntil: "networkidle" });
    await signIn(page);
    await verifyDestinationChevron(page);
    await capture(page, `${viewport.name}-${theme}-dashboard`);
    const dashboardPrimaryNavigationVisible = primaryNavigationVisible(
      await page.evaluate(readPrimaryNavigationGeometry),
    );
    const dashboardAudioLayout = await page.evaluate(() => {
      const card = document.querySelector("#audio-reading-track .audio-item");
      const open = card?.querySelector(".audio-open");
      const artwork = card?.querySelector(".audio-glyph");
      if (!card || !open || !artwork) return { available: false };
      return {
        available: true,
        cardWidth: card.getBoundingClientRect().width,
        artworkWidth: artwork.getBoundingClientRect().width,
        columns: getComputedStyle(open).gridTemplateColumns.split(" ").filter(Boolean).length,
      };
    });

    const videoPlayer =
      viewport.name === "desktop" && theme === "light"
        ? await verifyVideoPlayer(page)
        : { available: false };

    await openFirstLibrary(page);
    await page
      .locator("#filters option")
      .first()
      .waitFor({ state: "attached" });
    await page.waitForFunction(
      () =>
        !/opening|loading/i.test(
          document.querySelector("#status")?.textContent || "",
        ),
    );
    await page.waitForTimeout(800);
    await capture(page, `${viewport.name}-${theme}-library`);

    const destinationRail = await verifyResponsiveDestinationRail(page, viewport.width);
    const stickyContentShell = await verifyStickyContentShell(page);

    const genericView = page.locator("#view-menu");
    if (await genericView.isVisible()) {
      await genericView.locator('[data-layout="list"]').click();
      assert.equal(await genericView.locator('[data-layout="list"]').getAttribute("aria-pressed"), "true", "the layout toggle did not select list");
      await genericView.locator('[data-layout="covers"]').click();
      assert.equal(await genericView.locator('[data-layout="covers"]').getAttribute("aria-pressed"), "true", "the layout toggle did not return to covers");
    }

    if (viewport.name === "desktop" && theme === "dark") {
      await verifyPublicationDestinations(page);
      await captureMusicSurface(page, "desktop-dark-music");
      await verifyAudiobookCatalogue(page);
      await verifyPodcastCatalogue(page);
    }

    const vaultLayout =
      viewport.name === "desktop" && theme === "dark"
        ? await verifyVaultLayout(page)
        : { available: false };

    const measurements = await page.evaluate(() => {
      const root = document.documentElement;
      const controls = [
        ...document.querySelectorAll("button, select, summary, .masthead a"),
      ];
      const smallTargets = controls.flatMap((element) => {
        const style = getComputedStyle(element);
        const bounds = element.getBoundingClientRect();
        const closedDetails = element.closest("details:not([open])");
        const concealed =
          closedDetails &&
          element !== closedDetails.querySelector(":scope > summary");
        const visible =
          style.display !== "none" &&
          style.visibility !== "hidden" &&
          !concealed &&
          bounds.width > 0 &&
          bounds.height > 0 &&
          bounds.bottom >= 0 &&
          bounds.top <= innerHeight;
        if (!visible || (bounds.width >= 44 && bounds.height >= 44)) return [];
        return [
          {
            selector: element.id
              ? `#${element.id}`
              : `${element.tagName.toLowerCase()}.${String(element.className).trim().replaceAll(" ", ".")}`,
            width: Math.round(bounds.width),
            height: Math.round(bounds.height),
          },
        ];
      });
      return {
        overflow: Math.max(0, root.scrollWidth - innerWidth),
        smallTargets,
        mobileLibraryControlsContained: (() => {
          if (innerWidth > 760) return true;
          const shell = document.querySelector(".shell-context")?.getBoundingClientRect();
          const controls = document.querySelector(".browse-controls:not([hidden])")?.getBoundingClientRect();
          if (!shell || !controls) return true;
          return controls.left >= shell.left - 1 && controls.right <= shell.right + 1;
        })(),
      };
    });
    const libraryPrimaryNavigationVisible = primaryNavigationVisible(
      await page.evaluate(readPrimaryNavigationGeometry),
    );

    const settingsIsolation =
      viewport.name === "desktop" && theme === "dark"
        ? await verifySettingsIsolation(page)
        : false;
    const metadataReview =
      viewport.name === "desktop" && theme === "dark"
        ? await verifyMetadataReview(page)
        : { available: false };
    const audio =
      viewport.name === "mobile" && theme === "dark"
        ? await verifyAudioPersistence(page)
        : { available: false };

    return {
      name: `${viewport.name}-${theme}`,
      viewport: viewport.name,
      pageErrors,
      audio,
      videoPlayer,
      settingsIsolation,
      metadataReview,
      vaultLayout,
      destinationRail,
      stickyContentShell,
      dashboardAudioLayout,
      dashboardPrimaryNavigationVisible,
      mobilePrimaryNavigationVisible: libraryPrimaryNavigationVisible,
      ...measurements,
    };
  } finally {
    await context.close();
  }
}

async function verifyStickyContentShell(page) {
  const result = await page.evaluate(async () => {
    const masthead = document.querySelector('.masthead');
    const context = document.querySelector('.shell-context:not([hidden])');
    const content = document.querySelector('#book-catalog > *, #series-index > *, #audio-items > *, #video-index > *');
    if (!masthead || !context || !content || document.documentElement.scrollHeight <= innerHeight + 120) return { available: false };
    const before = {
      masthead: masthead.getBoundingClientRect().top,
      context: context.getBoundingClientRect().top,
      content: content.getBoundingClientRect().top,
    };
    scrollTo({ top: Math.min(640, document.documentElement.scrollHeight - innerHeight), behavior: 'instant' });
    await new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve)));
    const after = {
      masthead: masthead.getBoundingClientRect().top,
      context: context.getBoundingClientRect().top,
      content: content.getBoundingClientRect().top,
    };
    scrollTo({ top: 0, behavior: 'instant' });
    await new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve)));
    return { available: true, before, after };
  });
  if (!result.available) return result;
  assert.ok(Math.abs(result.after.masthead - result.before.masthead) <= 1, `the masthead moved with catalogue content: ${JSON.stringify(result)}`);
  assert.ok(Math.abs(result.after.context - result.before.context) <= 1, `library controls moved with catalogue content: ${JSON.stringify(result)}`);
  assert.ok(result.after.content < result.before.content - 80, `catalogue content did not scroll beneath the persistent shell: ${JSON.stringify(result)}`);
  return result;
}

async function verifyResponsiveDestinationRail(page, viewportWidth) {
  const rail = page.locator(".publication-destinations:not([hidden]), .music-destinations:not([hidden])").first();
  if (!(await rail.count())) return { available: false };
  const buttons = rail.locator("button");
  const count = await buttons.count();
  if (count < 2) return { available: false };
  const initial = await rail.evaluate((element) => ({
    clientWidth: element.clientWidth,
    scrollWidth: element.scrollWidth,
    activeCount: element.querySelectorAll("button[aria-current=page]").length,
    tabbableCount: [...element.querySelectorAll("button")].filter((button) => button.tabIndex === 0).length,
  }));
  assert.equal(initial.activeCount, 1, "destination rail does not expose exactly one active destination");
  assert.equal(initial.tabbableCount, 1, "destination rail does not expose one roving keyboard target");
  if (viewportWidth <= 760 && initial.scrollWidth > initial.clientWidth) {
    const last = buttons.last();
    await last.click();
    await page.waitForTimeout(100);
    const revealed = await last.evaluate((button) => {
      const item = button.getBoundingClientRect();
      const viewport = button.parentElement.getBoundingClientRect();
      return button.getAttribute("aria-current") === "page" && item.left >= viewport.left - 1 && item.right <= viewport.right + 1;
    });
    assert.equal(revealed, true, "compact destination rail did not reveal its newly active item");
  }
  return { available: true, ...initial };
}

async function verifyVaultLayout(page) {
  const shelf = page
    .locator("#filters option")
    .filter({ hasText: /wiki|notes|vault/i })
    .first();
  if (!(await shelf.count())) return { available: false };
  await page.locator("#filters").selectOption(await shelf.getAttribute("value"));
  await page.locator(".vault-workspace").waitFor();
  const firstPage = page.locator(".vault-page").first();
  if (!(await firstPage.count())) return { available: false };
  await firstPage.evaluate((button) => button.click());
  await page.locator(".vault-document article").waitFor();
  const result = await page.evaluate(() => {
    const pane = document.querySelector(".vault-pane");
    const tree = document.querySelector(".vault-tree");
    const toggle = document.querySelector("[data-vault-sidebar]");
    const switcher = document.querySelector(".library-tools");
    const article = pane?.querySelector("article");
    if (!pane || !tree || !toggle || !switcher || !article) return null;
    const marker = document.createElement("div");
    marker.style.height = "1800px";
    marker.setAttribute("aria-hidden", "true");
    article.append(marker);
    const before = {
      window: scrollY,
      toggle: toggle.getBoundingClientRect().top,
      tree: tree.getBoundingClientRect().top,
      switcher: switcher.getBoundingClientRect().top,
    };
    pane.scrollTop = 600;
    const after = {
      window: scrollY,
      toggle: toggle.getBoundingClientRect().top,
      tree: tree.getBoundingClientRect().top,
      switcher: switcher.getBoundingClientRect().top,
    };
    marker.remove();
    return {
      paneScrolled: pane.scrollTop > 0,
      shellStayedPut:
        before.window === after.window &&
        before.toggle === after.toggle &&
        before.tree === after.tree &&
        before.switcher === after.switcher,
    };
  });
  assert.equal(result?.paneScrolled, true, "the Vault article pane did not scroll");
  assert.equal(
    result?.shellStayedPut,
    true,
    "Vault navigation moved while the article pane scrolled",
  );
  await page.locator(".vault-sidebar-tools [data-vault-sidebar]").click();
  const reopen = page.locator(".vault-workspace > [data-vault-sidebar]");
  await reopen.waitFor({ state: "visible" });
  await reopen.click();
  await page.locator(".vault-sidebar-tools [data-vault-sidebar]").waitFor({ state: "visible" });

  const musicShelf = page
    .locator("#filters option")
    .filter({ hasText: /music/i })
    .first();
  if (await musicShelf.count()) {
    await page.locator("#filters").selectOption(await musicShelf.getAttribute("value"));
    await page.locator('#music-destinations [data-music-destination="favorites"]').click();
    await page.waitForFunction(() => !document.body.classList.contains("vault-reading"));
    assert.notEqual(
      await page.locator("body").evaluate((body) => getComputedStyle(body).overflowY),
      "hidden",
      "Music Favorites retained the Vault page-scroll lock",
    );
  }
  return { available: true, ...result };
}

async function signIn(page) {
  if (!new URL(page.url()).pathname.startsWith("/login")) return;
  assert.ok(
    username && password,
    "the server requires login; set OPDSHELF_USERNAME and OPDSHELF_PASSWORD",
  );
  await page.locator('input[name="username"]').fill(username);
  await page.locator('input[name="password"]').fill(password);
  await Promise.all([
    page.waitForURL((url) => !url.pathname.startsWith("/login")),
    page.locator('button[type="submit"]').click(),
  ]);
  await page.waitForLoadState("networkidle");
}

async function verifyVideoPlayer(page) {
  let item = page.locator("#video-reading-track .video-reading-item").first();
  let openedLibrary = false;
  if (!(await item.count())) {
    await openFirstLibrary(page);
    openedLibrary = true;
    const shelf = page
      .locator("#filters option")
      .filter({ hasText: /movie|anime|tv/i })
      .first();
    if (!(await shelf.count())) return { available: false };
    await page
      .locator("#filters")
      .selectOption(await shelf.getAttribute("value"));
    await page.locator("#video-index:not([hidden])").waitFor();
    await page.waitForFunction(
      () =>
        !/opening|loading/i.test(
          document.querySelector("#status")?.textContent || "",
        ),
    );
    item = page.locator("#videos .video-open[data-item]").first();
    if (!(await item.count())) {
      const show = page.locator("#videos .video-show").first();
      if (await show.count()) {
        await show.locator(":scope > summary").click();
        const season = show.locator(".video-season").first();
        await season.locator(":scope > summary").click();
        item = season.locator(".video-open[data-item]").first();
        await item.waitFor();
      }
    }
    if (!(await item.count())) return { available: false };
  }
  await item.click();
  const player = page.locator("#video-player");
  await player.locator("#video-element").waitFor({ state: "visible" });
  assert.equal(await player.evaluate((element) => element.open), true);
  await page.locator("#video-close").click();
  await page.waitForFunction(
    () => !document.querySelector("#video-player")?.open,
  );
  if (openedLibrary)
    await page.locator("#filters").selectOption("__home__");
  return { available: true };
}

async function verifySettingsIsolation(page) {
  await page.getByRole("button", { name: "Settings" }).click();
  await page
    .locator('[data-settings-panel="libraries"]:not([hidden])')
    .waitFor();
  const providers = page.locator("[data-metadata-provider-settings]");
  if (!(await providers.evaluate((element) => element.open)))
    await providers.locator(":scope > summary").click();
  const token = page.locator("#tmdb-token");
  await token.fill("unsaved-browser-smoke-token");
  await page.locator('[data-settings-tab="libraries"]').click();
  const serverLibraries = page.locator(".server-library-settings");
  if (!(await serverLibraries.evaluate((element) => element.open)))
    await serverLibraries.locator(":scope > summary").click();
  await page.locator("#add-library").click();
  assert.equal(await token.inputValue(), "unsaved-browser-smoke-token");
  return true;
}

async function verifyMetadataReview(page) {
  // The metadata review now lives inside the Libraries tab.
  await page.locator('[data-settings-tab="libraries"]').click();
  await page.locator('[data-settings-panel="metadata"]:not([hidden])').waitFor();
  const selector = page.locator("#metadata-library");
  const lightNovels = selector.locator("option").filter({ hasText: /light.?novel/i }).first();
  if (await lightNovels.count()) {
    await selector.selectOption(await lightNovels.getAttribute("value"));
  }
  await page.waitForTimeout(250);
  const history = page.locator(".metadata-history");
  const saved = page.locator("#metadata-jobs [data-job]").first();
  let publicationAvailable = false;
  let groupCount = 0;
  if (await history.isVisible() && await saved.count()) {
    if (!(await history.evaluate((node) => node.open))) await history.locator(":scope > summary").click();
    await saved.click();
    await page.waitForFunction(() => /inspected/i.test(document.querySelector("#metadata-status")?.textContent || ""));
    await page.waitForTimeout(250);
    const groups = page.locator(".metadata-duplicate-group");
    groupCount = await groups.count();
    if (groupCount) {
      assert.match(await groups.first().locator(":scope > details > summary").textContent(), /share reading position/i);
      assert.equal(await groups.first().locator(":scope > details").evaluate((node) => node.open), false);
    }
    await capture(page, "desktop-dark-metadata-review");
    publicationAvailable = true;
  }
  const music = selector.locator("option").filter({ hasText: /^Music$/i }).first();
  let musicQuality = false;
  if (await music.count()) {
    await selector.selectOption(await music.getAttribute("value"));
    await page.getByRole("button", { name: "Check music quality" }).click();
    await page.waitForFunction(() => /tracks checked/i.test(document.querySelector("#metadata-status")?.textContent || ""));
    const summary = await page.locator("#metadata-summary").textContent();
    const status = await page.locator("#metadata-status").textContent();
    assert.match(summary, /Album folders/i, "music quality still presents tag-derived values as an album count");
    assert.doesNotMatch(summary, /No folder sidecar/i, "music quality exposed the retired sidecar-only finding");
    assert.match(status, /embedded-artwork cache/i, "music quality does not explain its bounded artwork evidence");
    assert.equal(await page.locator("#metadata-batch").count(), 0, "publication correction actions leaked into music quality");
    await capture(page, "desktop-dark-music-quality");
    musicQuality = true;
  }
  return { available: publicationAvailable || musicQuality, publicationAvailable, groupCount, musicQuality };
}

async function openFirstLibrary(page) {
  const selector = page.locator("#filters");
  const libraries = selector.locator('option:not([value="__home__"]):not([value=""])');
  await libraries.first().waitFor({ state: "attached" });
  const value = await selector
    .locator('option:not([value="__home__"]):not([value=""])')
    .first()
    .getAttribute("value");
  assert.ok(value, "the destination selector has no library");
  await selector.selectOption(value);
}

async function verifyPublicationDestinations(page) {
  const selector = page.locator('#filters');
  const values = await selector.locator('option:not([value="__home__"]):not([value=""])').evaluateAll(options => options.map(option => option.value));
  for (const value of values) {
    await selector.selectOption(value);
    if (await page.locator('#publication-destinations').isVisible()) break;
  }
  const rail = page.locator('#publication-destinations');
  if (!(await rail.isVisible())) return { available: false };
  assert.deepEqual(await rail.locator('button:visible').allTextContents(), ['Series', 'Authors', 'All']);
  for (const destination of ['authors', 'books', 'series']) {
    await rail.locator(`[data-publication-destination="${destination}"]`).click();
    await page.waitForFunction(() => !/opening|loading/i.test(document.querySelector('#status')?.textContent || ''));
    assert.equal(await rail.locator(`[data-publication-destination="${destination}"]`).getAttribute('aria-current'), 'page');
    if (destination === 'authors') {
      const authors = page.locator('#series-index [data-bucket]');
      if (await authors.count() > 1) {
        await authors.last().scrollIntoViewIfNeeded();
        await page.waitForTimeout(350);
        const authorScroll = await page.evaluate(() => {
          const top = Number.parseFloat(document.documentElement.style.getPropertyValue('--catalog-sticky-top')) || 0;
          const visible = [...document.querySelectorAll('#series-index .author-card[data-bucket], #series-index .author-row[data-bucket]')]
            .find(item => item.getBoundingClientRect().bottom > top + 8);
          return { expected: visible?.dataset.bucket || '', active: document.querySelector('#alphabet .letter.active')?.dataset.bucket || '', scrollY, top };
        });
        if (authorScroll.scrollY > 18) assert.equal(authorScroll.active, authorScroll.expected, `Author scroll did not update its alphabet bucket: ${JSON.stringify(authorScroll)}`);
      }
    }
  }
  // Collections only appears once the library has some.
  const collections = rail.locator('[data-publication-destination="collections"]');
  if (await collections.isVisible()) {
    await collections.click();
    await page.waitForFunction(() => !/opening|loading/i.test(document.querySelector('#status')?.textContent || ''));
    assert.equal(await page.locator('#view-menu').isVisible(), true, 'Collections removed the shared view control');
  }
  return { available: true };
}

async function verifyDestinationChevron(page) {
  // The themed menu replaced the native select; its button is the control.
  const button = page.locator(".library-menu-button");
  await button.click();
  assert.equal(await button.getAttribute("aria-expanded"), "true", "the library menu button does not open the list");
  assert((await page.locator(".library-menu-list li").count()) > 0, "the library menu opened empty");
  await page.keyboard.press("Escape");
}

async function verifyAudioPersistence(page) {
  const shelf = page
    .locator("#filters option")
    .filter({ hasText: /music/i })
    .first();
  if (!(await shelf.count())) return { available: false };
  await page
    .locator("#filters")
    .selectOption(await shelf.getAttribute("value"));
  await page.locator("#music-destinations:not([hidden])").waitFor();
  await page.waitForFunction(
    () => !/opening|loading/i.test(document.querySelector("#status")?.textContent || ""),
  );
  const compactMusicHeader = await page.evaluate(() => {
    const destinations = document.querySelector("#music-destinations").getBoundingClientRect();
    const actions = document.querySelector(".audio-browse-head .audio-toolbar").getBoundingClientRect();
    return {
      sameRow: Math.abs((destinations.top + destinations.height / 2) - (actions.top + actions.height / 2)) <= 1,
      separated: destinations.right <= actions.left + 1,
      actionHeight: actions.height,
      destinationsTop: destinations.top,
      actionsTop: actions.top,
      destinationsHeight: destinations.height,
      destinationsWidth: destinations.width,
    };
  });
  assert.equal(compactMusicHeader.sameRow, true, `mobile music navigation and actions did not share one row: ${JSON.stringify(compactMusicHeader)}`);
  assert.equal(compactMusicHeader.separated, true, "mobile music navigation collided with its actions");
  assert.ok(compactMusicHeader.actionHeight >= 44, "mobile music actions lost their touch target height");
  assert.ok(compactMusicHeader.destinationsWidth >= 240, "mobile music destinations were squeezed out of view");
  const firstTrack = page.locator("#audio-items .audio-open").first();
  await firstTrack.waitFor();
  await capture(page, "mobile-dark-music");
  const sourceArtwork = firstTrack.locator(".audio-glyph img");
  const sourceHasArtwork = await sourceArtwork.count() > 0 && await sourceArtwork.evaluate(image => image.complete && image.naturalWidth > 0);
  await firstTrack.click();
  await page.locator("#audio-player:not([hidden])").waitFor();
  await verifyExpandedAudioPlayer(page, "mobile-dark-expanded-player", sourceHasArtwork);
  const before = await audioState(page);
  await page.getByRole("button", { name: "Settings" }).click();
  const after = await audioState(page);
  await capture(page, "mobile-dark-audio-settings");
  await page.locator('.audio-player-more > summary').click();
  await page.locator('#audio-close').click();
  await page.locator('#audio-player[hidden]').waitFor({ state: 'attached' });
  return {
    available: true,
    playerVisibleBefore: before.visible,
    playerVisibleAfter: after.visible,
    sourceBefore: before.source,
    sourceAfter: after.source,
  };
}

async function verifyExpandedAudioPlayer(page, captureName, expectArtwork = false) {
  await page.locator("#audio-expand").click();
  const player = page.locator("#audio-player.expanded");
  await player.waitFor();
  if (expectArtwork) {
    await page.waitForFunction(() => {
      const image = document.querySelector("#audio-player.expanded #audio-player-art img");
      return image?.complete && image.naturalWidth > 0;
    }, { timeout: 5000 });
  }
  const geometry = await player.evaluate(element => {
    const bounds = element.getBoundingClientRect();
    const copy = element.querySelector(".audio-player-copy").getBoundingClientRect();
    const artwork = element.querySelector(".audio-art-play").getBoundingClientRect();
    const transport = element.querySelector(".audio-expanded-transport").getBoundingClientRect();
    const labels = ["#audio-title", "#audio-context", "#audio-source-facts"]
      .map(selector => element.querySelector(selector))
      .filter(label => label.textContent.trim() && getComputedStyle(label).display !== "none")
      .map(label => ({ selector: `#${label.id}`, ...label.getBoundingClientRect().toJSON() }));
    return {
      viewport: { width: innerWidth, height: innerHeight },
      bounds: bounds.toJSON(),
      copy: copy.toJSON(),
      artwork: artwork.toJSON(),
      transport: transport.toJSON(),
      labels,
      labelOverlap: labels.slice(1).some((label, index) => labels[index].bottom > label.top + 1),
    };
  });
  assert.ok(geometry.bounds.left >= -1 && geometry.bounds.right <= geometry.viewport.width + 1, `expanded audio player escaped horizontally: ${JSON.stringify(geometry)}`);
  assert.ok(geometry.bounds.top >= -1 && geometry.bounds.bottom <= geometry.viewport.height + 1, `expanded audio player escaped vertically: ${JSON.stringify(geometry)}`);
  assert.equal(geometry.labelOverlap, false, `expanded audio title, context, and source facts overlap: ${JSON.stringify(geometry)}`);
  assert.ok(geometry.labels.every(label => label.top >= geometry.copy.top - 1 && label.bottom <= geometry.copy.bottom + 1), `expanded audio labels escaped their intrinsic content area: ${JSON.stringify(geometry)}`);
  assert.ok(geometry.artwork.top >= geometry.copy.bottom - 1, `expanded audio artwork collided with metadata: ${JSON.stringify(geometry)}`);
  assert.ok(geometry.transport.width > 0 && geometry.transport.height >= 44, `expanded audio transport is not reachable: ${JSON.stringify(geometry)}`);
  await capture(page, captureName);
  const queueToggle = page.locator("#audio-queue-toggle:not([hidden])");
  if (await queueToggle.count()) {
    await queueToggle.click();
    const queue = page.locator("#audio-player-queue:not([hidden])");
    await queue.waitFor();
    const rows = queue.locator("#audio-queue-items > button");
    assert.ok(await rows.count() > 0 && await rows.count() <= 60, "expanded player queue is empty or unbounded");
    assert.equal(await queue.locator('#audio-queue-items > button[aria-current="true"]').count(), 1, "expanded player queue did not identify the active track");
    await capture(page, `${captureName}-queue`);
    await page.keyboard.press("Escape");
    await page.locator("#audio-player-queue[hidden]").waitFor({ state: "attached" });
    assert.equal(await queueToggle.getAttribute("aria-expanded"), "false", "Escape left the queue toggle expanded");
  }
  await page.locator("#audio-collapse").click();
  await page.locator("#audio-player:not(.expanded)").waitFor();
}

async function captureMusicSurface(page, name) {
  const shelf = page
    .locator("#filters option")
    .filter({ hasText: /music/i })
    .first();
  if (!(await shelf.count())) return;
  await page
    .locator("#filters")
    .selectOption(await shelf.getAttribute("value"));
  await page.locator("#music-destinations:not([hidden])").waitFor();
  await page.waitForFunction(
    () => !/opening|loading/i.test(document.querySelector("#status")?.textContent || ""),
  );
  const recentlyPlayed = page.locator("#audio-items .music-discovery-section").filter({
    has: page.getByRole("heading", { name: "Recently played", exact: true }),
  });
  await recentlyPlayed.waitFor();
  const discoveryColumns = await recentlyPlayed.locator(".music-discovery-tracks").evaluate(
    (element) => getComputedStyle(element).gridTemplateColumns.split(" ").filter(Boolean).length,
  );
  assert.ok(
    discoveryColumns >= 4,
    `wide Music Discover did not use the desktop canvas (${discoveryColumns} track columns)`,
  );
  // Catalogue data can settle before artwork has decoded. Wait for every
  // currently visible discovery image to either load or reach its intentional
  // fallback state so visual reviews capture the finished surface.
  await page.waitForFunction(() => {
    const images = [...document.querySelectorAll("#audio-items .music-discovery-section img")]
      .filter(image => {
        const bounds = image.getBoundingClientRect();
        return bounds.width > 0 && bounds.height > 0;
      });
    // Failed artwork nodes are removed by the loader, leaving the card's
    // built-in fallback visible; surviving nodes must have decoded content.
    return images.length === 0 || images.every(image =>
      image.complete && image.naturalWidth > 0
    );
  }, { timeout: 5000 }).catch(() => {});
  const destinationState = await page.locator("#music-destinations > button").evaluateAll(buttons => ({
    active: buttons.filter(button => button.classList.contains("active")).map(button => button.textContent.trim()),
    current: buttons.filter(button => button.getAttribute("aria-current") === "page").map(button => button.textContent.trim()),
    visible: buttons.filter(button => {
      const style = getComputedStyle(button);
      const bounds = button.getBoundingClientRect();
      return style.display !== "none" && style.visibility !== "hidden" && Number(style.opacity) > 0 && bounds.width > 0 && bounds.height > 0;
    }).map(button => button.textContent.trim()),
  }));
  assert.deepEqual(destinationState, {
    active: ["Discover"],
    current: ["Discover"],
    visible: ["Discover", "Recent", "Albums", "Artists", "Favorites", "Playlists", "Tracks", "Queue"],
  }, "music destinations were missing or selection was not exclusive");
  const musicHeaderGeometry = await page.locator(".audio-browse-head").evaluate(header => {
    const rail = header.querySelector("#music-destinations").getBoundingClientRect();
    const actions = header.querySelector(".audio-toolbar").getBoundingClientRect();
    const buttons = [...header.querySelectorAll("#music-destinations > button")].map(button => button.getBoundingClientRect());
    return {
      overlap: Math.max(0, rail.right - actions.left),
      minimumButtonGap: buttons.slice(1).reduce((smallest, bounds, index) => Math.min(smallest, bounds.left - buttons[index].right), Infinity),
    };
  });
  assert.equal(musicHeaderGeometry.overlap, 0, `music destinations overlap the View action: ${JSON.stringify(musicHeaderGeometry)}`);
  assert.ok(musicHeaderGeometry.minimumButtonGap >= 5, `music destination labels run together: ${JSON.stringify(musicHeaderGeometry)}`);
  const passiveDestination = page.locator('#music-destinations [data-music-destination="playlists"]');
  await passiveDestination.hover();
  assert.equal(
    await passiveDestination.evaluate(button => getComputedStyle(button).backgroundColor),
    "rgba(0, 0, 0, 0)",
    "a hovered music destination looked like a second selected tab",
  );
  await page.mouse.move(0, 0);
  assert.equal(
    await recentlyPlayed.locator(".audio-item i").count(),
    0,
    "Discover rendered per-track progress bars for recently played music",
  );
  const incompleteArtwork = await recentlyPlayed.locator(".audio-glyph").evaluateAll(glyphs => glyphs.filter(glyph => {
    const image = glyph.querySelector("img");
    const fallback = glyph.querySelector("b");
    const hasArtwork = image?.complete && image.naturalWidth > 0;
    const hasFallback = Boolean(fallback?.textContent?.trim());
    return !hasArtwork && !hasFallback;
  }).length);
  assert.equal(incompleteArtwork, 0, "music rows with missing artwork lost their typographic fallback");
  await capture(page, name);
  const featuredAlbum = page.locator('#audio-items [data-discovery-group][data-discovery-type="albums"]').first();
  if (await featuredAlbum.count()) {
    const selectedTitle = (await featuredAlbum.locator("strong").textContent())?.trim();
    await featuredAlbum.click();
    const collectionHeading = page.locator("#audio-items .music-group-heading.featured h2");
    await collectionHeading.waitFor();
    assert.equal(
      (await collectionHeading.textContent())?.trim(),
      selectedTitle,
      "a featured Discover album did not open the selected collection",
    );
    assert.ok(
      (await page.locator("#audio-items .audio-item").count()) > 0,
      "a featured Discover album did not reveal its tracks",
    );
    await page.locator('#music-destinations [data-music-destination="discover"]').click();
    await page.locator("#audio-items .music-discovery").waitFor();
  }
  await page.locator('#music-destinations [data-music-destination="recently-played"]').click();
  await page.locator("#audio-items .music-discovery").waitFor({ state: "detached" });
  await page.waitForFunction(
    () => !/opening|loading/i.test(document.querySelector("#status")?.textContent || ""),
  );
  assert.equal(
    await page.locator("#media-alphabet").isHidden(),
    true,
    "Recently played exposed an alphabetical rail for chronologically ordered music",
  );
  if (await page.locator("#audio-items .audio-item").count()) {
    await page.locator(".audio-view-menu > summary").click();
    await page.locator("#music-layout").selectOption("covers");
    await page.locator("#audio-items .music-track-grid").waitFor();
    await page.locator(".audio-view-menu > summary").click();
    await page.locator("#music-layout").selectOption("list");
    await page.locator("#audio-items > .audio-item").first().waitFor();
  }
  await page.locator('#music-destinations [data-music-destination="tracks"]').click();
  await page.locator("#audio-items .audio-item").first().waitFor();
  await page.locator(".audio-view-menu > summary").click();
  await page.locator("#music-layout").selectOption("covers");
  await page.locator("#audio-items.music-track-grid > .music-track-card").first().waitFor();
  await page.locator(".audio-view-menu > summary").click();
  await page.locator("#music-layout").selectOption("list");
  await page.locator("#audio-items:not(.music-track-grid) > .audio-item").first().waitFor();
  await page.locator('#music-destinations [data-music-destination="albums"]').click();
  const firstAlbum = page.locator("#audio-items .music-group-card").first();
  await firstAlbum.waitFor();
  const albumAlphabetState = async () => page.evaluate(() => {
    const top = Number.parseFloat(getComputedStyle(document.documentElement).getPropertyValue('--catalog-sticky-top')) || 0;
    const bucket = value => {
      const normalized = String(value || '').trim().replace(/^(the|an|a)\s+/i, '');
      const first = normalized.charAt(0).toUpperCase();
      return first >= 'A' && first <= 'Z' ? first : '#';
    };
    const cards = [...document.querySelectorAll('#audio-items .music-group-card')];
    const visible = cards
      .find(card => card.getBoundingClientRect().bottom > top + 8);
    return {
      expected: bucket(visible?.querySelector('strong')?.textContent),
      active: document.querySelector('#media-alphabet .letter.active')?.dataset.bucket || '',
      scrollY: window.scrollY,
      viewportHeight: window.innerHeight,
      documentHeight: document.documentElement.scrollHeight,
      top,
      firstBottom: cards[0]?.getBoundingClientRect().bottom ?? null,
      lastBottom: cards.at(-1)?.getBoundingClientRect().bottom ?? null,
    };
  });
  await page.locator('#media-alphabet:not([hidden])').waitFor();
  await page.waitForTimeout(100);
  const initialAlbumAlphabetState = await albumAlphabetState();
  assert.equal(
    initialAlbumAlphabetState.active,
    initialAlbumAlphabetState.expected,
    `music albums did not immediately select the visible alphabet bucket: ${JSON.stringify(initialAlbumAlphabetState)}`,
  );
  const activeAlbumLetter = page.locator('#media-alphabet .letter.active');
  const activeAlbumBucket = await activeAlbumLetter.getAttribute('data-bucket');
  await activeAlbumLetter.click();
  assert.equal(
    await page.locator('#media-alphabet .letter.active').getAttribute('data-bucket'),
    activeAlbumBucket,
    'clicking the active music jump letter cleared its selection',
  );
  const albumDestinationState = await page.locator("#music-destinations > button").evaluateAll(buttons => {
    const railBounds = buttons[0]?.parentElement.getBoundingClientRect();
    return buttons.map(button => {
      const style = getComputedStyle(button);
      const bounds = button.getBoundingClientRect();
      return {
        label: button.textContent.trim(),
        visible: style.display !== "none" && style.visibility !== "hidden" && Number(style.opacity) > 0 && bounds.width > 0 && bounds.height > 0,
        insideRail: Boolean(railBounds && bounds.left >= railBounds.left - 1 && bounds.right <= railBounds.right + 1),
        color: style.color,
      };
    });
  });
  assert.equal(albumDestinationState.length, 8, `music destinations changed in Albums: ${JSON.stringify(albumDestinationState)}`);
  assert.ok(albumDestinationState.every(destination => destination.visible), `music destinations became invisible in Albums: ${JSON.stringify(albumDestinationState)}`);
  assert.ok(albumDestinationState.every(destination => destination.insideRail), `wide Albums navigation clipped destinations despite available desktop space: ${JSON.stringify(albumDestinationState)}`);
  await capture(page, `${name}-albums`);
  await page.locator(".audio-view-menu > summary").click();
  await page.locator(".audio-view-menu > div").waitFor({ state: "visible" });
  await page.locator("#music-layout").selectOption("list");
  await page.locator("#audio-items .music-group-grid.list").waitFor();
  const listAlphabetState = await albumAlphabetState();
  assert.equal(listAlphabetState.active, listAlphabetState.expected, `music layout rerender lost its alphabet bucket: ${JSON.stringify(listAlphabetState)}`);
  const wideListColumns = await page
    .locator("#audio-items .music-group-grid.list")
    .evaluate((element) => getComputedStyle(element).gridTemplateColumns.split(" ").filter(Boolean).length);
  assert.ok(wideListColumns >= 2, "wide music collection lists did not use the available desktop width");
  await page.locator(".audio-view-menu > summary").click();
  await page.locator("#music-layout").selectOption("covers");
  await page.locator("#audio-items .music-group-grid:not(.list)").waitFor();
  await page.locator(".audio-view-menu > summary").click();
  await firstAlbum.click();
  await page.locator("#audio-items .music-group-heading").waitFor();
  assert.equal(
    (await page.locator("#status").textContent())?.trim(),
    "",
    "album detail repeated its identity above the owned detail heading",
  );
  assert.ok(
    (await page.locator("#audio-items .audio-item").count()) > 0,
    "opening a music album did not reveal its tracks",
  );
  await page.locator("#audio-items [data-play-music-group]").waitFor();
  await page.locator("#audio-items [data-shuffle-music-group]").waitFor();
  await page.locator('#audio-items a[href^="/api/audio-group-download"]').waitFor();
  const focusedTitle = await page.locator("#audio-items .music-group-heading.featured h2").evaluate((element) => {
    const style = getComputedStyle(element);
    const text = element.firstChild;
    const finalCharacter = document.createRange();
    finalCharacter.setStart(text, Math.max(0, text.length - 1));
    finalCharacter.setEnd(text, text.length);
    const characterBounds = finalCharacter.getBoundingClientRect();
    const headingBounds = element.closest(".music-group-heading").getBoundingClientRect();
    return {
      overflow: style.overflow,
      textOverflow: style.textOverflow,
      whiteSpace: style.whiteSpace,
      lineClamp: style.webkitLineClamp,
      finalCharacterVisible: characterBounds.bottom <= headingBounds.bottom + 1,
    };
  });
  assert.deepEqual(
    focusedTitle,
    { overflow: "visible", textOverflow: "clip", whiteSpace: "normal", lineClamp: "none", finalCharacterVisible: true },
    "focused album titles are visually truncated",
  );
  const focusedArtwork = page.locator("#audio-items .music-group-heading-art img");
  if (await focusedArtwork.count()) {
    await focusedArtwork.waitFor({ state: "visible" });
    await page.waitForFunction(
      (image) => image.classList.contains("loaded") && image.naturalWidth > 0,
      await focusedArtwork.elementHandle(),
    );
    assert.ok(
      (await focusedArtwork.evaluate((image) => image.naturalWidth)) > 0,
      "focused album artwork remained on its fallback",
    );
  }
  assert.equal(
    await page.locator("#audio-play-all").isHidden(),
    true,
    "album detail duplicated its primary Play action in the sticky toolbar",
  );
  assert.equal(
    await page.locator("#audio-shuffle").isHidden(),
    true,
    "album detail duplicated its Shuffle action in the sticky toolbar",
  );
  await capture(page, `${name}-album-tracks`);
  const albumTrack = page.locator("#audio-items .audio-open").first();
  const albumTrackArtwork = albumTrack.locator(".audio-glyph img");
  const albumTrackHasArtwork = await albumTrackArtwork.count() > 0 && await albumTrackArtwork.evaluate(image => image.complete && image.naturalWidth > 0);
  await albumTrack.click();
  await page.locator("#audio-player:not([hidden])").waitFor();
  await verifyExpandedAudioPlayer(page, `${name}-expanded-player`, albumTrackHasArtwork);
  await page.locator(".audio-player-more > summary").click();
  await page.locator("#audio-close").click();
  await page.locator("#audio-player[hidden]").waitFor({ state: "attached" });
  const firstTrackOptions = page.locator("#audio-items .audio-item .audio-more").first();
  await firstTrackOptions.locator("summary").click();
  await firstTrackOptions.locator('[data-action="track-info"]').click();
  const trackInfo = page.locator("dialog.music-track-info");
  await trackInfo.waitFor();
  assert.equal(await trackInfo.getAttribute("open"), "", "track information did not open as a modal");
  assert.ok((await trackInfo.locator("dl > div").count()) > 0, "track information omitted all known metadata");
  await capture(page, `${name}-track-info`);
  await trackInfo.locator('button').click();
  await page.locator("#audio-items [data-music-group-back]").click();
  await page.locator("#audio-items .music-group-card").first().waitFor();
  await page.locator('#music-destinations [data-music-destination="artists"]').click();
  const firstArtist = page.locator("#audio-items .music-group-card").first();
  await firstArtist.waitFor();
  await firstArtist.click();
  await page.locator("#audio-items .music-group-heading").waitFor();
  const firstArtistAlbum = page.locator("#audio-items .music-group-card").first();
  await firstArtistAlbum.waitFor();
  await firstArtistAlbum.click();
  await page.locator("#audio-items .audio-item").first().waitFor();
  assert.equal(
    await page.locator("#media-alphabet").isHidden(),
    true,
    "the global alphabet remained visible inside an album",
  );
  await page.locator('#music-destinations [data-music-destination="playlists"]').click();
  const firstPlaylist = page.locator("#audio-items [data-open-playlist]").first();
  if (await firstPlaylist.count()) {
    await firstPlaylist.click();
    await page.locator("#audio-items .music-playlist-detail").waitFor();
    const playlistTitleStyle = await page.locator("#audio-items .music-playlist-heading h2").evaluate((element) => {
      const style = getComputedStyle(element);
      return { overflow: style.overflow, textOverflow: style.textOverflow, whiteSpace: style.whiteSpace };
    });
    assert.deepEqual(
      playlistTitleStyle,
      { overflow: "visible", textOverflow: "clip", whiteSpace: "normal" },
      "focused playlist title is visually truncated",
    );
    const playlistArtwork = page.locator("#audio-items .music-playlist-heading > .music-playlist-art img");
    if (await playlistArtwork.count()) {
      await page.waitForFunction(
        (image) => image.naturalWidth > 0,
        await playlistArtwork.elementHandle(),
      );
    }
    await capture(page, `${name}-music-playlist`);
    const dragState = await page.locator("#audio-items .music-playlist-tracks").evaluate(element => ({
      finePointer: matchMedia("(pointer: fine)").matches,
      rows: [...element.querySelectorAll(":scope > .audio-item")].map(row => row.draggable),
    }));
    assert.ok(
      dragState.rows.every(value => value === dragState.finePointer),
      "playlist drag ordering was not scoped to fine-pointer browsers",
    );
  }
  await page.locator('#music-destinations [data-music-destination="queue"]').click();
  const queue = page.locator("#audio-items .queue-group");
  if (await queue.count()) {
    const dragState = await queue.evaluate(element => ({
      finePointer: matchMedia("(pointer: fine)").matches,
      rows: [...element.querySelectorAll(":scope > .audio-item")].map(row => row.draggable),
    }));
    assert.ok(
      dragState.rows.every(value => value === dragState.finePointer),
      "queue drag ordering was not scoped to fine-pointer browsers",
    );
  }
  await page.locator('#music-destinations [data-music-destination="discover"]').click();
}

async function verifyAudiobookCatalogue(page) {
  const shelf = page.locator("#filters option").filter({ hasText: /audiobook/i }).first();
  if (!(await shelf.count())) return;
  await page.locator("#filters").selectOption(await shelf.getAttribute("value"));
  await page.waitForFunction(
    () => !/opening|loading/i.test(document.querySelector("#status")?.textContent || ""),
  );
  const navigation = await page.evaluate(() => {
    const rail = document.querySelector("#audiobook-destinations")?.getBoundingClientRect();
    const actions = document.querySelector('.audio-browse-head[data-kind="audiobooks"] .audio-toolbar')?.getBoundingClientRect();
    if (!rail || !actions) return null;
    return {
      sameRow: Math.abs((rail.top + rail.height / 2) - (actions.top + actions.height / 2)) <= 1,
      separated: rail.right <= actions.left + 1,
      railWidth: rail.width,
      railHeight: rail.height,
      actionsHeight: actions.height,
    };
  });
  assert.ok(navigation, "audiobook destination navigation was not rendered");
  assert.equal(navigation.sameRow, true, `audiobook navigation and actions did not share one row: ${JSON.stringify(navigation)}`);
  assert.equal(navigation.separated, true, `audiobook navigation collided with its actions: ${JSON.stringify(navigation)}`);
  assert.ok(navigation.railWidth >= 240 && navigation.railHeight >= 44, `audiobook navigation was squeezed: ${JSON.stringify(navigation)}`);
  assert.ok(navigation.actionsHeight >= 44, `audiobook actions lost their target height: ${JSON.stringify(navigation)}`);
  await capture(page, "desktop-dark-audiobooks");
  const firstSeries = page.locator("#audio-items [data-audiobook-series]").first();
  if (!(await firstSeries.count())) return;
  await firstSeries.click();
  await page.locator("#audio-items .audiobook-focus").waitFor();
  assert.ok(
    (await page.locator("#audio-items .audiobook-focus .audio-item, #audio-items .audiobook-focus .audio-card").count()) > 0,
    "opening an audiobook series did not reveal its books",
  );
  await page.locator("#audio-items .audiobook-back").click();
  await page.locator("#audio-items [data-audiobook-series]").first().waitFor();
}

async function verifyPodcastCatalogue(page) {
  const shelf = page.locator("#filters option").filter({ hasText: /podcast/i }).first();
  if (!(await shelf.count())) return;
  await page.locator("#filters").selectOption(await shelf.getAttribute("value"));
  await page.waitForFunction(
    () => !/opening|loading/i.test(document.querySelector("#status")?.textContent || ""),
  );
  const firstShow = page.locator("#audio-items [data-podcast]").first();
  if (!(await firstShow.count())) return;
  await firstShow.click();
  await page.locator("#audio-items .podcast-focus").waitFor();
  assert.ok(
    (await page.locator("#audio-items .podcast-focus .audio-item").count()) > 0,
    "opening a podcast did not reveal its episodes",
  );
  await page.locator("#audio-items .podcast-back").click();
  await page.locator("#audio-items [data-podcast]").first().waitFor();
}

function audioState(page) {
  return page.evaluate(() => {
    const player = document.querySelector("#audio-player");
    const active = [...player.querySelectorAll("audio")].find(
      (element) => element.currentSrc || element.src,
    );
    return {
      visible: !player.hidden,
      source: active?.currentSrc || active?.src || "",
    };
  });
}

async function capture(page, name) {
  if (!artifactDirectory) return;
  await page.screenshot({
    path: `${artifactDirectory}/${name}.png`,
    fullPage: false,
  });
}
