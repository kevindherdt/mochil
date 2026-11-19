import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

const css = await readFile(
  new URL("../static/vault.css", import.meta.url),
  "utf8",
);
const documentLibrary = await readFile(
  new URL("../static/document-library.js", import.meta.url),
  "utf8",
);
const app = await readFile(new URL("../static/app.js", import.meta.url), "utf8");
const router = await readFile(new URL("../static/library-view-router.js", import.meta.url), "utf8");
const deferredRoutes = await readFile(
  new URL("../static/deferred-route-controllers.js", import.meta.url),
  "utf8",
);

test("vault keeps shell controls anchored while the page owns scrolling", () => {
  assert.match(css, /body\.vault-reading\{overflow:hidden\}/);
  assert.match(css, /\.vault-workspace\{[^}]*height:100%[^}]*overflow:hidden/);
  assert.match(css, /\.vault-pane\{[^}]*overflow-y:auto[^}]*overscroll-behavior:contain/);
  assert.match(css, /\.vault-tree\{[^}]*overflow:auto/);
});

test("vault collapse always leaves an external reopen control", () => {
  assert.match(
    documentLibrary,
    /\$\{sidebarToggle\(false\)\}<aside>/,
  );
  assert.match(
    css,
    /\.vault-workspace\[data-sidebar=open\]>\.vault-sidebar-toggle\{display:none\}/,
  );
});

test("leaving a vault for video or audio restores normal page scrolling", () => {
  assert.doesNotMatch(app, /^import .*document-library\.js/m);
  // Any asset version: pinning one here turns every cache-bust into a test failure.
  assert.match(deferredRoutes, /import\("\.\/document-library\.js\?v=\d+"\)/);
  assert.match(deferredRoutes, /mountDocumentLibrary\(document\)/);
  const videoBranch = router.slice(
    router.indexOf("if (isVideoLibrary())"),
    router.indexOf("if (isPlannedAudioLibrary())"),
  );
  const audioBranch = router.slice(
    router.indexOf("if (isPlannedAudioLibrary())"),
    router.indexOf("if (isDocumentLibrary())"),
  );
  assert.match(videoBranch, /deferredRoutes\.abortDocument\(\)/);
  assert.match(audioBranch, /deferredRoutes\.abortDocument\(\)/);
});

test("settings and metadata remain outside the Home dependency graph", () => {
  assert.doesNotMatch(app, /^import .*settings\.js/m);
  assert.doesNotMatch(app, /^import .*metadata-review\.js/m);
  assert.match(deferredRoutes, /import\("\.\/settings\.js\?v=\d+"\)/);
  assert.match(deferredRoutes, /import\("\.\/metadata-review\.js\?v=\d+"\)/);
  assert.match(deferredRoutes, /view\.inert = true/);
  assert.match(deferredRoutes, /view\.removeAttribute\("aria-busy"\)/);
});
