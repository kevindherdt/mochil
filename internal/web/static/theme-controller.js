import { mountThemeSettings } from "./theme-settings-template.js?v=9";
import { mountThemeSwatches } from "./theme-swatches.js?v=1";
import { mountVaultSettings } from "./vault-settings-template.js?v=1";
import { mountReaderEngine } from "./reader-engine.js?v=3";
import { motionReduced } from "./motion-preference.js?v=1";

const palettes = new Set([
  "mochil",
  "kiln",
  "paper",
  "eink",
]);
// Ids saved by older builds. They wear the nearest of the four rather than being dropped.
const legacyPalettes = {
  dusk: "kiln",
  twilight: "kiln",
  tide: "kiln",
  manjaro: "kiln",
  material: "kiln",
  nocturne: "kiln",
  ember: "mochil",
  rosewood: "mochil",
  ubuntu: "mochil",
};
const canonicalPalette = (value) =>
  palettes.has(value) ? value : legacyPalettes[value] || "mochil";
const modes = new Set(["system", "light", "dark"]);
const vaultWidths = new Set(["centered", "wide"]);

export function createThemeController({
  document,
  root,
  toggle,
  paletteSelect,
  modeSelect,
  vaultWidthSelect,
  motionToggle,
  storage,
  media,
}) {
  if (document) {
    mountThemeSettings(document);
    mountThemeSwatches(document);
    mountVaultSettings(document);
    mountReaderEngine(document, storage);
    paletteSelect ??= document.querySelector("#theme-palette");
    modeSelect ??= document.querySelector("#theme-mode");
    vaultWidthSelect ??= document.querySelector("#vault-width");
    motionToggle ??= document.querySelector("#reduce-motion");
  }
  let palette = canonicalPalette(storage.getItem("mochil-palette"));
  let mode =
    storage.getItem("mochil-theme-mode") ||
    storage.getItem("mochil-theme") ||
    storage.getItem("opdshelf-theme") ||
    "system";
  // A build that dropped a palette keeps its settings instead of resetting to the default.
  try {
    if (storage.getItem("mochil-palette") !== palette) storage.setItem("mochil-palette", palette);
  } catch {
    /* private windows can refuse writes; the palette still applies for this visit */
  }
  if (!modes.has(mode)) mode = "system";
  let vaultWidth = storage.getItem("mochil-vault-width") || "centered";
  if (!vaultWidths.has(vaultWidth)) vaultWidth = "centered";

  const deviceWantsLessMotion = () => Boolean(document?.defaultView?.matchMedia?.("(prefers-reduced-motion: reduce)")?.matches);
  const reduceMotion = () => motionReduced(storage, palette, deviceWantsLessMotion());

  function apply() {
    const resolved =
      mode === "system" ? (media.matches ? "dark" : "light") : mode;
    root.dataset.palette = palette;
    root.dataset.theme = resolved;
    root.dataset.motion = reduceMotion() ? "reduced" : "full";
    if (motionToggle) motionToggle.checked = reduceMotion();
    toggle?.setAttribute(
      "aria-label",
      resolved === "dark" ? "Use light theme" : "Use dark theme",
    );
    if (paletteSelect) paletteSelect.value = palette;
    if (modeSelect) modeSelect.value = mode;
    if (vaultWidthSelect) vaultWidthSelect.value = vaultWidth;
  }

  function save() {
    storage.setItem("mochil-palette", palette);
    storage.setItem("mochil-theme-mode", mode);
    apply();
  }

  if (toggle)
    toggle.onclick = () => {
      mode = root.dataset.theme === "dark" ? "light" : "dark";
      save();
    };
  if (paletteSelect)
    paletteSelect.onchange = (event) => {
      palette = canonicalPalette(event.target.value);
      save();
    };
  if (modeSelect)
    modeSelect.onchange = (event) => {
      mode = event.target.value;
      save();
    };
  if (motionToggle)
    motionToggle.onchange = (event) => {
      storage.setItem("mochil-reduce-motion", event.target.checked ? "1" : "0");
      apply();
    };
  if (vaultWidthSelect)
    vaultWidthSelect.onchange = (event) => {
      vaultWidth = vaultWidths.has(event.target.value) ? event.target.value : "centered";
      storage.setItem("mochil-vault-width", vaultWidth);
      vaultWidthSelect.value = vaultWidth;
    };
  media.addEventListener?.("change", () => {
    if (mode === "system") apply();
  });
  apply();

  return { apply };
}
