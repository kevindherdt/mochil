import assert from "node:assert/strict";
import test from "node:test";
import { createStorageSettings } from "../static/storage-settings.js";
import {
  mountStorageSettings,
  storageSettingsHTML,
} from "../static/storage-settings-template.js";

test("the scanning card is one action, and no knobs", () => {
  assert.match(storageSettingsHTML, /id="scan-settings"/);
  assert.match(storageSettingsHTML, /id="scan-all-settings"/);
  for (const retired of ["scan-every", "cover-cache-mib", "scan-help", "cover-cache-help"]) {
    assert.ok(!storageSettingsHTML.includes(`id="${retired}"`), `retired field ${retired} came back`);
  }
  assert.doesNotMatch(storageSettingsHTML, /Default scan interval|Cover cache/);
  // The drop folder for Import lives entirely on the Import page's own picker; this card must never
  // offer a second, conflicting way to set the same two fields.
  assert.doesNotMatch(storageSettingsHTML, /managed-ingest/);
});

test("storage settings mount once inside the Libraries panel", () => {
  const insertions = [];
  let mounted = false;
  const libraries = {
    insertAdjacentHTML(position, markup) {
      insertions.push({ position, markup });
      mounted = true;
    },
  };
  const document = {
    querySelector(selector) {
      return selector === "#scan-settings"
        ? mounted
          ? {}
          : null
        : selector === '[data-settings-panel="libraries"]'
          ? libraries
          : null;
    },
  };
  mountStorageSettings(document);
  mountStorageSettings(document);
  assert.deepEqual(insertions, [
    { position: "beforeend", markup: storageSettingsHTML },
  ]);
});

test("the storage controller leaves the settings document alone", () => {
  // The card only offers a button now. load() and write() must not create, drop or normalise anything:
  // they used to own the scan interval and the cover cache, and a save with those fields missing was how
  // an untouched value once got overwritten.
  const controller = createStorageSettings({
    document: { querySelector: () => ({}) },
    $: () => ({ addEventListener() {} }),
    onDirty: () => {},
  });
  const settings = {
    scan_every: "30m",
    cover_cache: { maximum_bytes: 268435456, custom: "kept" },
    page_cache: { maximum_bytes: 2147483648, custom: "kept" },
    managed_ingest: { incoming_directory: "/in", managed_directory: "/out" },
  };
  const before = JSON.stringify(settings);
  controller.load(settings);
  controller.write(settings);
  assert.equal(JSON.stringify(settings), before, "the controller changed the settings document");
});

