import assert from "node:assert/strict";
import test from "node:test";
import { createStorageSettings } from "../static/storage-settings.js";
import {
  mountStorageSettings,
  storageSettingsHTML,
} from "../static/storage-settings-template.js";

test("storage template retains cache and accessibility contracts", () => {
  for (const id of ["scan-every", "scan-help", "cover-cache-mib", "cover-cache-help"])
    assert.match(storageSettingsHTML, new RegExp(`id="${id}"`));
  assert.doesNotMatch(storageSettingsHTML, /reading-cache|remote books/);
  // The drop folder for Import now lives entirely on the Import page's own
  // picker (0.1.20); this card must never offer a second, conflicting way to
  // set the same two fields.
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

test("storage controller reads and writes only cache policy fields, leaving managed_ingest untouched", () => {
  const field = () => ({
    value: "",
    listeners: {},
    addEventListener(type, handler) {
      (this.listeners[type] ||= []).push(handler);
    },
    dispatch(type) {
      this.listeners[type]?.forEach((handler) => handler());
    },
  });
  const elements = {
    "#scan-every": field(),
    "#cover-cache-mib": field(),
  };
  let dirty = 0;
  const controller = createStorageSettings({
    document: { querySelector: () => ({}) },
    $: (selector) => elements[selector],
    onDirty: () => dirty++,
  });
  controller.load({
    scan_every: "30m",
    cover_cache: { maximum_bytes: 268435456, custom: "kept" },
    page_cache: { maximum_bytes: 2147483648, custom: "kept" },
  });
  assert.equal(elements["#cover-cache-mib"].value, 256);
  controller.bind();
  elements["#scan-every"].dispatch("input");
  assert.equal(dirty, 1);
  // A library whose drop folder was set from the Import page, with no trace
  // of managed_ingest in this card's own fields, must survive a Settings
  // save untouched -- this is the exact bug that silently deleted it before.
  const settings = {
    cover_cache: { custom: "kept" },
    page_cache: { custom: "kept" },
    managed_ingest: {
      incoming_directory: "/media/incoming",
      managed_directory: "/data/catalogs/managed-ingest",
      maximum_file_bytes: 8589934592,
    },
  };
  controller.write(settings);
  assert.equal(settings.scan_every, "30m");
  assert.equal(settings.cover_cache.custom, "kept");
  assert.equal(settings.page_cache.custom, "kept");
  assert.deepEqual(settings.managed_ingest, {
    incoming_directory: "/media/incoming",
    managed_directory: "/data/catalogs/managed-ingest",
    maximum_file_bytes: 8589934592,
  });
});
