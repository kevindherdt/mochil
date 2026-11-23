// Shell variants (plan docs/rewrite-plan.md §4.2): phone below 760px AND tablet 760–1099 share the phone shell
// (never a half-sidebar); desktop is >=1100. This is the ONLY responsive switch in the app — everything else is
// container queries inside primitives (law L7), not per-screen @media.
import { createContext, useContext, useEffect, useState } from "react";

export type ShellVariant = "phone" | "desktop";

const DESKTOP_BREAKPOINT = 1100;

function variantFor(width: number): ShellVariant {
  return width >= DESKTOP_BREAKPOINT ? "desktop" : "phone";
}

const ShellVariantContext = createContext<ShellVariant>("phone");

// react-refresh/only-export-components would rather this hook lived in its own file; co-locating a context's
// provider with its one reader is the more common React pattern and not worth a second file for two lines.
// eslint-disable-next-line react-refresh/only-export-components
export function useShellVariant(): ShellVariant {
  return useContext(ShellVariantContext);
}

export function ShellVariantProvider({ children }: { children: React.ReactNode }) {
  const [variant, setVariant] = useState<ShellVariant>(() =>
    typeof window === "undefined" ? "phone" : variantFor(window.innerWidth),
  );

  useEffect(() => {
    const onResize = () => setVariant(variantFor(window.innerWidth));
    window.addEventListener("resize", onResize);
    return () => window.removeEventListener("resize", onResize);
  }, []);

  return <ShellVariantContext.Provider value={variant}>{children}</ShellVariantContext.Provider>;
}
