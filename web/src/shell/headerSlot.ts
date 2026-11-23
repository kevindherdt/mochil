// shell/headerSlot.ts — the React context behind ScreenHeader, in its own file because a module that exports a
// component must export nothing else for Fast Refresh to work (eslint react-refresh/only-export-components).
import { createContext } from "react";

/** The header band element the shell mounted, or null before its first layout pass. */
export const HeaderSlotContext = createContext<HTMLElement | null>(null);
