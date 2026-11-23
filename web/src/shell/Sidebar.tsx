// Sidebar — the desktop nav's five places, vertical (plan §4.1). Same places/order as Dock — the two must never
// drift, so both read navPlaces() from the same shell/places.ts rather than keeping their own lists.
import { Link, useRouterState } from "@tanstack/react-router";
import { navPlaces } from "./places";
import styles from "./Sidebar.module.css";

export function Sidebar({ hermes = false }: { hermes?: boolean }) {
  const pathname = useRouterState({ select: (state) => state.location.pathname });

  return (
    <nav className={styles.sidebar} aria-label="Primary">
      {navPlaces({ hermes }).map((place) => {
        const Icon = place.icon;
        const active = pathname.startsWith(place.to);
        return (
          <Link key={place.id} to={place.to} className={active ? `${styles.place} ${styles.active}` : styles.place}>
            <Icon size={20} />
            <span>{place.label}</span>
          </Link>
        );
      })}
    </nav>
  );
}
