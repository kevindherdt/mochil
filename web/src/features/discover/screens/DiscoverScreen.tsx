// features/discover/screens/DiscoverScreen.tsx — plan B3, extended 22 Nov 2026.
//
// Four rails, all from endpoints that already existed: "Continue reading" (/api/reading), "Continue listening"
// (/api/audio-progress?continue=1, music excluded — that store is listening history for music, not continuation),
// "Continue watching" (/api/video-progress?continue=1) and "Recently added" (/api/books?sort=recent). Kevin:
// "on the discover page it still only shows continue reading, but no continue listening ... and then no continue
// watching either" — the progress stores already had a `continue` filter; Discover just never asked.
//
// The point of the screen is that whoever is mid-book, mid-episode or mid-film never has to find where they were.
//
// A rail is skipped rather than shown empty when it has nothing in it, and a screen with neither rail gets one
// honest empty state with the action that resolves it — the plan's rule for empty/error states.
import { useNavigate } from "@tanstack/react-router";
import { Heading, Stack, Rail, BookCard, MusicCard, Async, EmptyState, Page } from "../../../ui";
import { usePlayer } from "../../player/PlayerContext";
import { useContinueListening, useContinueReading, useContinueWatching, useRecentlyAdded } from "../useDiscover";
import { kindMeta } from "../../library/kinds";
import { displayTitle, displaySeriesIndex } from "../../library/titles";

export function DiscoverScreen() {
  const navigate = useNavigate();
  const continuing = useContinueReading();
  const listening = useContinueListening();
  const watching = useContinueWatching();
  const recent = useRecentlyAdded();
  const player = usePlayer();

  // "Empty" now means all four rails are, or the screen would claim there is nothing here while showing a rail.
  const nothingToContinue = continuing.data !== undefined && continuing.data.length === 0;
  const nothingListening = listening.data !== undefined && listening.data.length === 0;
  const nothingWatching = watching.data !== undefined && watching.data.length === 0;
  const nothingRecent = recent.data !== undefined && recent.data.length === 0;
  const bothEmpty = nothingToContinue && nothingListening && nothingWatching && nothingRecent;

  return (
    <Page wide>
      <Stack gap={24}>
        {bothEmpty ? (
          <EmptyState
            message="Nothing here yet — add some books to a library and they will show up."
            action={{ label: "Open the library", onClick: () => void navigate({ to: "/library" }) }}
          />
        ) : (
          <>
            <Stack gap={12}>
              <Heading size="16">Continue reading</Heading>
              <Async
                query={continuing}
                empty={(items) => items.length === 0}
                // No action here: "nothing in progress" isn't a failure, it's a reader who hasn't started anything.
                emptyMessage="Nothing in progress right now."
                skeleton={<Rail label="Continue reading"><div /></Rail>}
              >
                {(items) => (
                  <Rail label="Continue reading">
                    {items.map((item) => (
                      <BookCard
                        key={`${item.library_id}/${item.id}`}
                        title={displayTitle(item)}
                        author={item.authors.join(", ") || "Unknown author"}
                        coverUrl={item.cover_url}
                        kindIcon={kindMeta(item.library_kind).icon}
                        seriesIndex={displaySeriesIndex(item) || undefined}
                        progress={item.fraction}
                        onClick={() => void navigate({ to: "/library/book/$libraryId/$bookId", params: { libraryId: item.library_id, bookId: item.id } })}
                      />
                    ))}
                  </Rail>
                )}
              </Async>
            </Stack>

            <Stack gap={12}>
              <Heading size="16">Continue listening</Heading>
              <Async
                query={listening}
                empty={(items) => items.length === 0}
                emptyMessage="Nothing half-listened right now."
                skeleton={<Rail label="Continue listening"><div /></Rail>}
              >
                {(items) => (
                  <Rail label="Continue listening">
                    {items.map((item) => (
                      // Podcast artwork is square, an audiobook cover is 2:3 — drawing a podcast in the book card
                      // cropped the show name off both sides, which the first pass at this rail did.
                      item.kind === "podcasts" ? (
                        <MusicCard
                          key={`${item.libraryId}/${item.id}`}
                          title={item.title}
                          artist={item.subtitle || "Podcast"}
                          coverUrl={item.coverUrl}
                          kindIcon={kindMeta(item.kind).icon}
                          onClick={() => {
                            if (item.streamUrl === undefined) return;
                            player.play({
                              id: item.id,
                              libraryId: item.libraryId,
                              title: item.title,
                              subtitle: item.subtitle,
                              coverUrl: item.coverUrl,
                              streamUrl: item.streamUrl,
                            });
                          }}
                        />
                      ) : (
                      <BookCard
                        key={`${item.libraryId}/${item.id}`}
                        title={item.title}
                        author={item.subtitle || "Audiobook"}
                        coverUrl={item.coverUrl}
                        kindIcon={kindMeta(item.kind).icon}
                        progress={item.fraction > 0 ? item.fraction : undefined}
                        // An episode or audiobook resumes on the spot, like every other audio tap in the app; the
                        // big player is reached from the mini player, not by opening a detail page for a file.
                        onClick={() => {
                          if (item.streamUrl === undefined) return;
                          player.play({
                            id: item.id,
                            libraryId: item.libraryId,
                            title: item.title,
                            subtitle: item.subtitle,
                            coverUrl: item.coverUrl,
                            streamUrl: item.streamUrl,
                          });
                        }}
                      />
                      )
                    ))}
                  </Rail>
                )}
              </Async>
            </Stack>

            <Stack gap={12}>
              <Heading size="16">Continue watching</Heading>
              <Async
                query={watching}
                empty={(items) => items.length === 0}
                emptyMessage="Nothing half-watched right now."
                skeleton={<Rail label="Continue watching"><div /></Rail>}
              >
                {(items) => (
                  <Rail label="Continue watching">
                    {items.map((item) => (
                      <BookCard
                        key={`${item.libraryId}/${item.id}`}
                        title={item.title}
                        author={item.subtitle || "Video"}
                        coverUrl={item.coverUrl}
                        kindIcon={kindMeta("movies").icon}
                        progress={item.fraction > 0 ? item.fraction : undefined}
                        onClick={() =>
                          void navigate({ to: "/library/video/$libraryId/$itemId", params: { libraryId: item.libraryId, itemId: item.id } })
                        }
                      />
                    ))}
                  </Rail>
                )}
              </Async>
            </Stack>

            <Stack gap={12}>
              <Heading size="16">Recently added</Heading>
              <Async
                query={recent}
                empty={(items) => items.length === 0}
                emptyMessage="No books have been added yet."
                skeleton={<Rail label="Recently added"><div /></Rail>}
              >
                {(items) => (
                  <Rail label="Recently added">
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
                  </Rail>
                )}
              </Async>
            </Stack>
          </>
        )}
      </Stack>
    </Page>
  );
}
