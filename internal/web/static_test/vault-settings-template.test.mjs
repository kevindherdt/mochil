import assert from "node:assert/strict";
import test from "node:test";
import {
  mountVaultSettings,
  vaultSettingsHTML,
} from "../static/vault-settings-template.js";

test("Appearance owns the Wiki reading-width preference", () => {
  assert.match(vaultSettingsHTML, /id="vault-width"/);
  assert.match(vaultSettingsHTML, /Readable line length/);
  assert.match(vaultSettingsHTML, /Wide, aligned left/);
});

test("Wiki page width mounts once inside Appearance", () => {
  const insertions = [];
  let mounted = false;
  const card = {
    insertAdjacentHTML(position, markup) {
      insertions.push({ position, markup });
      mounted = true;
    },
  };
  const document = {
    querySelector(selector) {
      return selector === "#vault-width"
        ? mounted
          ? {}
          : null
        : card;
    },
  };
  mountVaultSettings(document);
  mountVaultSettings(document);
  assert.deepEqual(insertions, [
    { position: "beforeend", markup: vaultSettingsHTML },
  ]);
});
