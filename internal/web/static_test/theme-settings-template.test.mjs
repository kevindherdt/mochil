import assert from "node:assert/strict";
import test from "node:test";
import {
  mountThemeSettings,
  themeSettingsHTML,
} from "../static/theme-settings-template.js";

test("appearance template retains palette, brightness, and settings contracts", () => {
  assert.match(themeSettingsHTML, /data-settings-panel="appearance" hidden/);
  assert.match(themeSettingsHTML, /id="theme-palette"/);
  assert.match(themeSettingsHTML, /id="theme-mode"/);
  for (const value of [
    "mochil",
    "kiln",
    "paper",
    "eink",
    "system",
    "light",
    "dark",
  ])
    assert.match(themeSettingsHTML, new RegExp(`value="${value}"`));
  for (const gone of ["dusk", "twilight", "tide", "rosewood", "ember", "manjaro", "ubuntu", "material"])
    assert.doesNotMatch(themeSettingsHTML, new RegExp(`value="${gone}"`));
  assert.match(themeSettingsHTML, /E-ink removes shadows/);
  assert.equal((themeSettingsHTML.match(/<option value="(?:mochil|kiln|paper|eink)"/g) || []).length, 4);
  assert.doesNotMatch(themeSettingsHTML, /nocturne/i);
});

test("Appearance has no tap-zone select (the reader owns it) and no background picker", () => {
  assert.doesNotMatch(themeSettingsHTML, /tap-zones-default/);
  assert.doesNotMatch(themeSettingsHTML, /Tap zones/i);
  assert.doesNotMatch(themeSettingsHTML, /theme-backdrop/);
});

test("appearance settings mount once inside Settings", () => {
  const insertions = [];
  let mounted = false;
  const settings = {
    insertAdjacentHTML(position, markup) {
      insertions.push({ position, markup });
      mounted = true;
    },
  };
  const document = {
    querySelector(selector) {
      return selector === '[data-settings-panel="appearance"]'
        ? mounted
          ? {}
          : null
        : settings;
    },
  };
  mountThemeSettings(document);
  mountThemeSettings(document);
  assert.deepEqual(insertions, [
    { position: "beforeend", markup: themeSettingsHTML },
  ]);
});
