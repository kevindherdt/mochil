// features/library/MediaShelf.tsx — the shelves served by their own endpoints rather than /api/books (see
// api/media.ts). Two independent questions decide how a shelf draws, and they genuinely differ:
//
//   * **where the items come from** — `audio`, `videos` or `shows` (kinds.ts's shelfSource);
//   * **which card draws them** — the book card (2:3 poster, right for a film and for an audiobook) or the square
//     music card.
//
// Audiobooks are the case that proves they're separate: 410 audio files, zero catalogue rows, drawn with the book
// card. Treating "audio" as one family made that shelf look empty.
//
// What a tap does is a third, separate question (Kevin's correction, 22 Nov): a video/show has nowhere to go but
// a detail page, but an audio item has nowhere to "open" at all — tapping a track should start playing it on the
// spot, the same as any music app, not push a subpage with nothing on it but a play button. Only the mini
// player's own "now playing" bar opens the detail/now-playing screen (features/player/MiniPlayer.tsx).
//
// No stylesheet here on purpose — law L2: a feature component arranges primitives, and every size comes from them.
import { useNavigate } from "@tanstack/react-router";
import { Async, Button, BookCard, Inline, MediaGrid, MediaList, MusicCard } from "../../ui";
import { kindMeta } from "./kinds";
import type { ShelfSource } from "./kinds";
import type { ShelfItem } from "./api/media";
import { useShelfMedia, type ShelfAudioFilter } from "./useShelfMedia";
import { usePlayer } from "../player/PlayerContext";

export interface MediaShelfProps {
  libraryId: string;
  /** The library's kind, for the card's glyph. */
  kind: string;
  source: Exclude<ShelfSource, "catalogue">;
  /** `music` = square artwork; `book` = 2:3, used for films and audiobooks. */
  card: "book" | "music";
  q: string;
  layout: "grid" | "list";
  /** "titles", "albums", "audiobooks" — the empty state names what the shelf holds. */
  noun: string;
  /** Audio shelves only: the server-side ordering and played filter (podcasts, audiobooks). */
  audio?: ShelfAudioFilter;
}

export function MediaShelf({ libraryId, kind, source, card, q, layout, noun, audio }: MediaShelfProps) {
  const shelf = useShelfMedia(libraryId, source, q, audio);
  const icon = kindMeta(kind).icon;
  const navigate = useNavigate();
  const player = usePlayer();

  // Audio plays the *shelf* from the tapped track, not the track alone. Playing a single item left the queue empty,
  // so "ended" had nowhere to go and the shelf could never advance by itself — the one thing a music library is
  // expected to do. Shows and videos still have a real detail page to go to.
  const playFrom = (list: ShelfItem[], index: number): void => {
    const tracks = list.flatMap((entry) =>
      entry.streamUrl === undefined
        ? []
        : [
            {
              id: entry.id,
              libraryId,
              title: entry.title,
              subtitle: entry.subtitle,
              coverUrl: entry.coverUrl,
              formatLabel: entry.formatLabel,
              streamUrl: entry.streamUrl,
            },
          ],
    );
    const tapped = list[index];
    if (tapped === undefined) return;
    const startAt = tracks.findIndex((candidate) => candidate.id === tapped.id);
    if (startAt === -1) return;
    player.playQueue(tracks, startAt);
  };

  const open = (item: ShelfItem, index: number, list: ShelfItem[]): void => {
    if (source === "shows") {
      void navigate({ to: "/library/show/$libraryId/$itemId", params: { libraryId, itemId: item.id } });
      return;
    }
    if (source === "videos") {
      void navigate({ to: "/library/video/$libraryId/$itemId", params: { libraryId, itemId: item.id } });
      return;
    }
    if (item.streamUrl === undefined) return; // the server always sends one for /api/audio; defensive only
    playFrom(list, index);
  };

  return (
    <>
      <Async query={shelf.query} empty={(items) => items.length === 0} emptyMessage={q !== "" ? `No ${noun} match "${q}".` : `No ${noun} in this shelf yet.`}>
        {(items) =>
          card === "music" ? (
            layout === "list" ? (
              <MediaList
                items={items.map((item, index) => ({
                  id: item.id,
                  title: item.title,
                  subtitle: [item.subtitle, item.formatLabel].filter((part) => part !== undefined && part !== "").join(" · "),
                  coverUrl: item.coverUrl,
                  // A podcast episode is long-form and is resumed like a book, so the list shows the same bar.
                  progress: item.progress,
                  finished: item.finished,
                  onClick: () => open(item, index, items),
                }))}
              />
            ) : (
              /* Square artwork, no progress chrome, format strip (FLAC/MP3) on the artwork — plan B2's music card. */
              <MediaGrid>
                {items.map((item, index) => (
                  <MusicCard
                    key={item.id}
                    title={item.title}
                    artist={item.subtitle || "Unknown artist"}
                    coverUrl={item.coverUrl}
                    formatLabel={item.formatLabel}
                    kindIcon={icon}
                    onClick={() => open(item, index, items)}
                  />
                ))}
              </MediaGrid>
            )
          ) : layout === "list" ? (
            <MediaList
              items={items.map((item, index) => ({
                id: item.id,
                title: item.title,
                subtitle: item.subtitle,
                coverUrl: item.coverUrl,
                progress: item.progress,
                finished: item.finished,
                onClick: () => open(item, index, items),
              }))}
            />
          ) : (
            /* A poster is 2:3 like a cover, so the book card is the right card — the resolution label reuses the
               slot the cover glyph strip already had. */
            <MediaGrid>
              {items.map((item, index) => (
                <BookCard
                  key={item.id}
                  title={item.title}
                  author={item.subtitle}
                  coverUrl={item.coverUrl}
                  kindIcon={icon}
                  formatLabel={item.formatLabel}
                  progress={item.progress}
                  finished={item.finished}
                  onClick={() => open(item, index, items)}
                />
              ))}
            </MediaGrid>
          )
        }
      </Async>
      {shelf.hasMore && (
        <Inline align="center" gap={8}>
          <Button variant="secondary" onClick={shelf.loadMore} disabled={shelf.isLoadingMore}>
            {shelf.isLoadingMore ? "Loading…" : "Show more"}
          </Button>
        </Inline>
      )}
    </>
  );
}
