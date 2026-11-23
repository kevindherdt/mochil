// PillRow — the library/media-type switcher (ui-overhaul-plan.md §4 "Header, phone"), also used for the AI sheet's
// avatar and voice pickers.
//
// It is data-driven: real library ids/names/icons are a feature concern, not this primitive's — it only knows how
// to render a selectable, horizontally-scrollable row of pills.
//
// Scrolling is the whole story here. The row is `overflow-x: auto` with the scrollbar hidden, which works with a
// trackpad or a finger and is *broken with a mouse*: a plain vertical wheel does nothing on a horizontal scroller,
// so a row that overflows by 2400px (this server has 30 narration voices) looks frozen past the few pills that
// happen to fit. Kevin: "i cant scroll the pills to select the other voices". So the row now owns three
// affordances:
//
//   1. a vertical wheel scrolls it horizontally when it actually overflows, and leaves the page alone when it does
//      not — the listener is attached natively with `passive: false`, because React's synthetic `onWheel` is
//      passive at the root and `preventDefault()` there is ignored (the page scrolls instead);
//   2. mouse drag scrolls it, with a 4px travel threshold so ending a drag never counts as a pill click;
//   3. the active pill is scrolled into view on mount, so a picker whose selection sits at position 24 does not
//      open showing the start of the list and leave you hunting for the tick.
//
// The scrollbar stays hidden: 30 pills above a visible scrollbar reads as a table column, and the right-edge fade
// is the affordance instead. `trailing` is pinned outside the scroll box so a control placed there stays reachable
// at any width.
import { useCallback, useEffect, useRef, useState, type ComponentType, type ReactNode } from "react";
import styles from "./PillRow.module.css";

export interface PillRowItem {
  id: string;
  label: string;
  /** Lucide-style glyph. `strokeWidth` is allowed because a 12px glyph needs a slightly heavier stroke to stay
      legible at the mock-up's pill size. */
  icon?: ComponentType<{ size?: number; strokeWidth?: number }>;
}

export interface PillRowProps {
  items: PillRowItem[];
  activeId?: string | undefined;
  onSelect?: (id: string) => void;
  /** A control pinned at the right edge, outside the scrolling pills (the browse-by Select, or an overflow `⋯`). */
  trailing?: ReactNode;
}

export function PillRow({ items, activeId, onSelect, trailing }: PillRowProps) {
  const scroller = useRef<HTMLDivElement | null>(null);
  // Fade the right edge only while something is actually hidden there; a row that fits should look finished.
  const [fading, setFading] = useState(false);
  // Drag state lives here rather than in the pointer handlers below so the effect can clear it: a drag that never
  // sees its pointerup must not survive into a later hover.
  const drag = useRef<{ startX: number; startLeft: number; moved: boolean; pointerId: number } | null>(null);
  // Set when a drag ends so the click closing that gesture does not select a pill.
  const justDragged = useRef(false);

  const measure = useCallback(() => {
    const el = scroller.current;
    if (el === null) return;
    setFading(el.scrollWidth - el.clientWidth > 2 && el.scrollLeft + el.clientWidth < el.scrollWidth - 2);
  }, []);

  useEffect(() => {
    const el = scroller.current;
    if (el === null) return undefined;

    function onWheel(event: WheelEvent) {
      const node = scroller.current;
      if (node === null) return;
      if (node.scrollWidth <= node.clientWidth + 2) return;
      // Shift-wheel and trackpad swipes arrive as deltaX and already work; only reinterpret the vertical axis.
      if (Math.abs(event.deltaY) <= Math.abs(event.deltaX)) return;
      const before = node.scrollLeft;
      node.scrollLeft += event.deltaY;
      // Take the gesture only if the row actually moved. At either end of the rail the page must keep scrolling:
      // swallowing it there is what makes the rail feel like it has grabbed the wheel.
      if (node.scrollLeft === before) return;
      event.preventDefault();
    }

    el.addEventListener("wheel", onWheel, { passive: false });
    el.addEventListener("scroll", measure, { passive: true });
    // Capture should guarantee these, but a release during a re-render (or a browser that drops capture) must never
    // leave the rail dragging. One window-level listener is cheap insurance.
    const forgetDrag = () => {
      // If the release never reached the rail (no capture, e.g. a pointer the browser would not capture), still mark
      // the gesture as a drag so its trailing click cannot select a pill.
      if (drag.current?.moved === true) justDragged.current = true;
      drag.current = null;
    };
    window.addEventListener("pointerup", forgetDrag);
    window.addEventListener("pointercancel", forgetDrag);
    const observer = new ResizeObserver(measure);
    observer.observe(el);

    measure();

    return () => {
      el.removeEventListener("wheel", onWheel);
      el.removeEventListener("scroll", measure);
      window.removeEventListener("pointerup", forgetDrag);
      window.removeEventListener("pointercancel", forgetDrag);
      observer.disconnect();
    };
  }, [items.length, measure]);

  // Keep the selection visible without moving the rail under the cursor.
  //
  // This used to re-centre the active pill on every selection change, which is why picking a voice felt like the pill
  // jumping to the left: you click the fourth visible pill, the list refetches, and the rail re-lays itself around
  // it. Centring is only right once, when the row opens and the saved choice may be twenty pills off-screen;
  // afterwards the rule is "move only if the selection is actually out of view, and then only as far as needed".
  const scrolledOnce = useRef(false);
  useEffect(() => {
    const el = scroller.current;
    if (el === null) return;
    const active = el.querySelector('[aria-selected="true"]');
    if (!(active instanceof HTMLElement)) return;

    if (!scrolledOnce.current) {
      scrolledOnce.current = true;
      el.scrollLeft = Math.max(0, active.offsetLeft - el.clientWidth / 2 + active.clientWidth / 2);
      return;
    }

    const left = active.offsetLeft;
    const right = left + active.offsetWidth;
    const viewLeft = el.scrollLeft;
    const viewRight = viewLeft + el.clientWidth;
    if (left < viewLeft) el.scrollLeft = Math.max(0, left - 12);
    else if (right > viewRight) el.scrollLeft = right - el.clientWidth + 12;
  }, [activeId, items.length]);

  // Drag-to-scroll for a mouse.
  //
  // Two constraints pull against each other here, and fixing one broke the other:
  //
  //   - a drag must not outlive its own release, or the rail follows the cursor forever ("i cant stop it from
  //     sticking to the scroll even when i click somewhere else on the page") — hence the capture and the
  //     window-level pointerup below;
  //   - but capturing on *press* retargets the pointerup to the rail, so the browser never dispatches the click on
  //     the pill underneath, and nothing in the rail can be selected any more ("now i cannot select the actual
  //     voice").
  //
  // So capture starts only once the pointer has travelled past the threshold. At that moment it is a drag, not a
  // click, and retargeting the release is exactly what we want; a press that never travels stays an ordinary click.
  function onPointerDown(event: React.PointerEvent<HTMLDivElement>) {
    if (event.pointerType !== "mouse") return; // touch and pen scroll natively
    if (event.button !== 0) return;
    const el = scroller.current;
    if (el === null || el.scrollWidth <= el.clientWidth + 2) return;
    justDragged.current = false;
    drag.current = { startX: event.clientX, startLeft: el.scrollLeft, moved: false, pointerId: event.pointerId };
  }

  function onPointerMove(event: React.PointerEvent<HTMLDivElement>) {
    const el = scroller.current;
    const started = drag.current;
    if (el === null || started === null) return;
    // Belt and braces next to capture: no button held means no drag, whatever state the ref is in.
    if (event.buttons === 0) {
      drag.current = null;
      return;
    }
    const travelled = event.clientX - started.startX;
    if (!started.moved && Math.abs(travelled) < 4) return;
    if (!started.moved) {
      started.moved = true;
      // Guarded: a pointer that is not active (replayed or synthetic) makes this throw, and a throw here would
      // abort the drag mid-gesture. The window-level pointerup still clears the state either way.
      try {
        el.setPointerCapture(event.pointerId);
      } catch {
        /* pointer not active */
      }
    }
    el.scrollLeft = started.startLeft - travelled;
  }

  function onPointerUp() {
    const started = drag.current;
    drag.current = null;
    if (started === null || !started.moved) return;
    // The click the browser synthesises for this same gesture must not select a pill. Cleared on the next press
    // rather than on a timer, so there is no window in which it can swallow a real click.
    justDragged.current = true;
    try {
      scroller.current?.releasePointerCapture(started.pointerId);
    } catch {
      /* capture already released */
    }
  }

  return (
    <div className={styles.row}>
      <div
        ref={scroller}
        className={fading ? `${styles.scroll} ${styles.fading}` : styles.scroll}
        role="tablist"
        onPointerDown={onPointerDown}
        onPointerMove={onPointerMove}
        onPointerUp={onPointerUp}
        onPointerCancel={onPointerUp}
      >
        {items.map((item) => {
          const Icon = item.icon;
          const active = item.id === activeId;
          return (
            <button
              key={item.id}
              type="button"
              role="tab"
              aria-selected={active}
              className={active ? `${styles.pill} ${styles.active}` : styles.pill}
              onClick={() => {
                if (justDragged.current) return;
                onSelect?.(item.id);
              }}
            >
              {Icon && <Icon size={12} strokeWidth={2} />}
              <span>{item.label}</span>
            </button>
          );
        })}
      </div>
      {trailing}
    </div>
  );
}
