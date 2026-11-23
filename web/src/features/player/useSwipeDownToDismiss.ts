// features/player/useSwipeDownToDismiss.ts — "slide from big back to mini" (Kevin, 22 Nov: "make it possible to
// slide from mini to big player possible? Right now when big player is there its stuck there"). The big player
// doesn't need its own transformed "mini" state to animate into — collapsing it is just leaving the screen, so a
// swipe down past the threshold navigates back exactly like the header's back button does; a swipe that doesn't
// clear the threshold snaps back to place. The drag only follows the finger downward (translateY is clamped to
// >=0) — there is nothing above this screen to reveal by dragging up.
//
// The live delta is read from a ref inside `onPointerUp`, not from the `dragY` state closure: React batches the
// `setDragY` call from `onPointerMove`, so a pointerup dispatched in the same tick (as a synthetic drag in a
// test — and, rarely, a very fast real one) can still fire the *previous* render's `onPointerUp`, whose closure
// captured the state from before the last move. A ref is always current regardless of when React gets around to
// re-rendering.
import { useRef, useState } from "react";
import type { PointerEvent as ReactPointerEvent } from "react";

const DISMISS_THRESHOLD = 80;

export interface SwipeHandlers {
  onPointerDown: (event: ReactPointerEvent) => void;
  onPointerMove: (event: ReactPointerEvent) => void;
  onPointerUp: (event: ReactPointerEvent) => void;
  /** Apply to the dragged element: follows the finger while dragging, snaps back to 0 on release otherwise. */
  style: { transform: string; transition: string };
}

export function useSwipeDownToDismiss(onDismiss: () => void): SwipeHandlers {
  const startY = useRef<number | undefined>(undefined);
  const liveDragY = useRef(0);
  const [dragY, setDragY] = useState(0);
  const [dragging, setDragging] = useState(false);

  return {
    onPointerDown: (event) => {
      startY.current = event.clientY;
      setDragging(true);
    },
    onPointerMove: (event) => {
      if (startY.current === undefined) return;
      const next = Math.max(0, event.clientY - startY.current);
      liveDragY.current = next;
      setDragY(next);
    },
    onPointerUp: () => {
      startY.current = undefined;
      setDragging(false);
      const dismiss = liveDragY.current > DISMISS_THRESHOLD;
      liveDragY.current = 0;
      setDragY(0);
      if (dismiss) onDismiss();
    },
    style: {
      transform: `translateY(${dragY}px)`,
      // No easing while the finger is down — it must track 1:1; the snap-back on release is the only animated
      // part, and only that part needs a duration.
      transition: dragging ? "none" : "transform 150ms ease-out",
    },
  };
}
