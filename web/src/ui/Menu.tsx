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
  /** The trigger's glyph. It has no visible text, so `label` is required as its accessible name. */
  icon: ComponentType<{ size?: number }>;
  label: string;
}

export function Menu({ items, activeIds = [], onSelect, icon: Icon, label }: MenuProps) {
  const [open, setOpen] = useState(false);
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
        onClick={() => setOpen((value) => !value)}
      >
        <Icon size={16} />
      </button>
      {open && (
        <div className={styles.panel} id={panelId} role="menu">
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
