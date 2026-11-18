import assert from "node:assert/strict";
import test from "node:test";
import {
  createReadingSyncSettings,
  readingSyncValue,
} from "../static/reading-sync-settings.js";
import {
  mountReadingSyncSettings,
  readingSyncSettingsHTML,
} from "../static/reading-sync-settings-template.js";

test("reading sync values normalize providers without leaking saved state", () => {
  assert.deepEqual(
    readingSyncValue({
      provider: "external",
      bridge: true,
      serverURL: " https://sync.example.test/ ",
      username: " reader ",
      password: "secret",
    }),
    {
      provider: "external",
      enabled: true,
      web_reader_bridge: true,
      server_url: "https://sync.example.test",
      username: "reader",
      password: "secret",
    },
  );
  assert.deepEqual(readingSyncValue({ provider: "invalid" }), {
    provider: "built-in",
    enabled: true,
    web_reader_bridge: false,
    server_url: "",
  });
});

test("reading sync template retains sync, plugin, app, and accessibility contracts", () => {
  for (const id of [
    "kosync-provider",
    "kosync-webreader-bridge",
    "kosync-external-fields",
    "kosync-server-url",
    "kosync-local-help",
    "kosync-credentials",
    "kosync-username",
    "kosync-password",
    "plugin-setup",
    "download-plugin",
    "plugin-status",
    "android-setup",
    "download-android",
    "android-version",
    "android-status",
  ])
    assert.match(readingSyncSettingsHTML, new RegExp(`id="${id}"`));
  assert.match(readingSyncSettingsHTML, /data-settings-panel="sync" hidden/);
  assert.match(readingSyncSettingsHTML, /role="status"/g);
});

test("reading sync template mounts once inside Settings", () => {
  const insertions = [];
  let mounted = false;
  const storage = {
    insertAdjacentHTML(position, markup) {
      insertions.push({ position, markup });
      mounted = true;
    },
  };
  const document = {
    querySelector(selector) {
      return selector === '[data-settings-panel="sync"]'
        ? mounted
          ? {}
          : null
        : storage;
    },
  };
  mountReadingSyncSettings(document);
  mountReadingSyncSettings(document);
  assert.deepEqual(insertions, [
    { position: "beforeend", markup: readingSyncSettingsHTML },
  ]);
});

test("reading sync controller owns dependent state and non-destructive serialization", () => {
  const field = (extra = {}) => ({
    value: "",
    checked: false,
    disabled: false,
    hidden: false,
    required: false,
    dataset: {},
    addEventListener(type, handler) {
      this[type] = handler;
    },
    ...extra,
  });
  const elements = {
    "#kosync-provider": field(),
    "#kosync-webreader-bridge": field(),
    "#kosync-server-url": field(),
    "#kosync-username": field(),
    "#kosync-password": field(),
    "#kosync-external-fields": field(),
    "#kosync-credentials": field(),
    "#kosync-local-help": field({ textContent: "" }),
    "#kosync-account-help": field({ textContent: "" }),
    "#plugin-setup": field(),
    "#download-plugin": field(),
    "#plugin-status": field({ textContent: "" }),
    "#download-android": field(),
    "#android-version": field(),
    "#android-status": field(),
  };
  const controller = createReadingSyncSettings({
    document: { querySelector: () => ({}) },
    $: (selector) => elements[selector],
    origin: "https://local.test",
  });
  controller.load(
    {
      provider: "external",
      web_reader_bridge: true,
      server_url: "https://sync.test",
      effective_username: "reader",
      password_configured: true,
    },
    "https://books.test",
  );
  assert.equal(elements["#kosync-external-fields"].hidden, false);
  assert.equal(elements["#kosync-local-help"].hidden, true);
  assert.equal(elements["#plugin-setup"].hidden, true);
  assert.equal(elements["#kosync-password"].placeholder, "Saved password");
  elements["#kosync-password"].value = "replacement";
  const settings = {
    kosync: {
      future_option: "kept",
      password_configured: true,
      effective_username: "old",
    },
  };
  controller.write(settings);
  assert.equal(settings.kosync.future_option, "kept");
  assert.equal(settings.kosync.password, "replacement");
  assert.equal(settings.kosync.username, "reader");
  assert.equal("password_configured" in settings.kosync, false);
  assert.equal("effective_username" in settings.kosync, false);
});

test("Android download check exposes the current build or a recoverable unavailable state", async () => {
  const element = () => ({
    textContent: "",
    href: "",
    attributes: {},
    removeAttribute(name) {
      delete this.attributes[name];
      if (name === "href") this.href = "";
    },
    setAttribute(name, value) {
      this.attributes[name] = value;
    },
  });
  const base = {
    "#kosync-provider": { value: "built-in" },
    "#kosync-webreader-bridge": {},
    "#kosync-server-url": {},
    "#kosync-username": {},
    "#kosync-password": {},
    "#download-android": element(),
    "#android-version": element(),
    "#android-status": element(),
  };
  const success = createReadingSyncSettings({
    document: { querySelector: () => ({}) },
    $: (selector) => base[selector],
    fetchImpl: async () => ({ ok: true, headers: { get: () => "0.16.1" } }),
  });
  await success.checkAndroidDownload();
  assert.equal(
    base["#android-version"].textContent,
    "Version 0.16.1 · Android 8 or newer",
  );
  const failure = createReadingSyncSettings({
    document: { querySelector: () => ({}) },
    $: (selector) => base[selector],
    fetchImpl: async () => ({ ok: false, status: 404 }),
  });
  await failure.checkAndroidDownload();
  assert.equal(base["#download-android"].attributes["aria-disabled"], "true");
});
