// Dock — the phone nav's five places (ui-overhaul-plan.md: icon over label, flat, no top border). Active place
// highlights in the accent colour; the mini player sits in AppShell's separate `player` slot above this, shown
// only while playing — never owned by the dock itself.
import { Link, useRouterState } from "@tanstack/react-router";
import { navPlaces } from "./places";
import styles from "./Dock.module.css";

export function Dock({ hermes = false }: { hermes?: boolean }) {
  const pathname = useRouterState({ select: (state) => state.location.pathname });

  return (
    // data-dock is the contract the layout gate uses: the class name is hashed by the CSS module, so a selector on it
    // cannot survive a rename, and the gate silently stopped finding the dock when the shell was ported.
    <nav className={styles.dock} data-dock="" aria-label="Primary">
      {/* No editor here: the phone's dock offers what works on a phone, and the editor is a desktop spreadsheet. */}
      {navPlaces({ hermes }).map((place) => {
        const Icon = place.icon;
        const active = pathname.startsWith(place.to);
        return (
          <Link key={place.id} to={place.to} className={active ? `${styles.place} ${styles.active}` : styles.place}>
            <Icon size={17} strokeWidth={1.8} />
            <span>{place.label}</span>
          </Link>
        );
      })}
    </nav>
  );
}
