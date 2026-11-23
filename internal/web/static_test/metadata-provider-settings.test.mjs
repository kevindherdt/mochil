import assert from "node:assert/strict";
import test from "node:test";
import {
  createMetadataProviderSettings,
  metadataProviderUsage,
} from "../static/metadata-provider-settings.js";
import {
  metadataProviderSettingsHTML,
  mountMetadataProviderSettings,
} from "../static/metadata-provider-settings-template.js";

test("metadata provider usage names every assigned library", () => {
  const configured = {
    hardcover_token_configured: true,
    tmdb_token_configured: true,
  };
  // "novels" and "movies" save no provider at all: the server picks one for
  // them, so the usage line must still name them (it used to say "not assigned"
  // for exactly the libraries the new folder flow creates).
  const libraries = [
    { id: "books", name: "Books", kind: "books", metadata_provider: "hardcover" },
    { id: "novels", kind: "books" },
    { id: "movies", name: "Movies", kind: "movies" },
  ];
  assert.equal(
    metadataProviderUsage(libraries, "hardcover", configured),
    "Used by Books, novels.",
  );
  assert.equal(
    metadataProviderUsage(libraries, "tmdb", configured),
    "Used by Movies.",
  );
  assert.equal(
    metadataProviderUsage(libraries, "none", configured),
    "Not assigned to a library.",
  );
});

test("metadata provider template keeps privacy and accessibility contracts", () => {
  for (const id of [
    "hardcover-token",
    "hardcover-provider-status",
    "tmdb-token",
    "tmdb-provider-status",
  ])
    assert.match(metadataProviderSettingsHTML, new RegExp(`id="${id}"`));
  assert.match(
    metadataProviderSettingsHTML,
    /Mochil never uploads media files/,
  );
  assert.match(metadataProviderSettingsHTML, /role="status"/g);
  assert.match(metadataProviderSettingsHTML, /data-metadata-provider-settings/);
});

test("metadata provider template mounts once at the end of the metadata panel", () => {
  const insertions = [];
  let mounted = false;
  const panel = {
    insertAdjacentHTML(position, markup) {
      insertions.push({ position, markup });
      mounted = true;
    },
  };
  const document = {
    querySelector(selector) {
      return selector === "[data-metadata-provider-settings]"
        ? mounted
          ? {}
          : null
        : panel;
    },
  };
  mountMetadataProviderSettings(document);
  mountMetadataProviderSettings(document);
  assert.deepEqual(insertions, [
    { position: "beforeend", markup: metadataProviderSettingsHTML },
  ]);
});

test("metadata provider controller tests tokens and preserves future provider fields", async () => {
  const input = () => ({
    value: "",
    placeholder: "",
    addEventListener(type, handler) {
      this[type] = handler;
    },
  });
  const button = () => ({ disabled: false, textContent: "Test connection" });
  const output = () => ({ textContent: "", dataset: {} });
  const elements = {
    "#hardcover-token": input(),
    "#tmdb-token": input(),
    "#hardcover-provider-status": output(),
    "#tmdb-provider-status": output(),
    '[data-test-provider="hardcover"]': button(),
    '[data-test-provider="tmdb"]': button(),
  };
  const calls = [];
  const controller = createMetadataProviderSettings({
    document: { querySelector: () => ({}) },
    $: (selector) => elements[selector],
    requestJSON: async (...args) => {
      calls.push(args);
      return {
        data: {
          connected: true,
          message: "Connected.",
          libraries: [{ name: "Books" }],
        },
      };
    },
  });
  controller.load({ hardcover_token_configured: true }, [
    { id: "books", name: "Books", kind: "books", metadata_provider: "hardcover" },
  ]);
  assert.equal(elements["#hardcover-token"].placeholder, "Saved token");
  assert.equal(
    elements["#hardcover-provider-status"].textContent,
    "Hardcover token saved. Used by Books.",
  );
  controller.bind();
  elements["#hardcover-token"].value = " pasted-token ";
  await elements['[data-test-provider="hardcover"]'].onclick();
  assert.equal(calls[0][0], "/api/metadata/provider-connection");
  assert.deepEqual(JSON.parse(calls[0][1].body), {
    provider: "hardcover",
    token: "pasted-token",
  });
  assert.equal(
    elements["#hardcover-provider-status"].dataset.state,
    "connected",
  );
  const settings = {
    metadata_providers: {
      future_provider: "kept",
      hardcover_token_configured: true,
    },
  };
  controller.write(settings);
  assert.equal(settings.metadata_providers.future_provider, "kept");
  assert.equal(settings.metadata_providers.hardcover_token, "pasted-token");
  assert.equal(
    "hardcover_token_configured" in settings.metadata_providers,
    false,
  );
});

// A whole-library rewrite of titles is maintenance, so it lives with the metadata provider in Settings
// instead of on the shelf. These pin both halves: the section exists in the libraries panel, and the video
// shelf no longer mounts the bar among its browse controls.
test("bulk TMDB matching renders one row per video library, and none for books", () => {
  const element = (tag = "div") => {
    let text = "";
    const node = {
      tagName: tag,
      dataset: {},
      hidden: false,
      disabled: false,
      value: "",
      placeholder: "",
      children: [],
      append(child) {
        node.children.push(child);
      },
      insertAdjacentHTML() {},
      addEventListener() {},
    };
    // Setting textContent to "" removes the children, as it does in the DOM — the renderer relies on it
    // to clear yesterday's rows before drawing today's.
    Object.defineProperty(node, "textContent", {
      get: () => text,
      set: (value) => {
        text = value;
        if (value === "") node.children.length = 0;
      },
    });
    node.querySelector = (selector) => {
      node.found ??= {};
      node.found[selector] ??= element();
      return node.found[selector];
    };
    return node;
  };
  const section = element("section");
  section.hidden = true;
  const host = element();
  const document = {
    createElement: (tag) => element(tag),
    querySelector(selector) {
      if (selector === "[data-metadata-provider-settings]") return null;
      if (selector === '[data-settings-panel="libraries"]') return element("section");
      if (selector === "[data-video-bulk-settings]") return section;
      if (selector === "[data-video-bulk-libraries]") return host;
      return null;
    },
  };
  const controller = createMetadataProviderSettings({
    document,
    $: () => element("input"),
    requestJSON: async () => ({ data: {} }),
  });

  controller.load({}, [
    { id: "books", name: "Library", kind: "books" },
    { id: "films", name: "Films", kind: "movies" },
    { id: "shows", name: "Shows", kind: "tv" },
  ]);
  assert.equal(section.hidden, false);
  assert.deepEqual(
    host.children.map((row) => row.dataset.library),
    ["films", "shows"],
  );

  controller.load({}, [{ id: "books", name: "Library", kind: "books" }]);
  assert.equal(section.hidden, true);
  assert.equal(host.children.length, 0);
});

test("the video shelf no longer mounts the bulk match bar", async () => {
  const { readFile } = await import("node:fs/promises");
  const source = await readFile(
    new URL("../static/video-library.js", import.meta.url),
    "utf8",
  );
  assert.doesNotMatch(source, /mountVideoMetadataMatch/);
  assert.doesNotMatch(source, /metadataTools/);
});

test("the bulk match section is part of the libraries panel template", () => {
  assert.match(metadataProviderSettingsHTML, /data-video-bulk-settings hidden/);
  assert.match(metadataProviderSettingsHTML, /data-video-bulk-libraries/);
  assert.match(metadataProviderSettingsHTML, /Match a video library with TMDB/);
});
