// features/settings/screens/SettingsScreen.tsx — the Settings hub. Was a StubScreen with raw <Link>s; the app now
// has enough panels (Appearance, Modules, About) that a list of them is the screen, and the same list is where the
// remaining Phase 2 panels land. Each row is a plain Link so it works with the browser's back button, middle-click
// and keyboard focus without re-inventing navigation.
import { Link } from "@tanstack/react-router";
import { ChevronRight, Palette, Puzzle, Info, Users, LibraryBig, MonitorPlay, Activity, Sparkles, RefreshCw, Volume2, ListChecks } from "lucide-react";
import { Page, Stack, Heading, Text, Card, GridList } from "../../../ui";

const PANELS = [
  { to: "/settings/appearance", label: "Appearance", description: "Theme and light or dark mode.", icon: Palette },
  { to: "/settings/libraries", label: "Libraries", description: "Folders of books, comics, audio and video.", icon: LibraryBig },
  { to: "/settings/playback", label: "Playback", description: "Streaming quality and default languages.", icon: MonitorPlay },
  { to: "/settings/metadata", label: "Metadata", description: "Hardcover and TMDB tokens for covers and details.", icon: Sparkles },
  { to: "/settings/sync", label: "Reading sync", description: "Progress between KOReader and this server.", icon: RefreshCw },
  { to: "/settings/narration", label: "Narration", description: "The voice that reads books aloud.", icon: Volume2 },
  { to: "/settings/modules", label: "Modules", description: "Turn optional features on or off.", icon: Puzzle },
  { to: "/settings/accounts", label: "Accounts", description: "Everyone who can sign in, and their role.", icon: Users },
  { to: "/settings/activity", label: "Activity", description: "Scans, downloads and transcoding, live.", icon: ListChecks },
  { to: "/settings/diagnostics", label: "Diagnostics", description: "Uptime, memory and what has been failing.", icon: Activity },
  { to: "/settings/about", label: "About", description: "Build, version and the signed-in account.", icon: Info },
] as const;

export function SettingsScreen() {
  return (
    <Page>
      <Stack gap={24}>
        <Heading level={1}>Settings</Heading>
        <GridList>
          {PANELS.map(({ to, label, description, icon: Icon }) => (
            <Link key={to} to={to} style={{ textDecoration: "none", color: "inherit" }}>
              <Card>
                <div style={{ display: "flex", alignItems: "center", gap: "var(--space-12)" }}>
                  <Icon size={18} aria-hidden="true" />
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <Stack gap={4}>
                      <Text as="span" size="14">{label}</Text>
                      <Text size="12" muted>{description}</Text>
                    </Stack>
                  </div>
                  <ChevronRight size={16} aria-hidden="true" />
                </div>
              </Card>
            </Link>
          ))}
        </GridList>
        {/* This line used to say accounts/libraries/metadata/playback/narration "are still to come" — every one of
            those panels has since shipped, so it was telling a visitor the app was missing what it was showing.
            What is genuinely not here: the reader's own settings, which live inside the reader. */}
        <Text size="11" muted>
          Reading settings (type, theme, narration while reading) live inside the reader itself, not here.
        </Text>
      </Stack>
    </Page>
  );
}
