// FilterChips — the status filter row (ui-overhaul-plan.md §6: "the sheet's row is a filter, not navigation" —
// `All · In Progress · Unread` for books, a different set per shelf kind). Data-driven like PillRow: which
// chips exist for a given shelf is a feature concern, not this primitive's.
import styles from "./FilterChips.module.css";

export interface FilterChipItem {
  id: string;
  label: string;
}

export interface FilterChipsProps {
  items: FilterChipItem[];
  activeId?: string;
  onSelect?: (id: string) => void;
}

export function FilterChips({ items, activeId, onSelect }: FilterChipsProps) {
  return (
    <div className={styles.row} role="tablist">
      {items.map((item) => {
        const active = item.id === activeId;
        return (
          <button
            key={item.id}
            type="button"
            role="tab"
            aria-selected={active}
            className={active ? `${styles.chip} ${styles.active}` : styles.chip}
            onClick={() => onSelect?.(item.id)}
          >
            {item.label}
          </button>
        );
      })}
    </div>
  );
}
