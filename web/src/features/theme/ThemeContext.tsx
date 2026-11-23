// features/theme/ThemeContext.tsx — lifts `lib/useTheme` into a context so a Settings screen (a different
// component subtree than App.tsx's root) can read and change the active theme, and persists the pick to
// localStorage so it survives a reload — previously every visit reset silently to Tide/dark regardless of what
// had been picked, because `useTheme` was only ever called once at the app root with no storage underneath it.
import { createContext, useContext, useEffect, useState } from "react";
import type { ReactNode } from "react";
import { defaultTheme, type CanonicalTheme } from "../../lib/tokenMeta";
import type { ThemeMode } from "../../lib/useTheme";

const STORAGE_KEY = "mochil:theme";

interface StoredTheme {
  palette: CanonicalTheme;
  mode: ThemeMode;
}

function readStored(): StoredTheme | undefined {
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    return raw ? (JSON.parse(raw) as StoredTheme) : undefined;
  } catch {
    return undefined;
  }
}

export interface ThemeContextValue {
  palette: CanonicalTheme;
  mode: ThemeMode;
  setPalette: (palette: CanonicalTheme) => void;
  setMode: (mode: ThemeMode) => void;
}

const ThemeContext = createContext<ThemeContextValue | undefined>(undefined);

export function ThemeProvider({ children }: { children: ReactNode }) {
  const stored = readStored();
  const [palette, setPalette] = useState<CanonicalTheme>(stored?.palette ?? defaultTheme);
  const [mode, setMode] = useState<ThemeMode>(stored?.mode ?? "dark");

  useEffect(() => {
    document.documentElement.dataset.palette = palette;
    document.documentElement.dataset.theme = mode;
    try {
      window.localStorage.setItem(STORAGE_KEY, JSON.stringify({ palette, mode }));
    } catch {
      // A fresh visitor with no storage (or a full quota) just gets the default every time — not worth failing
      // the whole app over.
    }
  }, [palette, mode]);

  return <ThemeContext.Provider value={{ palette, mode, setPalette, setMode }}>{children}</ThemeContext.Provider>;
}

// react-refresh/only-export-components would rather this hook lived in its own file; co-locating a context's
// provider and consumer hook is worth the HMR cost (matches features/player/PlayerContext.tsx's own precedent).
// eslint-disable-next-line react-refresh/only-export-components
export function useThemeContext(): ThemeContextValue {
  const value = useContext(ThemeContext);
  if (value === undefined) throw new Error("useThemeContext must be used inside <ThemeProvider>");
  return value;
}
