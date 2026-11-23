// shell/ShellLayout.tsx — the shell's own route component, kept out of app/router.tsx because it is a component and
// the router module is configuration (a mixed module breaks fast refresh, and the lint rule says so).
//
// It decides one thing: whether this route gets chrome. The full-screen player is the only path that must not share
// the viewport with a dock, a header band or the mini player, and only the shell knows how to position — and therefore
// how to withhold — those. Reading the path here keeps that decision in one place instead of teaching every screen to
// hide things.
import { Outlet, useRouterState } from "@tanstack/react-router";
import { AppShell } from "./AppShell";
import { MiniPlayer } from "../features/player/MiniPlayer";

export function ShellLayout() {
  const pathname = useRouterState({ select: (state) => state.location.pathname });
  // The player owns the whole viewport; everything else shares it with the dock. (The book reader is the legacy
  // page at /reader/…, which is its own document — see docs/rewrite-progress.md's decision note.)
  const bare = pathname.startsWith("/play/");
  return (
    <AppShell player={bare ? undefined : <MiniPlayer />} bare={bare}>
      <Outlet />
    </AppShell>
  );
}
