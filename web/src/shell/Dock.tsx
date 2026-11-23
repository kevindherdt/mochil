// Dock — the phone nav's five places (ui-overhaul-plan.md: icon over label, flat, no top border). Active place
// highlights in the accent colour; the mini player sits in AppShell's separate `player` slot above this, shown
// only while playing — never owned by the dock itself.
import { Link, useRouterState } from "@tanstack/react-router";
import { navPlaces } from "./places";
import styles from "./Dock.module.css";

export function Dock({ hermes = false }: { hermes?: boolean }) {
  const pathname = useRouterState({ select: (state) => state.location.pathname });

  return (
    <nav className={styles.dock} aria-label="Primary">
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
