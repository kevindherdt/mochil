// /__gallery — the design system contract (plan §4.4). Renders every primitive at both viewports in every
// theme; Playwright screenshots this page so a token or primitive edit shows its diff on every surface at once,
// and Kevin approves the diff before it ships. Every primitive is added here in the same commit that builds it.
import { useState } from "react";
import { canonicalThemes, fontSizeKeys, radiusKeys, spaceKeys, themeLabels } from "../lib/tokenMeta";
import { useTheme } from "../lib/useTheme";
import { Page, Stack, Inline, Text, Heading, Button, Card, Skeleton, EmptyState, ErrorState, Async, MediaGrid, BookCard, VideoCard, MusicCard, TextField, PillRow, FilterChips, Menu, GroupCard, Switch, type AsyncQueryLike } from "../ui";
import { BookOpen, Headphones, Clapperboard, Music2, BookImage, MoreHorizontal, Layers, UserRound, LayoutList } from "lucide-react";

/** A deterministic, offline placeholder cover — no network dependency, no asset pipeline needed yet, same output
    every render so Playwright screenshots of /__gallery don't flake on a CDN. Real screens read real cover URLs
    from the API once that's wired (Phase 2). */
function placeholderCover(label: string, hue: number): string {
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="240" height="360">
    <rect width="240" height="360" fill="hsl(${hue} 45% 28%)" />
    <text x="120" y="190" font-family="sans-serif" font-size="22" fill="hsl(${hue} 60% 85%)" text-anchor="middle">${label}</text>
  </svg>`;
  return `data:image/svg+xml;utf8,${encodeURIComponent(svg)}`;
}

function Swatch({ role }: { role: string }) {
  const kebab = role.replace(/[A-Z]/g, (ch) => "-" + ch.toLowerCase());
  return (
    <Stack gap={4} align="center">
      <div
        style={{
          width: 48,
          height: 48,
          borderRadius: "var(--radius-8)",
          border: "1px solid var(--color-line)",
          background: `var(--color-${kebab})`,
        }}
      />
      <Text size="11" muted>{role}</Text>
    </Stack>
  );
}

const ROLES = [
  "bg", "surface", "raised", "line", "text", "muted", "accent", "accentInk", "wash", "error", "success",
  "accentVideo", "accentMusic", "accentPodcast", "accentComics",
];

/** A hand-built AsyncQueryLike, so the gallery can show <Async>'s loading/error/empty/loaded states on demand
    without a real network request (plan §4.4: the gallery proves the primitive, not a live data source). */
function demoQuery(state: "loading" | "error" | "empty" | "loaded"): AsyncQueryLike<string[]> {
  if (state === "loading") return { data: undefined, isLoading: true, isError: false };
  if (state === "error") return { data: undefined, isLoading: false, isError: true, refetch: () => {} };
  if (state === "empty") return { data: [], isLoading: false, isError: false };
  return { data: ["The Bellwether Line", "A Quiet Harbor"], isLoading: false, isError: false };
}

export function GalleryPage() {
  const { palette, mode, setPalette, setMode } = useTheme();
  const [asyncState, setAsyncState] = useState<"loading" | "error" | "empty" | "loaded">("loading");
  const [activePill, setActivePill] = useState("books");
  const [activeChip, setActiveChip] = useState("all");
  const [menuView, setMenuView] = useState("books");
  const [switchOn, setSwitchOn] = useState(true);

  const VIEW_OPTIONS = [
    { id: "books", label: "All books" },
    { id: "series", label: "Series" },
    { id: "authors", label: "Authors" },
  ];

  const pillItems = [
    { id: "books", label: "Books", icon: BookOpen },
    { id: "audiobooks", label: "Audiobooks", icon: Headphones },
    { id: "video", label: "TV & Movies", icon: Clapperboard },
    { id: "music", label: "Music", icon: Music2 },
    { id: "manga", label: "Manga", icon: BookImage },
  ];
  const chipItems = [
    { id: "all", label: "All" },
    { id: "in_progress", label: "In Progress" },
    { id: "unread", label: "Unread" },
  ];

  return (
    <div style={{ minHeight: "100vh", background: "var(--color-bg)" }}>
      <Page>
        <Stack gap={24}>
          <Heading level={1} size="26">Gallery</Heading>
          <Text muted>
            Every primitive, every theme, both viewports (plan §4.4). This is the living contract — a design
            change shows its diff here before it ships anywhere else.
          </Text>

          <Inline gap={8} wrap>
            {canonicalThemes.map((id) => (
              <Button key={id} variant={id === palette ? "primary" : "secondary"} onClick={() => setPalette(id)}>
                {themeLabels[id]}
              </Button>
            ))}
            <Button variant="secondary" onClick={() => setMode(mode === "dark" ? "light" : "dark")}>{mode}</Button>
          </Inline>

          <section data-gallery-section="colors">
            <Heading size="18">Colour roles</Heading>
            <Inline gap={16} wrap>
              {ROLES.map((role) => (
                <Swatch key={role} role={role} />
              ))}
            </Inline>
          </section>

          <section data-gallery-section="scale">
            <Heading size="18">Type scale</Heading>
            <Stack gap={4}>
              {fontSizeKeys.map((size) => (
                <Text key={size} size={size}>{size}px — The quick brown fox jumps over the lazy dog</Text>
              ))}
            </Stack>
          </section>

          <section data-gallery-section="radius">
            <Heading size="18">Radius scale</Heading>
            <Inline gap={12} wrap>
              {radiusKeys.map((r) => (
                <div
                  key={r}
                  style={{
                    width: 64,
                    height: 64,
                    background: "var(--color-surface)",
                    border: "1px solid var(--color-line)",
                    borderRadius: `var(--radius-${r})`,
                    display: "flex",
                    alignItems: "center",
                    justifyContent: "center",
                  }}
                >
                  <Text size="11" muted>{r}</Text>
                </div>
              ))}
            </Inline>
          </section>

          <section data-gallery-section="space">
            <Heading size="18">Space scale</Heading>
            <Stack gap={8}>
              {spaceKeys.map((s) => (
                <Inline key={s} gap={8}>
                  <Text size="11" muted>{s}</Text>
                  <div style={{ background: "var(--color-accent)", height: 12, width: `var(--space-${s})` }} />
                </Inline>
              ))}
            </Stack>
          </section>

          <section data-gallery-section="buttons">
            <Heading size="18">Button</Heading>
            <Inline gap={8}>
              <Button variant="primary">Primary</Button>
              <Button variant="secondary">Secondary</Button>
              <Button variant="primary" disabled>Disabled</Button>
            </Inline>
          </section>

          <section data-gallery-section="textfield">
            <Heading size="18">TextField</Heading>
            <div style={{ maxWidth: "280px" }}>
              <Stack gap={12}>
                <TextField label="Username" autoComplete="username" />
                <TextField label="Password" type="password" autoComplete="current-password" />
              </Stack>
            </div>
          </section>

          <section data-gallery-section="pillrow">
            <Heading size="18">PillRow — library switcher (ui-overhaul-plan.md §4)</Heading>
            <PillRow
              items={pillItems}
              activeId={activePill}
              onSelect={setActivePill}
              trailing={
                <button
                  type="button"
                  aria-label="More libraries"
                  style={{
                    display: "inline-flex",
                    alignItems: "center",
                    justifyContent: "center",
                    flexShrink: 0,
                    width: 34,
                    height: 34,
                    border: "none",
                    borderRadius: "var(--radius-pill)",
                    background: "var(--color-raised)",
                    color: "var(--color-muted)",
                  }}
                >
                  <MoreHorizontal size={16} />
                </button>
              }
            />
          </section>

          <section data-gallery-section="filterchips">
            <Heading size="18">FilterChips — status filter (ui-overhaul-plan.md §6)</Heading>
            <FilterChips items={chipItems} activeId={activeChip} onSelect={setActiveChip} />
          </section>

          <section data-gallery-section="select">
            <Heading size="18">Menu — compact icon trigger for browse-by (pinned at the pill row's end)</Heading>
            <PillRow
              items={pillItems}
              activeId={activePill}
              onSelect={setActivePill}
              trailing={<Menu label="Browse by" icon={LayoutList} items={VIEW_OPTIONS} activeIds={[menuView]} onSelect={setMenuView} />}
            />
          </section>

          <section data-gallery-section="switch">
            <Heading size="18">Switch — Settings → Modules (one switch per module, options inside a card)</Heading>
            <Inline gap={16} wrap>
              <Switch checked={switchOn} onChange={setSwitchOn} label="Example module" />
              <Switch checked={false} onChange={() => {}} label="Example module, off" />
              <Switch checked onChange={() => {}} label="Example module, disabled while saving" disabled />
            </Inline>
          </section>

          <section data-gallery-section="groupcard">
            <Heading size="18">GroupCard — Series / Authors browse lists</Heading>
            <MediaGrid>
              <GroupCard name="Ancient Egypt" count={7} coverUrl={placeholderCover("Ancient Egypt", 35)} kindIcon={Layers} />
              <GroupCard name="Arcane Ascension" count={3} coverUrl={placeholderCover("Arcane Ascension", 265)} kindIcon={Layers} />
              <GroupCard name="Adam Nevill" count={1} coverUrl={placeholderCover("Adam Nevill", 130)} kindIcon={UserRound} />
            </MediaGrid>
          </section>

          <section data-gallery-section="card">
            <Heading size="18">Card</Heading>
            <Card>
              <Text>A card is the surface container — background, hairline border, radius and a shadow step,
                all from tokens.</Text>
            </Card>
          </section>

          <section data-gallery-section="skeleton">
            <Heading size="18">Skeleton</Heading>
            <Stack gap={8}>
              <Skeleton />
              <Skeleton width="60%" />
            </Stack>
          </section>

          <section data-gallery-section="states">
            <Heading size="18">Empty / error state</Heading>
            <Stack gap={12}>
              <EmptyState message="No books in this library yet." action={{ label: "Add a library", onClick: () => {} }} />
              <ErrorState onRetry={() => {}} />
            </Stack>
          </section>

          <section data-gallery-section="async">
            <Heading size="18">Async (loading / error / empty / loaded)</Heading>
            <Stack gap={8}>
              <Inline gap={8}>
                {(["loading", "error", "empty", "loaded"] as const).map((s) => (
                  <Button key={s} variant={s === asyncState ? "primary" : "secondary"} onClick={() => setAsyncState(s)}>
                    {s}
                  </Button>
                ))}
              </Inline>
              <Card>
                <Async query={demoQuery(asyncState)} empty={(items) => items.length === 0}>
                  {(items) => (
                    <Stack gap={4}>
                      {items.map((title) => <Text key={title}>{title}</Text>)}
                    </Stack>
                  )}
                </Async>
              </Card>
            </Stack>
          </section>

          <section data-gallery-section="media-grid-books">
            <Heading size="18">MediaGrid — BookCard (shared by books/audiobooks/light novels/Storyteller)</Heading>
            <Text size="12" muted>No tag pill — the kind is a glyph on the cover (Kevin, 22 Nov).</Text>
            <MediaGrid>
              <BookCard
                title="Neuromancer"
                author="William Gibson"
                coverUrl={placeholderCover("Neuromancer", 200)}
                kindIcon={BookOpen}
                progress={0.72}
                hasAudio
                saved
                onToggleSave={() => {}}
              />
              <BookCard
                title="Count Zero"
                author="William Gibson"
                coverUrl={placeholderCover("Count Zero", 280)}
                kindIcon={Headphones}
                onToggleSave={() => {}}
              />
              <BookCard
                title="Mona Lisa Overdrive"
                author="William Gibson"
                coverUrl={placeholderCover("Mona Lisa Overdrive", 20)}
                kindIcon={BookOpen}
                finished
                onToggleSave={() => {}}
              />
              <BookCard
                title="7th Time Loop: The Villainess Enjoys a Carefree Married Life"
                author="Touko Amekawa"
                coverUrl={placeholderCover("7th Time Loop", 320)}
                kindIcon={BookImage}
                seriesIndex={1}
                onToggleSave={() => {}}
              />
            </MediaGrid>
          </section>

          <section data-gallery-section="media-grid-video">
            <Heading size="18">MediaGrid — VideoCard (shared by Movies/TV)</Heading>
            <MediaGrid>
              <VideoCard
                title="Blade Runner 2049"
                meta="Movie · Sci-Fi"
                coverUrl={placeholderCover("Blade Runner 2049", 15)}
                formatLabel="4K HDR"
                onToggleSave={() => {}}
              />
              <VideoCard
                title="The Matrix"
                meta="Movie · Sci-Fi"
                coverUrl={placeholderCover("The Matrix", 120)}
                formatLabel="4K HDR · Direct Play"
                progress={0.35}
                saved
                onToggleSave={() => {}}
              />
              <VideoCard
                title="Cyberpunk: Edgerunners"
                meta="TV Series · 1 Season"
                coverUrl={placeholderCover("Edgerunners", 70)}
                formatLabel="4K HDR"
                onToggleSave={() => {}}
              />
            </MediaGrid>
          </section>

          <section data-gallery-section="media-grid-music">
            <Heading size="18">MediaGrid — MusicCard (deliberately different: square, no progress chrome)</Heading>
            <MediaGrid>
              <MusicCard title="Oh!" artist="Girls' Generation" coverUrl={placeholderCover("Oh!", 330)} formatLabel="FLAC" kindIcon={Music2} />
              <MusicCard title="Mega Drive" artist="Lazerhawk" coverUrl={placeholderCover("Mega Drive", 290)} kindIcon={Music2} />
              <MusicCard title="Kavinsky" artist="Kavinsky" coverUrl={placeholderCover("Kavinsky", 0)} formatLabel="FLAC" kindIcon={Music2} />
            </MediaGrid>
          </section>
        </Stack>
      </Page>
    </div>
  );
}
