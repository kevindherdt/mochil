// shell/places.ts — the dock's/sidebar's five places (ui-overhaul-plan.md: "Dock. Five places, icon over label,
// flat, no top border"), read verbatim from the legacy internal/web/static/dock.js's PLACES table so Phase 2's
// real routes land on names and an order Kevin already knows, not a redesign. Lyana is conditional there too:
// without the module the dock/sidebar stay four places and never advertise something that isn't there.
import type { ComponentType } from "react";
import { BookOpen, Compass, NotebookText, Settings, Sparkles, Table2 } from "lucide-react";

export interface NavPlace {
  id: "library" | "discover" | "editor" | "hermes" | "notes" | "settings";
  label: string;
  to: string;
  icon: ComponentType<{ size?: number; strokeWidth?: number }>;
}

const PLACES: NavPlace[] = [
  { id: "library", label: "Library", to: "/library", icon: BookOpen },
  { id: "discover", label: "Discover", to: "/discover", icon: Compass },
  { id: "editor", label: "Editor", to: "/editor", icon: Table2 },
  { id: "hermes", label: "AI", to: "/lyana", icon: Sparkles },
  { id: "notes", label: "Notes", to: "/notes", icon: NotebookText },
  { id: "settings", label: "Settings", to: "/settings", icon: Settings },
];

/**
 * Mirrors dockPlaces({ hermes }) from the legacy dock.js — Lyana only shows once the module/a chat page exists.
 *
 * The editor is a desktop place: it is a spreadsheet with resizable columns and a keyboard cursor, and it is not usable
 * on a phone. So it joins the sidebar and not the dock, and the caller says which it is rendering.
 */
export function navPlaces({ hermes = false, editor = false }: { hermes?: boolean; editor?: boolean } = {}): NavPlace[] {
  return PLACES.filter((place) => {
    if (place.id === "hermes") return hermes;
    if (place.id === "editor") return editor;
    return true;
  });
}
