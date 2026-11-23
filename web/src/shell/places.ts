// shell/places.ts — the dock's/sidebar's five places (ui-overhaul-plan.md: "Dock. Five places, icon over label,
// flat, no top border"), read verbatim from the legacy internal/web/static/dock.js's PLACES table so Phase 2's
// real routes land on names and an order Kevin already knows, not a redesign. Lyana is conditional there too:
// without the module the dock/sidebar stay four places and never advertise something that isn't there.
import type { ComponentType } from "react";
import { BookOpen, Compass, NotebookText, Settings, Sparkles } from "lucide-react";

export interface NavPlace {
  id: "library" | "discover" | "hermes" | "notes" | "settings";
  label: string;
  to: string;
  icon: ComponentType<{ size?: number; strokeWidth?: number }>;
}

const PLACES: NavPlace[] = [
  { id: "library", label: "Library", to: "/library", icon: BookOpen },
  { id: "discover", label: "Discover", to: "/discover", icon: Compass },
  { id: "hermes", label: "AI", to: "/lyana", icon: Sparkles },
  { id: "notes", label: "Notes", to: "/notes", icon: NotebookText },
  { id: "settings", label: "Settings", to: "/settings", icon: Settings },
];

/** Mirrors dockPlaces({ hermes }) from the legacy dock.js — Lyana only shows once the module/a chat page exists. */
export function navPlaces({ hermes = false }: { hermes?: boolean } = {}): NavPlace[] {
  return PLACES.filter((place) => place.id !== "hermes" || hermes);
}
