// lib/useTheme — applies the palette/mode to document.documentElement (plan §4.1: themes are selected via
// [data-palette]/[data-theme] attributes, mirroring the legacy app so the existing theme-switcher UX needs no
// new plumbing at cutover). CSS custom properties scoped to `:root[data-palette=…]` in tokens.css only take
// effect when those attributes sit on the actual root element — not on an arbitrary wrapper div, which is a
// real bug this hook exists to make impossible to repeat (found while building the gallery: swatches and page
// background rendered with no colour at all because the attributes were on a <div>, not <html>).
import { useEffect, useState } from "react";
import { defaultTheme, type CanonicalTheme } from "./tokenMeta";

export type ThemeMode = "dark" | "light";

export interface ThemeState {
  palette: CanonicalTheme;
  mode: ThemeMode;
  setPalette: (palette: CanonicalTheme) => void;
  setMode: (mode: ThemeMode) => void;
}

/** Defaults to `defaultTheme` (Tide/kiln, tokens.json's meta.reference_theme) — not `canonicalThemes[0]`, which
    is only the order a switcher lists themes in. Kevin picked Tide as the product's default look, so a fresh
    visitor gets it without touching a setting. */
export function useTheme(initialPalette: CanonicalTheme = defaultTheme, initialMode: ThemeMode = "dark"): ThemeState {
  const [palette, setPalette] = useState<CanonicalTheme>(initialPalette);
  const [mode, setMode] = useState<ThemeMode>(initialMode);

  useEffect(() => {
    document.documentElement.dataset.palette = palette;
    document.documentElement.dataset.theme = mode;
  }, [palette, mode]);

  return { palette, mode, setPalette, setMode };
}
