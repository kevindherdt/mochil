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
