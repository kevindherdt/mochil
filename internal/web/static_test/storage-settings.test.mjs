import assert from "node:assert/strict";
import test from "node:test";
import { createStorageSettings } from "../static/storage-settings.js";
import {
  mountStorageSettings,
  storageSettingsHTML,
} from "../static/storage-settings-template.js";

test("storage template retains cache, ingest, and accessibility contracts", () => {
  for (const id of [
    "scan-every",
    "scan-help",
    "cover-cache-mib",
    "cover-cache-help",
    "managed-ingest-incoming",
    "managed-ingest-help",
    "managed-ingest-target",
    "managed-ingest-target-help",
    "managed-ingest-gib",
    "managed-ingest-limit-help",
  ])
    assert.match(storageSettingsHTML, new RegExp(`id="${id}"`));
  assert.doesNotMatch(storageSettingsHTML, /reading-cache|remote books/);
  assert.match(
    storageSettingsHTML,
    /never deletes incoming files or replaces managed files/,
  );
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

test("storage controller owns dependent validation and preserves cache policy fields", () => {
  const field = () => ({
    value: "",
    required: false,
    disabled: false,
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
    "#managed-ingest-incoming": field(),
    "#managed-ingest-target": field(),
    "#managed-ingest-gib": field(),
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
  assert.equal(elements["#managed-ingest-gib"].disabled, true);
  controller.bind();
  elements["#managed-ingest-incoming"].value = " /incoming ";
  elements["#managed-ingest-target"].value = " /library ";
  elements["#managed-ingest-gib"].value = "12";
  elements["#managed-ingest-incoming"].dispatch("input");
  assert.equal(dirty, 1);
  assert.equal(elements["#managed-ingest-incoming"].required, true);
  assert.equal(elements["#managed-ingest-target"].required, true);
  assert.equal(elements["#managed-ingest-gib"].disabled, false);
  const settings = {
    cover_cache: { custom: "kept" },
    page_cache: { custom: "kept" },
  };
  controller.write(settings);
  assert.equal(settings.scan_every, "30m");
  assert.equal(settings.cover_cache.custom, "kept");
  assert.equal(settings.page_cache.custom, "kept");
  assert.deepEqual(settings.managed_ingest, {
    incoming_directory: "/incoming",
    managed_directory: "/library",
    maximum_file_bytes: 12884901888,
  });
});
