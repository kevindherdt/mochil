// features/library/screens/LibraryScreen.tsx — the real Library screen (ui-overhaul-plan.md "Library hub"/§4).
//
// Every piece of "what am I looking at" (shelf, browse mode, drilled-in group, grid/list) lives in the URL's
// search params, not in useState: a useState resets to its default on every mount, which is what made the
// phone's back button land back on Books after opening a track from Music (Kevin, 22 Nov: "clicking to play a
// song and pressing the back button opens the book li[b]rary instead of the music library i was at"). The
// browser's own history now carries this state, so back/forward restores it for free — see router.tsx's
// `librarySearchSchema`.
//
// Browse modes, by shelf kind:
//   catalogue (books/manga/comics/…) → All books | Series | Authors        (existing, /api/series, /api/authors)
//   music                            → All tracks | Albums | Artists      (/api/audio-groups, group_type=albums|artists)
//   audiobooks                       → All audiobooks | Series | Authors  (/api/audio-groups, group_type=series|authors)
//   video / podcasts                 → no browse modes, layout only       (nothing to group by yet)
import { useEffect, useState } from "react";
import { useNavigate, useSearch } from "@tanstack/react-router";
import { Page, Stack, Inline, Text, Button, PillRow, MediaGrid, MediaList, BookCard, GroupCard, Menu, IconButton, SearchField, FilterChips, Async } from "../../../ui";
import { ScreenHeader } from "../../../shell/ScreenHeader";
import { Layers, Search, Sparkles, UserRound, X } from "lucide-react";
import { useLibraries } from "../useLibraries";
import { useBooks } from "../useBooks";
import { MediaShelf } from "../MediaShelf";
import { useGroups } from "../useGroups";
import { AudioGroupList } from "../AudioGroupList";
import { AudioGroupTracks } from "../AudioGroupTracks";
import { PodcastShows } from "../PodcastShows";
import { CollectionBooks, CollectionsList } from "../CollectionsView";
import type { AudioGroupKind } from "../api/audioGroups";
import { useAiEnabled } from "../../../lib/useModules";
import { kindMeta, libraryIcon, cardFamily, isShelvable, isNoteShelf, shelfSource, unbuiltReason } from "../kinds";
import { displayTitle, displaySeriesIndex } from "../titles";
import type { BookInfo } from "../api/books";

type StatusFilter = "all" | "in_progress" | "unread" | "read";

/** The sheet's chip row, as the plan states it: a filter over the shelf, with Audio Synced / Light Novels left out. */
const STATUS_CHIPS = [
  { id: "all", label: "All" },
  { id: "in_progress", label: "In Progress" },
  { id: "unread", label: "Unread" },
];

/** Podcasts have a listening state, not a reading one — "In Progress/Unread" would be the wrong words for an
 *  episode, and "Played" is the one that matters when a show has years of back catalogue. All four map onto the
 *  server's own `played=` parameter (internal/web/audio_http.go's matchesPlayedFilter). */
const PODCAST_STATUS_CHIPS = [
  { id: "all", label: "All" },
  { id: "continue", label: "Playing" },
  { id: "unplayed", label: "Unplayed" },
  { id: "played", label: "Played" },
];

/** Ordering for audio shelves. "Newest"/"Oldest" are by publish/modified date (sortAudioItems' recent/oldest),
 *  which is the only sane way into a show with a thousand episodes: oldest first is how you start at the beginning. */
const AUDIO_SORT_ITEMS = [
  { id: "sort-title", label: "Title", section: "Order" },
  { id: "sort-recent", label: "Newest first", section: "Order" },
  { id: "sort-oldest", label: "Oldest first", section: "Order" },
];

const BOOK_VIEW_MENU_ITEMS = [
  { id: "books", label: "All books", section: "Show" },
  { id: "series", label: "Series", section: "Show" },
  { id: "authors", label: "Authors", section: "Show" },
  { id: "collections", label: "Collections", section: "Show" },
  { id: "grid", label: "Covers", section: "Layout" },
  { id: "list", label: "List", section: "Layout" },
];

const MUSIC_VIEW_MENU_ITEMS = [
  { id: "tracks", label: "All tracks", section: "Show" },
  { id: "albums", label: "Albums", section: "Show" },
  { id: "artists", label: "Artists", section: "Show" },
  { id: "grid", label: "Covers", section: "Layout" },
  { id: "list", label: "List", section: "Layout" },
];

const AUDIOBOOK_VIEW_MENU_ITEMS = [
  { id: "tracks", label: "All audiobooks", section: "Show" },
  { id: "series", label: "Series", section: "Show" },
  { id: "authors", label: "Authors", section: "Show" },
  { id: "grid", label: "Covers", section: "Layout" },
  { id: "list", label: "List", section: "Layout" },
];

/** A podcast shelf has one real browse mode (Kevin: "podcast series stacking") plus the same layout and order
 *  controls every audio shelf has. "tracks" is kept as the id so it matches every other audio shelf's flat view. */
const PODCAST_VIEW_MENU_ITEMS = [
  { id: "tracks", label: "All episodes", section: "Show" },
  { id: "shows", label: "Shows", section: "Show" },
  { id: "grid", label: "Covers", section: "Layout" },
  { id: "list", label: "List", section: "Layout" },
];

const LAYOUT_ONLY_MENU_ITEMS = [
  { id: "grid", label: "Covers", section: "Layout" },
  { id: "list", label: "List", section: "Layout" },
];

const LAST_SHELF_KEY = "mochil:last-shelf";
const LIBRARY_VIEW_KEY = "mochil:library-view";

/** plan §5.1 "last library shelf": a visitor who closes the tab and comes back should land where they left off,
    not always on Books. `librarySearchSchema`'s URL state already answers the in-session back button; this is the
    across-visits case, so it only needs to act once, before anything in the URL has a say — a fresh `/library`
    load with no search params at all. */
function readLastShelf(): { lib: string; view?: string; layout?: "grid" | "list" } | undefined {
  try {
    const raw = window.localStorage.getItem(LAST_SHELF_KEY);
    return raw ? (JSON.parse(raw) as { lib: string; view?: string; layout?: "grid" | "list" }) : undefined;
  } catch {
    return undefined;
  }
}

/** Per-library "stack by" memory (Kevin, 22 Nov 2026: switching from Books->Series back to Light Novels shouldn't
    reset Light Novels back to the flat list every time — each shelf keeps its own browse mode and layout, not just
    whichever shelf happened to be visited last). Keyed separately from LAST_SHELF_KEY, which only ever answers
    "which single shelf was open across visits" for the very first landing; this answers "what did THIS shelf show
    last time", for every later switch within the session too. */
function readLibraryView(libraryId: string): { view?: string; layout?: "grid" | "list" } | undefined {
  try {
    const raw = window.localStorage.getItem(LIBRARY_VIEW_KEY);
    const map = raw ? (JSON.parse(raw) as Record<string, { view?: string; layout?: "grid" | "list" }>) : {};
    return map[libraryId];
  } catch {
    return undefined;
  }
}

function writeLibraryView(libraryId: string, view: string, layout: "grid" | "list"): void {
  try {
    const raw = window.localStorage.getItem(LIBRARY_VIEW_KEY);
    const map = raw ? (JSON.parse(raw) as Record<string, { view?: string; layout?: "grid" | "list" }>) : {};
    map[libraryId] = { view, layout };
    window.localStorage.setItem(LIBRARY_VIEW_KEY, JSON.stringify(map));
  } catch {
    // Storage can be unavailable (private browsing, quota) — remembering the shelf is a convenience, not a
    // requirement, so a write failure here must never break browsing.
  }
}

export function LibraryScreen() {
  const librariesQuery = useLibraries();
  const navigate = useNavigate();
  const search = useSearch({ strict: false }) as { lib?: string; view?: string; layout?: "grid" | "list"; group?: string; groupName?: string };
  const layout = search.layout ?? "grid";

  // One place to change the URL, so every setter below reads the same way: merge, don't replace, so a change to
  // one field (say, layout) never silently drops another (say, which library) that wasn't touched.
  //
  // `replace: true`: switching shelves/views/layout/drilling into a group is lateral movement within one screen,
  // not a new page — it must not push a history entry. Kevin, 22 Nov: "clicking back from the music pushes u
  // straight into the book library still" — before this, every pill click pushed, so the back button undid only
  // the single most recent switch rather than leaving the screen, landing on whatever shelf happened to be
  // selected one click before the player was opened. Only navigating to a genuinely different screen (a detail
  // page, Discover, …) should push, and those routes are reached through `navigate()` elsewhere, not here.
  function updateSearch(patch: Partial<{ lib: string; view: string | undefined; layout: "grid" | "list"; group: string | undefined; groupName: string | undefined }>) {
    void navigate({ to: "/library", search: (previous) => ({ ...previous, ...patch }), replace: true });
  }

  // searchOpen/searchText are the one piece of this screen's state that stays local: a search box mid-type isn't
  // a navigation the back button should ever need to undo, and debouncing a URL write on every keystroke would
  // mean every letter typed also replaces history.
  const [searchOpen, setSearchOpen] = useState(false);
  const [searchText, setSearchText] = useState("");
  const [searchQuery, setSearchQuery] = useState("");
  const [status, setStatus] = useState<StatusFilter>("all");
  // Podcast listening state and audio ordering (kevin, 22 Nov: "playing, unplayed, all pills ... and by title,
  // newest and oldest ... so u can start from the first ones"). Local state like `status`, deliberately not in the
  // URL: both are filters over one shelf, not a place the back button should ever need to return to.
  const [podcastStatus, setPodcastStatus] = useState<"all" | "continue" | "unplayed" | "played">("all");
  const [audioSort, setAudioSort] = useState<"title" | "recent" | "oldest">("title");
  const aiEnabled = useAiEnabled();

  // Debounced: the server's ?q= is a real query, so typing shouldn't fire one request per keystroke.
  useEffect(() => {
    const timer = setTimeout(() => setSearchQuery(searchText.trim()), 250);
    return () => clearTimeout(timer);
  }, [searchText]);

  // The switcher lists shelves you can actually browse. This server's configured libraries start with a notes
  // vault (48 items), and landing there made the app open on "isn't built yet" — Kevin: "not sure if landing on
  // notes by default is the best tho i would much prefer libraries and books as first". So a notes shelf never
  // gets a pill at all, and the default selection is the first renderable shelf, which here is Books.
  const libraries = (librariesQuery.data ?? []).filter((library) => !isNoteShelf(library.kind));
  const firstRenderable = libraries.find((library) => isShelvable(library.kind));
  const activeId = search.lib ?? firstRenderable?.id ?? libraries[0]?.id;
  const activeLibrary = libraries.find((library) => library.id === activeId);
  const family = activeLibrary === undefined ? "none" : cardFamily(activeLibrary.kind);
  const activeIsShelvable = family !== "none";

  const catalogueShelf = activeLibrary !== undefined && shelfSource(activeLibrary.kind) === "catalogue";
  // Music and audiobooks can be browsed by album/artist or series/author, the audio equivalent of the catalogue's
  // Series/Authors (/api/audio-groups — internal/web/android_audio.go's androidAudioGroupingAllowed is the one
  // place that says which group_type values a library kind accepts). Podcasts and video have no grouping yet.
  const audioGroupable = activeLibrary?.kind === "music" || activeLibrary?.kind === "audiobooks";
  const isPodcasts = activeLibrary?.kind.toLowerCase() === "podcasts";
  const isAudioShelf = activeLibrary !== undefined && shelfSource(activeLibrary.kind) === "audio";

  const view = search.view ?? (catalogueShelf ? "books" : "tracks");
  // Drilled into one series/author: the *group list* is no longer what this screen is, its books are. This has to
  // short-circuit `grouped`, because `grouped` also drives the render branch — with it still true, opening "Ancient
  // Egypt" drew the whole series list again under an "← All series" header and the seven books never appeared.
  const insideCatalogueGroup = catalogueShelf && search.group !== undefined;
  // Collections are browsed like Series/Authors but do not come from /api/series or /api/authors — they have their
  // own endpoints and their own drill-down, so they branch before `grouped` picks the series/author query.
  const collectionsView = catalogueShelf && view === "collections";
  const grouped = view !== "books" && catalogueShelf && !insideCatalogueGroup && !collectionsView;
  const audioGroupType: AudioGroupKind | undefined = !audioGroupable
    ? undefined
    : activeLibrary?.kind === "music"
      ? view === "albums"
        ? "albums"
        : view === "artists"
          ? "artists"
          : undefined
      : view === "series"
        ? "series"
        : view === "authors"
          ? "authors"
          : undefined;

  /** What this shelf calls its items. A music shelf with "No books in this shelf yet." was telling a visible lie
      about what the reader was looking at — the card family is already known here, so the sentence uses it. */
  const noun =
    family === "music"
      ? "albums"
      : family === "video"
        ? "titles"
        : activeLibrary?.kind.toLowerCase() === "audiobooks"
          ? "audiobooks"
          : "books";

  // Fresh landing (no search params at all, e.g. a new tab/visit): redirect once to the remembered shelf instead
  // of always defaulting to Books. Only fires when `lib` is genuinely absent — a pill click, the back button, or
  // a shared link all set it explicitly and must win over whatever was remembered. Waits for the real library
  // list so a shelf that was since removed doesn't get redirected to and silently show nothing.
  useEffect(() => {
    if (search.lib !== undefined || librariesQuery.data === undefined) return;
    const remembered = readLastShelf();
    if (remembered === undefined) return;
    if (!libraries.some((library) => library.id === remembered.lib)) return;
    void navigate({ to: "/library", search: (previous) => ({ ...previous, ...remembered }), replace: true });
    // eslint-disable-next-line react-hooks/exhaustive-deps -- intentionally only re-checks on these two signals
  }, [search.lib, librariesQuery.data]);

  // Opening a shelf without saying how to browse it lands on how that shelf was last browsed — the rule selectLibrary
  // already applies to a pill click, extended to every other way in (a sidebar link, a bookmark, the back button).
  // Without this the default wins on arrival and the effect below then saves that default over the memory: the shelf
  // was remembered correctly and overwritten one render later, which is exactly why "Series" never stuck.
  const restorePending =
    search.lib !== undefined && search.view === undefined && readLibraryView(search.lib)?.view !== undefined;
  useEffect(() => {
    if (search.lib === undefined || search.view !== undefined) return;
    if (librariesQuery.data === undefined) return;
    if (!libraries.some((library) => library.id === search.lib)) return;
    const remembered = readLibraryView(search.lib);
    if (remembered?.view === undefined) return;
    void navigate({
      to: "/library",
      search: (previous) => ({ ...previous, view: remembered.view, layout: remembered.layout ?? previous.layout }),
      replace: true,
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps -- re-checks only when the shelf or its view arrives
  }, [search.lib, search.view, librariesQuery.data]);

  // Written back whenever the shelf/view/layout actually settle on something real, so the *next* visit (not this
  // one) remembers it. Not `group`/`groupName` — a drilled-in album is session-local, not "the shelf I was on".
  // Both keys are written: LAST_SHELF_KEY for "which single shelf to land on for a brand-new visit", LIBRARY_VIEW
  // for "what THIS shelf specifically showed", so later switching back to it (same session or a future visit)
  // restores its own view/layout rather than whatever shelf happened to be open last.
  useEffect(() => {
    if (activeId === undefined) return;
    // The restore above is about to say what this shelf shows; saving the default now would overwrite the memory the
    // restore is reading, and the next visit would land flat again.
    if (restorePending) return;
    try {
      window.localStorage.setItem(LAST_SHELF_KEY, JSON.stringify({ lib: activeId, view, layout }));
    } catch {
      // Storage can be unavailable (private browsing, quota) — remembering the shelf is a convenience, not a
      // requirement, so a write failure here must never break browsing.
    }
    writeLibraryView(activeId, view, layout);
  }, [activeId, view, layout, restorePending]);


  const filter = {
    ...(search.group !== undefined && view === "series" ? { series: search.group } : {}),
    ...(search.group !== undefined && view === "authors" ? { author: search.group } : {}),
    ...(searchQuery !== "" ? { q: searchQuery } : {}),
    ...(status !== "all" ? { status: status as "in_progress" | "unread" | "read" } : {}),
  };
  const shelfId = activeLibrary !== undefined && activeIsShelvable ? activeLibrary.id : undefined;
  const books = useBooks(catalogueShelf ? shelfId : undefined, filter);

  /** Inside a series or author, the shelf's own ordering is by title — which scatters "Vol. 2, Vol. 4, Vol. 3, Vol. 1"
   *  across the grid and makes a series unreadable. Volume order is the whole point of the view, so it is applied
   *  here; outside a group nothing is re-ordered. */
  function inVolumeOrder(items: BookInfo[]): BookInfo[] {
    if (!insideCatalogueGroup) return items;
    const key = (item: BookInfo) =>
      `${item.series ?? "\uffff"}\u0000${String(item.series_index ?? 9999).padStart(6, "0")}\u0000${item.title}`;
    return [...items].sort((a, b) => (key(a) < key(b) ? -1 : key(a) > key(b) ? 1 : 0));
  }
  const seriesQuery = useGroups("series", grouped && view === "series" ? shelfId : undefined);
  const authorsQuery = useGroups("authors", grouped && view === "authors" ? shelfId : undefined);
  const groups = view === "authors" ? authorsQuery : seriesQuery;

  const browseMenuItems = !activeIsShelvable || !(catalogueShelf || audioGroupable || isPodcasts)
    ? LAYOUT_ONLY_MENU_ITEMS
    : catalogueShelf
      ? BOOK_VIEW_MENU_ITEMS
      : isPodcasts
        ? PODCAST_VIEW_MENU_ITEMS
        : activeLibrary?.kind === "music"
          ? MUSIC_VIEW_MENU_ITEMS
          : AUDIOBOOK_VIEW_MENU_ITEMS;
  // Ordering is offered where it changes something: every audio shelf, podcasts included (which have no browse
  // mode at all, so before this their menu was layout-only).
  const viewMenuItems = isAudioShelf ? [...browseMenuItems, ...AUDIO_SORT_ITEMS] : browseMenuItems;
  const activeMenuIds = isAudioShelf ? [view, layout, `sort-${audioSort}`] : [view, layout];

  // Switching shelf or view drops any open drill-down: "Ancient Egypt" only means something inside Books → Series.
  // Each shelf keeps its OWN remembered view/layout (readLibraryView), not whichever shelf was visited last overall
  // — switching libraries must not silently reset a shelf someone already set to Series/Authors back to the flat
  // list the moment they look at something else and come back.
  function selectLibrary(id: string) {
    const remembered = readLibraryView(id);
    updateSearch({ lib: id, view: remembered?.view, layout: remembered?.layout ?? layout, group: undefined, groupName: undefined });
  }
  function selectView(next: string) {
    updateSearch({ view: next, group: undefined, groupName: undefined });
  }
  /** One menu, two axes: "books|series|authors|tracks|albums|artists" changes the browse mode, "grid|list" only
      the layout. */
  function selectBrowseOrLayout(id: string) {
    if (id === "grid" || id === "list") {
      updateSearch({ layout: id });
      return;
    }
    if (id.startsWith("sort-")) {
      setAudioSort(id.slice("sort-".length) as "title" | "recent" | "oldest");
      return;
    }
    selectView(id);
  }

  return (
    <>
      <ScreenHeader>
        <Stack gap={8}>
          <Async query={librariesQuery} empty={(items) => items.length === 0} emptyMessage="No libraries configured yet.">
            {() => (
              <PillRow
                items={libraries.map((library) => ({ id: library.id, label: library.name, icon: libraryIcon(library.kind, library.id) }))}
                activeId={activeId}
                onSelect={selectLibrary}
              />
            )}
          </Async>

          {/* The action row under the pills: the reading-status filter on the left, the icon cluster on the right
              (mock-up: search, view, AI at the right edge; the chips row is stated as a filter, not navigation).
              The chips only exist for the plain book list — Series/Authors are groupings, not a reading state, and
              showing a status filter there would promise something the endpoint can't answer. */}
          {activeIsShelvable && (
            <div style={{ display: "flex", alignItems: "center", gap: "var(--space-8)" }}>
              {searchOpen ? (
                <>
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <SearchField value={searchText} onChange={setSearchText} placeholder="Search this shelf" />
                  </div>
                  <IconButton
                    icon={X}
                    label="Close search"
                    onClick={() => {
                      setSearchOpen(false);
                      setSearchText("");
                    }}
                  />
                </>
              ) : (
                <>
                  <div style={{ flex: 1, minWidth: 0 }}>
                    {view === "books" && search.group === undefined && catalogueShelf && (
                      <FilterChips items={STATUS_CHIPS} activeId={status} onSelect={(id) => setStatus(id as StatusFilter)} />
                    )}
                    {/* Inside one show the pills matter most: that is exactly where "hide the ones I already
                        heard" is the difference between a usable back catalogue and a wall of 400 rows. */}
                    {isPodcasts && (
                      <FilterChips
                        items={PODCAST_STATUS_CHIPS}
                        activeId={podcastStatus}
                        onSelect={(id) => setPodcastStatus(id as typeof podcastStatus)}
                      />
                    )}
                  </div>
                  <IconButton icon={Search} label="Search" onClick={() => setSearchOpen(true)} />
                  <Menu label="Browse and layout" icon={Layers} items={viewMenuItems} activeIds={activeMenuIds} onSelect={selectBrowseOrLayout} />
                  {aiEnabled && <IconButton icon={Sparkles} label="Ask the AI about this library" onClick={() => void navigate({ to: "/lyana" })} />}
                </>
              )}
            </div>
          )}

          {(catalogueShelf || (isPodcasts && view === "shows")) && search.group !== undefined && !collectionsView && (
            <Inline gap={8} align="center">
              <Button variant="secondary" onClick={() => updateSearch({ group: undefined })}>
                ← {isPodcasts ? "All shows" : `All ${view}`}
              </Button>
              {/* The show's name, not its feed hash — the drill-down carries both. */}
              <Text size="13" muted>{search.groupName ?? search.group}</Text>
            </Inline>
          )}
        </Stack>
      </ScreenHeader>

      <Page wide>
        <Stack gap={16}>
          {activeLibrary && !activeIsShelvable ? (
            <Text muted>
              {activeLibrary.name} isn't built yet — {unbuiltReason(activeLibrary.kind)}.
            </Text>
          ) : audioGroupType !== undefined && search.group !== undefined ? (
            /* Inside an album/artist/series/author: tapping a track plays it immediately, no further subpage
               (Kevin's correction — a track is not a navigation target, and the big player's transport controls
               have no place in a browsing list). */
            <AudioGroupTracks
              libraryId={activeLibrary?.id}
              kind={activeLibrary?.kind ?? ""}
              groupType={audioGroupType}
              groupId={search.group}
              layout={layout}
              groupName={search.groupName ?? search.group}
              onBack={() => updateSearch({ group: undefined, groupName: undefined })}
            />
          ) : audioGroupType !== undefined ? (
            <AudioGroupList
              libraryId={activeLibrary?.id}
              groupType={audioGroupType}
              q={searchQuery}
              layout={layout}
              onOpen={(groupId, groupName) => updateSearch({ group: groupId, groupName })}
            />
          ) : isPodcasts && view === "shows" && search.group === undefined ? (
            <PodcastShows layout={layout} onOpen={(id, title) => updateSearch({ group: id, groupName: title })} />
          ) : activeLibrary !== undefined && shelfSource(activeLibrary.kind) !== "catalogue" ? (
            /* Video and audio shelves: /api/videos, /api/video-shows and /api/audio. Their own component, because
               the catalogue endpoints count 0 for a 1388-track music library (see api/media.ts). */
            <MediaShelf
              libraryId={activeLibrary.id}
              kind={activeLibrary.kind}
              source={shelfSource(activeLibrary.kind) as Exclude<ReturnType<typeof shelfSource>, "catalogue">}
              card={family === "music" ? "music" : "book"}
              q={searchQuery}
              layout={layout}
              noun={noun}
              audio={{
                sort: audioSort,
                played: podcastStatus === "all" ? undefined : podcastStatus,
                // Drilled into one show: the same episode list, filtered by the show's feed id.
                feed: isPodcasts && view === "shows" ? search.group : undefined,
              }}
            />
          ) : collectionsView && search.group !== undefined ? (
            <CollectionBooks
              collectionId={search.group}
              layout={layout}
              name={search.groupName ?? "Collection"}
              onBack={() => updateSearch({ group: undefined, groupName: undefined })}
            />
          ) : collectionsView ? (
            <CollectionsList
              libraryId={activeLibrary?.id}
              layout={layout}
              onOpen={(id, name) => updateSearch({ group: id, groupName: name })}
            />
          ) : !grouped ? (
            <Stack gap={16}>
              <Async
                query={books.query}
                empty={(items) => items.length === 0}
                emptyMessage={
                  (search.group !== undefined
                    ? "This group has no books."
                    : searchQuery !== ""
                      ? `No ${noun} match "${searchQuery}".`
                      : status === "in_progress"
                        ? `No ${noun} in progress.`
                        : status === "unread"
                          ? `Every item here has been opened.`
                          : status === "read"
                            ? `No finished ${noun} yet.`
                            : `No ${noun} in this shelf yet.`) as string
                }
              >
                {(rawItems) => {
                  const items = inVolumeOrder(rawItems);
                  return layout === "grid" ? (
                    <MediaGrid>
                      {items.map((book) => (
                        <BookCard
                          key={book.id}
                          title={displayTitle(book)}
                          author={book.authors.join(", ") || "Unknown author"}
                          coverUrl={book.cover_url}
                          kindIcon={kindMeta(book.library_kind).icon}
                          seriesIndex={displaySeriesIndex(book) || undefined}
                          progress={book.reading_progress?.finished ? undefined : book.reading_progress?.fraction}
                          finished={book.reading_progress?.finished}
                          onClick={() => void navigate({ to: "/library/book/$libraryId/$bookId", params: { libraryId: book.library_id, bookId: book.id } })}
                        />
                      ))}
                    </MediaGrid>
                  ) : (
                    <MediaList
                      items={items.map((book) => ({
                        id: book.id,
                        title: displayTitle(book),
                        subtitle: book.authors.join(", ") || "Unknown author",
                        coverUrl: book.cover_url,
                        progress: book.reading_progress?.fraction,
                        finished: book.reading_progress?.finished,
                        onClick: () => void navigate({ to: "/library/book/$libraryId/$bookId", params: { libraryId: book.library_id, bookId: book.id } }),
                      }))}
                    />
                  );
                }}
              </Async>
              {books.hasMore && (
                /* Same control as the grouped lists: the shelf is paged by the server's cursor, so "Show more" is
                   the whole of the paging UI. No page numbers and no infinite scroll — a 5430-book shelf would
                   fire hundreds of cover requests while the reader is still scanning the first screen. */
                <Inline align="center" gap={8}>
                  <Button variant="secondary" onClick={books.loadMore} disabled={books.isLoadingMore}>
                    {books.isLoadingMore ? "Loading…" : "Show more"}
                  </Button>
                </Inline>
              )}
            </Stack>
          ) : (
            <Stack gap={12}>
              <Async query={groups.query} empty={(items) => items.length === 0} emptyMessage={view === "series" ? "No series in this shelf." : "No authors in this shelf."}>
                {(items) =>
                  layout === "list" ? (
                    <MediaList
                      items={items.map((group) => ({
                        id: group.name,
                        title: group.name,
                        subtitle: group.count === 1 ? "1 book" : `${group.count} books`,
                        coverUrl: group.cover_url ?? "",
                        onClick: () => updateSearch({ group: group.name }),
                      }))}
                    />
                  ) : (
                    <MediaGrid>
                      {items.map((group) => (
                        <GroupCard
                          key={group.name}
                          name={group.name}
                          count={group.count}
                          coverUrl={group.cover_url}
                          stack={view === "series"}
                          kindIcon={view === "authors" ? UserRound : undefined}
                          onClick={() => updateSearch({ group: group.name })}
                        />
                      ))}
                    </MediaGrid>
                  )
                }
              </Async>
              {groups.hasMore && (
                <Inline gap={8}>
                  <Button variant="secondary" onClick={groups.loadMore} disabled={groups.isLoadingMore}>
                    {groups.isLoadingMore ? "Loading…" : "Show more"}
                  </Button>
                </Inline>
              )}
            </Stack>
          )}
        </Stack>
      </Page>
    </>
  );
}
