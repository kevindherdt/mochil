import test from "node:test";
import assert from "node:assert/strict";
import { createThemeController } from "../static/theme-controller.js";

function fixture(values = {}, dark = true) {
  const saved = { ...values };
  const storage = {
    getItem: (key) => saved[key] ?? null,
    setItem: (key, value) => {
      saved[key] = value;
    },
  };
  const toggle = {
    setAttribute(name, value) {
      this[name] = value;
    },
  };
  const paletteSelect = {};
  const modeSelect = {};
  const vaultWidthSelect = {};
  const root = { dataset: {} };
  const media = {
    matches: dark,
    addEventListener(_name, listener) {
      this.listener = listener;
    },
  };
  createThemeController({
    root,
    toggle,
    paletteSelect,
    modeSelect,
    vaultWidthSelect,
    storage,
    media,
  });
  return { saved, storage, toggle, paletteSelect, modeSelect, vaultWidthSelect, root, media };
}

test("theme controller validates stored values and follows the device", () => {
  const state = fixture(
    { "bobobooks-palette": "invalid", "bobobooks-theme-mode": "system" },
    false,
  );
  assert.equal(state.root.dataset.palette, "bobobooks");
  assert.equal(state.root.dataset.theme, "light");
  assert.equal(state.toggle["aria-label"], "Use dark theme");
  assert.equal(state.saved["bobobooks-palette"], "bobobooks");
});

test("palettes dropped from the picker keep their settings by moving to the nearest theme", () => {
  for (const [saved, expected] of [
    ["dusk", "kiln"],
    ["twilight", "kiln"],
    ["tide", "kiln"],
    ["manjaro", "kiln"],
    ["material", "kiln"],
    ["ember", "bobobooks"],
    ["rosewood", "bobobooks"],
    ["ubuntu", "bobobooks"],
    ["nocturne", "kiln"],
  ]) {
    const state = fixture({ "bobobooks-palette": saved });
    assert.equal(state.root.dataset.palette, expected, `${saved} should wear ${expected}`);
    assert.equal(state.saved["bobobooks-palette"], expected, `${saved} should be re-saved as ${expected}`);
  }
});

test("theme controls persist palette and explicit brightness", () => {
  const state = fixture();
  state.paletteSelect.onchange({ target: { value: "eink" } });
  state.modeSelect.onchange({ target: { value: "dark" } });
  state.vaultWidthSelect.onchange({ target: { value: "wide" } });
  assert.equal(state.saved["bobobooks-palette"], "eink");
  assert.equal(state.saved["bobobooks-theme-mode"], "dark");
  assert.equal(state.root.dataset.palette, "eink");
  assert.equal(state.root.dataset.theme, "dark");
  assert.equal(state.saved["bobobooks-vault-width"], "wide");
});

function motionFixture(values = {}) {
  const saved = { ...values };
  const storage = { getItem: (key) => saved[key] ?? null, setItem: (key, value) => { saved[key] = value; } };
  const motionToggle = {};
  const root = { dataset: {} };
  createThemeController({ root, paletteSelect: {}, modeSelect: {}, vaultWidthSelect: {}, motionToggle, storage, media: { matches: false, addEventListener() {} } });
  return { saved, motionToggle, root };
}

test("animations are on by default, off by default with the E-ink theme, and the choice is kept", () => {
  assert.equal(motionFixture().root.dataset.motion, "full");
  assert.equal(motionFixture({ "bobobooks-palette": "eink" }).root.dataset.motion, "reduced");
  const choice = motionFixture({ "bobobooks-palette": "eink" });
  choice.motionToggle.onchange({ target: { checked: false } });
  assert.equal(choice.saved["bobobooks-reduce-motion"], "0");
  assert.equal(choice.root.dataset.motion, "full");
  const stored = motionFixture({ "bobobooks-reduce-motion": "1" });
  assert.equal(stored.root.dataset.motion, "reduced");
  assert.equal(stored.motionToggle.checked, true);
});

test("the reader page and the settings page agree on whether animations are off", async () => {
  const { motionReduced, applyMotion } = await import("../static/motion-preference.js");
  const storage = (values) => ({ getItem: (key) => values[key] ?? null });
  assert.equal(motionReduced(storage({}), "bobobooks", false), false);
  assert.equal(motionReduced(storage({}), "eink", false), true);
  assert.equal(motionReduced(storage({}), "bobobooks", true), true);
  assert.equal(motionReduced(storage({ "bobobooks-reduce-motion": "0" }), "eink", true), false);
  const root = { dataset: {} };
  applyMotion(root, storage({ "bobobooks-palette": "eink" }), { matchMedia: () => ({ matches: false }) });
  assert.equal(root.dataset.motion, "reduced");
});
