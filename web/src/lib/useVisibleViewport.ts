// useVisibleViewport — the ONE hook that reads real viewport geometry (plan docs/rewrite-plan.md §4.2, law L8).
// Lesson this exists to never repeat: Firefox on Android keeps its own bottom toolbar and lays the page out
// *behind* it, so `100dvh` reports the large (occluded) viewport there while Chrome has none, and a fixed
// one-screen-tall frame built on `100dvh` ends up with its dock pushed down and clipped under Firefox's bar
// (legacy bug, fixed in web build 0.1.86 via the same visualViewport approach, ported here as the hook version).
//
// This hook is the only thing in web/ that is allowed to read `window.visualViewport` or `innerHeight` directly
// (enforced by the eslint rule for L1) and the only thing that publishes `--vvh`/`--dock-bottom` as CSS vars on
// the document root; everything else — including `shell/`'s own components — reads those vars, never the
// viewport itself.
import { useEffect, useState } from "react";

export interface VisibleViewport {
  /** Height, in CSS px, of the space actually visible above any on-screen browser chrome (keyboard, toolbar). */
  height: number;
  /** How many px of `innerHeight` at the bottom are hidden behind browser chrome (e.g. Firefox's nav bar). */
  hiddenBottom: number;
  /** `visualViewport.offsetTop` — how far the visible viewport has scrolled down inside the layout viewport.
      Android Chrome auto-scrolls the page to keep a focused input clear of the keyboard, and a `position:
      fixed` element tracks the *layout* viewport, not the visual one — without compensating by this offset, the
      whole shell visibly drifts away from the real viewport the moment the keyboard opens (reported live: the
      dock and mini player ended up stacked mid-screen, above the keyboard, instead of pinned to its true edge). */
  offsetTop: number;
}

function measure(): VisibleViewport {
  const vv = window.visualViewport;
  if (!vv) {
    // No visualViewport API (old WebKit, some embedded webviews): innerHeight is the best we have, and it is
    // exactly the legacy bug's blind spot — but with no API to do better, there is nothing more to read.
    return { height: window.innerHeight, hiddenBottom: 0, offsetTop: 0 };
  }
  // `hiddenBottom` exists for persistent browser chrome that `.shell`'s own height+transform does NOT already
  // compensate for (Firefox's bottom toolbar, present with `offsetTop === 0` — nothing has scrolled, the chrome
  // is just permanently there). It must stay 0 whenever `offsetTop > 0`: that only happens when Android Chrome
  // is actively scrolling the layout viewport to clear a focused input from the keyboard, and at that point
  // `.shell`'s `height: var(--vvh)` + `transform: translateY(var(--vv-offset-top))` already places the shell
  // (and the dock inside it) exactly at the true visible edge. Computing hiddenBottom from the same numbers in
  // that case double-counts the keyboard's own occlusion on top of that already-correct placement, yanking the
  // sticky dock further up inside the shell — reported live as "the dock flies to the top" the moment the
  // keyboard opens (22 Nov 2026), reproduced by dispatching a synthetic visualViewport resize with a shrunk
  // height and a nonzero offsetTop, the same shape a real on-screen keyboard produces.
  const hiddenBottom = vv.offsetTop > 0 ? 0 : Math.max(0, window.innerHeight - vv.height);
  return { height: vv.height, hiddenBottom, offsetTop: vv.offsetTop };
}

/**
 * Tracks the real visible viewport (accounting for on-screen browser chrome and the virtual keyboard) and
 * publishes it as `--vvh` (height, px), `--dock-bottom` (hidden-bottom offset, px) and `--vv-offset-top` (the
 * keyboard-scroll compensation above) on `document.documentElement`. `shell/` components read those three CSS
 * vars; nothing else in the app should call this hook directly.
 */
export function useVisibleViewport(): VisibleViewport {
  const [viewport, setViewport] = useState<VisibleViewport>(() =>
    typeof window === "undefined" ? { height: 0, hiddenBottom: 0, offsetTop: 0 } : measure(),
  );

  useEffect(() => {
    const publish = () => {
      const next = measure();
      // The CSS variables are the real output; the state is for callers that want the numbers. Returning the previous
      // object when nothing moved lets React bail out — a keyboard opening fires dozens of these, and each one used to
      // re-render the whole shell and re-style a DOM holding hundreds of cards.
      setViewport((previous) =>
        previous.height === next.height && previous.hiddenBottom === next.hiddenBottom && previous.offsetTop === next.offsetTop
          ? previous
          : next,
      );
      const root = document.documentElement;
      root.style.setProperty("--vvh", `${next.height}px`);
      root.style.setProperty("--dock-bottom", `${next.hiddenBottom}px`);
      root.style.setProperty("--vv-offset-top", `${next.offsetTop}px`);
    };
    publish();

    const vv = window.visualViewport;
    vv?.addEventListener("resize", publish);
    vv?.addEventListener("scroll", publish);
    window.addEventListener("resize", publish);
    return () => {
      vv?.removeEventListener("resize", publish);
      vv?.removeEventListener("scroll", publish);
      window.removeEventListener("resize", publish);
    };
  }, []);

  return viewport;
}
