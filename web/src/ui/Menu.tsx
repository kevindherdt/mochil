// Menu — a compact icon trigger with a small popover of choices.
//
// It carries MORE than one axis on purpose: the library header had a grouping menu (All books / Series / Authors)
// and a separate cover/list toggle, two icons that looked alike and did related things. Kevin: "combine the all
// books, series, authors menu together with the card/list view icon so we can reduce the icon and have more
// functions in it". So items may be grouped into sections, and any number of them may be checked — one for the
// browse mode, one for the layout.
//
// Deliberately tiny otherwise: no nesting, no icons inside items, closes on pick / outside click / Escape.
import { useEffect, useId, useRef, useState, type ComponentType } from "react";
import { ChevronDown } from "lucide-react";
import styles from "./Menu.module.css";

export interface MenuItem {
  id: string;
  label: string;
  /** Items sharing a section render under one small heading, with a rule between sections. */
  section?: string;
}

export interface MenuProps {
  items: MenuItem[];
  /** Every id that is currently selected — the menu shows a tick on each (a browse mode AND a layout). */
  activeIds?: string[];
  onSelect: (id: string) => void;
  /** The trigger's glyph. Without `text` it is the whole button, so `label` carries its accessible name. */
  icon: ComponentType<{ size?: number }>;
  label: string;
  /** Shown beside the glyph — for menus where the current choice matters ("Books", "All books"). */
  text?: string;
  /**
   * Which edge the panel lines up with. `auto` (the default) picks it from where the trigger sits: a trigger in the
   * right half of the window opens leftwards, one in the left half opens rightwards, so the panel lands on screen
   * either way. It used to be anchored right always, which cut "Light Novels" down to "ovels" at the left edge; the
   * fix overcorrected and cut the library header's own menu at the right edge. `start`/`end` remain as overrides.
   */
  align?: "auto" | "start" | "end";
  /** Off while the thing it would start is already running: two jobs at once lose each other's progress. */
  disabled?: boolean;
}

export function Menu({ items, activeIds = [], onSelect, icon: Icon, label, text, align = "auto", disabled = false }: MenuProps) {
  const [open, setOpen] = useState(false);
  // Decided from the trigger's own position when the panel opens: there is no layout to measure before that, and
  // measuring after opening would show the panel on the wrong side for a frame.
  const [reversed, setReversed] = useState(false);
  const wrap = useRef<HTMLSpanElement>(null);
  const panelId = useId();

  useEffect(() => {
    if (!open) return;
    function onDocumentClick(event: MouseEvent) {
      if (!wrap.current?.contains(event.target as Node)) setOpen(false);
    }
    function onKey(event: KeyboardEvent) {
      if (event.key === "Escape") setOpen(false);
    }
    document.addEventListener("mousedown", onDocumentClick);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onDocumentClick);
      document.removeEventListener("keydown", onKey);
    };
  }, [open]);

  // Section headings, derived without mutating anything: a heading is shown on the first item of its section.
  const rows = items.map((item, index) => ({
    item,
    heading: item.section !== undefined && item.section !== items[index - 1]?.section ? item.section : undefined,
  }));

  return (
    <span className={styles.wrap} ref={wrap}>
      <button
        type="button"
        className={styles.trigger}
        aria-label={label}
        aria-haspopup="menu"
        aria-expanded={open}
        aria-controls={open ? panelId : undefined}
        disabled={disabled}
        onClick={() => {
          if (disabled) return;
          const rect = wrap.current?.getBoundingClientRect();
          if (rect !== undefined && !open) setReversed(rect.left > window.innerWidth / 2);
          setOpen((value) => !value);
        }}
      >
        <Icon size={16} />
        {text !== undefined && <span className={styles.text}>{text}</span>}
        {text !== undefined && <ChevronDown size={14} className={styles.chevron} />}
      </button>
      {open && (
        <div
          className={align === "end" || (align === "auto" && reversed) ? `${styles.panel} ${styles.panelEnd}` : styles.panel}
          id={panelId}
          role="menu"
        >
          {rows.map(({ item, heading }) => {
            const active = activeIds.includes(item.id);
            return (
              <div key={item.id}>
                {heading !== undefined && <p className={styles.section}>{heading}</p>}
                <button
                  type="button"
                  role="menuitemradio"
                  aria-checked={active}
                  className={active ? `${styles.item} ${styles.active}` : styles.item}
                  onClick={() => {
                    onSelect(item.id);
                    setOpen(false);
                  }}
                >
                  <span>{item.label}</span>
                  {active && <span aria-hidden="true">✓</span>}
                </button>
              </div>
            );
          })}
        </div>
      )}
    </span>
  );
}
