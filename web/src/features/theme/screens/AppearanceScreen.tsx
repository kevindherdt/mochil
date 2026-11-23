// features/theme/screens/AppearanceScreen.tsx — Settings → Appearance: the "no user-facing picker UI, just the
// one default theme applied" gap flagged in Phase 1 (docs/rewrite-progress.md). Two independent choices, same
// shape as the dev-only /__gallery switcher but durable (features/theme/ThemeContext persists the pick) and
// built from real primitives instead of plain <Button> rows: which of the app's four palettes, and dark vs
// light mode (both apply to all four palettes equally — including e-ink, which has its own light and dark
// variants in tokens.css, not just a single "print" look).
import { canonicalThemes, themeLabels } from "../../../lib/tokenMeta";
import { Page, Stack, Heading, Text, PillRow } from "../../../ui";
import { useThemeContext } from "../ThemeContext";

const MODE_ITEMS = [
  { id: "dark", label: "Dark" },
  { id: "light", label: "Light" },
];

export function AppearanceScreen() {
  const { palette, mode, setPalette, setMode } = useThemeContext();

  return (
    <Page>
      <Stack gap={24}>
        <Heading level={1}>Appearance</Heading>

        <Stack gap={8}>
          <Text size="13" muted>Theme</Text>
          <PillRow
            items={canonicalThemes.map((id) => ({ id, label: themeLabels[id] }))}
            activeId={palette}
            onSelect={(id) => setPalette(id as (typeof canonicalThemes)[number])}
          />
        </Stack>

        <Stack gap={8}>
          <Text size="13" muted>Mode</Text>
          <PillRow items={MODE_ITEMS} activeId={mode} onSelect={(id) => setMode(id as "dark" | "light")} />
        </Stack>
      </Stack>
    </Page>
  );
}
