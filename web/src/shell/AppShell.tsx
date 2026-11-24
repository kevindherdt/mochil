// AppShell — the only place that positions chrome (plan §3, §4.2). Picks PhoneShell or DesktopShell from
// useShellVariant and renders the header band + Dock/Sidebar + PlayerSlot around whatever screen the router gives it.
//
// The header band is a sibling of the scrolling region, not part of it: a screen's toolbar (the library pills,
// the browse-by menu) has to stay put while the grid scrolls. Screens fill it with <ScreenHeader>; the shell owns
// the element, the position and the fact that it never scrolls.
import { useState, type ReactNode } from "react";
import { useVisibleViewport } from "../lib/useVisibleViewport";
import { useShellVariant } from "./ShellVariant";
import { Dock } from "./Dock";
import { Sidebar } from "./Sidebar";
import { HeaderSlotContext } from "./headerSlot";
import { useAiEnabled } from "../lib/useModules";
import styles from "./AppShell.module.css";

export interface AppShellProps {
  /** The current screen, rendered by the router. */
  children: ReactNode;
  /** Lyana is conditional (legacy dock.js: without the module, the nav stays four places). */
  hermes?: boolean;
  /** The mini player / now-playing bar — a fixed slot (plan §5.6: "shell slots, not per-screen"). */
  player?: ReactNode;
  /** Chrome-free: no header band, no padding, no dock/sidebar, no player slot. For screens that own the whole
   *  viewport on purpose — the full-screen video player. The shell still tracks the visible viewport, so `--vvh` is
   *  published for those screens exactly as it is for every other (law L1: only shell/ reads viewport units). */
  bare?: boolean;
}

export function AppShell({ children, hermes, player, bare = false }: AppShellProps) {
  useVisibleViewport();
  const variant = useShellVariant();
  // Lyana is conditional (legacy dock.js: without the module, the nav stays four places). The real answer comes
  // from /api/modules, not a hard-coded default — the mock-up shows five places because the server has the module.
  const aiEnabled = useAiEnabled();
  const showLyana = hermes === undefined ? aiEnabled : hermes;
  // State, not a ref: the slot element has to trigger a re-render once it exists, or a screen's portal would
  // find nothing to render into on the first pass and never retry.
  const [headerNode, setHeaderNode] = useState<HTMLDivElement | null>(null);

  if (bare) {
    return (
      <div className={styles.shellBare} data-shell="">
        {children}
      </div>
    );
  }

  const header = <div className={styles.header} ref={setHeaderNode} />;

  if (variant === "desktop") {
    return (
      // data-shell is the frame the layout gate measures: <body> is only as tall as the window, so measuring it said
      // "the frame did not follow the visual viewport" whatever the shell did.
      <div className={styles.shell} data-shell="">
        <div className={styles.sidebar}>
          <Sidebar hermes={showLyana} />
        </div>
        <div className={styles.desktopMain}>
          {header}
          <HeaderSlotContext.Provider value={headerNode}>
            <main className={styles.main}>{children}</main>
          </HeaderSlotContext.Provider>
          {player}
        </div>
      </div>
    );
  }

  return (
    <div className={styles.shell} data-shell="">
      {header}
      <HeaderSlotContext.Provider value={headerNode}>
        <main className={styles.main}>{children}</main>
      </HeaderSlotContext.Provider>
      {player}
      <div className={styles.dock}>
        <Dock hermes={showLyana} />
      </div>
    </div>
  );
}
