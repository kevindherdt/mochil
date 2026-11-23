// features/library/screens/AudioDetailScreen.tsx — the mock-up's "big player" ("2 music player big"), for music
// tracks, podcast episodes and audiobook files: all three answer /api/audio and share a shape
// (title/artist-or-album/container/stream_url). Playback is the global player (features/player/PlayerContext),
// not a local <audio> element: the same track keeps playing if the reader navigates away, and the mini player
// (the shell's docked bar) reflects this exact screen's state — and is hidden while this exact screen is open
// (features/player/MiniPlayer.tsx), rather than showing two transports for the same track at once.
//
// Left out of the mock-up on purpose, and said so rather than faked: AI voice narration / Whispersync (a
// different subsystem — book narration, not generic audio playback), the favourite heart, and a queue/next-track
// (nothing here hands the player a playlist to advance through yet). Drag-to-seek is also not built — the
// progress line is display-only; -15/+15 are real seeks.
//
// **Chapters** (Kevin: "for the audiobooks that support it also be able to show chapters etc") come from the file
// itself — `audio.Item.Chapters`, probed from the m4b markers by the scanner and sent on `/api/audio`. They are
// shown only when the file actually has them: an mp3 audiobook without markers gets no empty "Chapters" heading
// pretending otherwise. Tapping one seeks *absolutely* (`seekTo`), and starts the track if it is not the one
// playing, so a chapter is one tap from the big player.
import { useParams, useRouter } from "@tanstack/react-router";
import { ArrowLeft, Music2 } from "lucide-react";
import { Async, CoverImage, Heading, IconButton, Inline, PlaybackControls, Page, Stack, Text } from "../../../ui";
import { useAudioDetail } from "../useMediaDetail";
import { usePlayer } from "../../player/PlayerContext";
import { useSwipeDownToDismiss } from "../../player/useSwipeDownToDismiss";

/** h:mm:ss when there are hours, m:ss otherwise — the same shape the transport uses. */
function formatClock(seconds: number): string {
  const total = Math.max(0, Math.floor(seconds));
  const hours = Math.floor(total / 3600);
  const minutes = Math.floor((total % 3600) / 60);
  const rest = total % 60;
  const pad = (value: number) => String(value).padStart(2, "0");
  return hours > 0 ? `${hours}:${pad(minutes)}:${pad(rest)}` : `${minutes}:${pad(rest)}`;
}

export function AudioDetailScreen() {
  const { libraryId, itemId } = useParams({ strict: false }) as { libraryId?: string; itemId?: string };
  const router = useRouter();
  const query = useAudioDetail(libraryId, itemId);
  const player = usePlayer();
  const isThisTrack = player.track?.id === itemId;
  const swipe = useSwipeDownToDismiss(() => router.history.back());

  return (
    <div onPointerDown={swipe.onPointerDown} onPointerMove={swipe.onPointerMove} onPointerUp={swipe.onPointerUp} style={swipe.style}>
    <Page>
      <Stack gap={24}>
        <div style={{ display: "flex", alignItems: "center", gap: "var(--space-8)" }}>
          <IconButton icon={ArrowLeft} label="Back" onClick={() => router.history.back()} />
          <div style={{ flex: 1, textAlign: "center" }}>
            <Heading size="16">Now Playing</Heading>
          </div>
          {/* Balances the back button's width so the title above sits in the true centre, not shifted right. */}
          <div style={{ width: "var(--space-32)", flexShrink: 0 }} aria-hidden="true" />
        </div>
        <Async query={query}>
          {(item) => {
            const title = item.title;
            const subtitle = item.artist ?? item.album_artist ?? item.album ?? "";
            const formatLabel = item.container?.toUpperCase();
            const chapters = (item.chapters ?? []).map((chapter) => ({
              title: chapter.title,
              start: chapter.start_seconds,
              end: chapter.end_seconds,
            }));
            // The queue, on the right at desktop width and below the transport on a phone. It only exists when the
            // shelf that started playback handed its list in — a single file played on its own has nothing after it,
            // and an empty "Up next" panel would be a control with nothing behind it.
            const upNext = (() => {
              const at = player.queue.findIndex((candidate) => candidate.id === player.track?.id);
              return at === -1 ? [] : player.queue.slice(at + 1);
            })();
            return (
              <div style={{ display: "flex", flexWrap: "wrap", gap: "var(--space-24)", alignItems: "flex-start" }}>
              <div style={{ flex: "1 1 420px", minWidth: 0 }}>
              <Stack gap={24} align="center">
                {/* A framed card behind the artwork, the same raised surface a Card uses — the mock-up's album
                    art sits on a pale mat rather than directly on the page background. */}
                <div style={{ width: "min(260px, 70%)", padding: "var(--space-16)", background: "var(--color-raised)", borderRadius: "var(--radius-16)" }}>
                  <CoverImage src={item.artwork_url ?? ""} alt={title} ratio="1x1" kindIcon={Music2} radius="8" />
                </div>
                {formatLabel !== undefined && (
                  <div style={{ padding: "var(--space-4) var(--space-12)", background: "var(--color-raised)", borderRadius: "var(--radius-pill)" }}>
                    <Text size="11" muted>{formatLabel}</Text>
                  </div>
                )}
                <Stack gap={4} align="center">
                  <Heading size="22">{title}</Heading>
                  <Text muted>{subtitle}</Text>
                  {item.album !== undefined && item.album !== "" && (
                    <Text size="13" muted>{item.album}</Text>
                  )}
                </Stack>
                {item.stream_url !== undefined && (
                  <div style={{ width: "100%" }}>
                    <PlaybackControls
                      isPlaying={isThisTrack && player.isPlaying}
                      position={isThisTrack ? player.position : 0}
                      duration={isThisTrack ? player.duration : 0}
                      onTogglePlay={() =>
                        isThisTrack
                          ? player.togglePlay()
                          : player.play({
                              id: item.id,
                              libraryId: libraryId!,
                              title,
                              subtitle,
                              coverUrl: item.artwork_url ?? "",
                              formatLabel,
                              streamUrl: item.stream_url!,
                            })
                      }
                      onSkip={player.skip}
                    />
                  </div>
                )}
                {item.description !== undefined && item.description !== "" && <Text size="14">{item.description}</Text>}

                {upNext.length > 0 && (
                  <div style={{ width: "100%" }}>
                  <Stack gap={8}>
                    <Heading size="16">Up next</Heading>
                    <div>
                      {upNext.map((queued, offset) => (
                        <button
                          key={queued.id}
                          type="button"
                          onClick={() => {
                            const index = player.queue.findIndex((candidate) => candidate.id === queued.id);
                            if (index !== -1) player.playQueue(player.queue, index);
                          }}
                          style={{
                            all: "unset",
                            cursor: "pointer",
                            display: "flex",
                            alignItems: "baseline",
                            gap: "var(--space-12)",
                            width: "100%",
                            boxSizing: "border-box",
                            padding: "var(--space-8) 0",
                            borderBottom: "1px solid var(--color-line)",
                          }}
                        >
                          <Text size="11" muted>
                            {offset + 1}
                          </Text>
                          <Stack gap={4}>
                            <Text size="13">{queued.title}</Text>
                            {queued.subtitle !== undefined && queued.subtitle !== "" && (
                              <Text size="11" muted>{queued.subtitle}</Text>
                            )}
                          </Stack>
                        </button>
                      ))}
                    </div>
                  </Stack>
                  </div>
                )}

                {chapters.length > 0 && (
                  <Stack gap={8}>
                    <Inline justify="space-between" align="center">
                      <Heading size="16">Chapters</Heading>
                      <Text size="11" muted>
                        {chapters.length}
                      </Text>
                    </Inline>
                    <div>
                      {chapters.map((chapter, index) => {
                        // The chapter playing right now gets the accent bar, so "where am I" is answerable from
                        // this list without looking at the transport.
                        const active = isThisTrack && player.position >= chapter.start && player.position < chapter.end;
                        return (
                          <button
                            key={`${chapter.start}-${index}`}
                            type="button"
                            onClick={() => {
                              if (!isThisTrack) {
                                if (item.stream_url === undefined) return;
                                player.play({
                                  id: item.id,
                                  libraryId: libraryId!,
                                  title,
                                  subtitle,
                                  coverUrl: item.artwork_url ?? "",
                                  formatLabel,
                                  streamUrl: item.stream_url,
                                });
                              }
                              // The element needs a tick to pick up a freshly-set src before a seek lands.
                              window.setTimeout(() => player.seekTo(chapter.start), isThisTrack ? 0 : 250);
                            }}
                            style={{
                              all: "unset",
                              cursor: "pointer",
                              display: "flex",
                              alignItems: "baseline",
                              gap: "var(--space-12)",
                              width: "100%",
                              boxSizing: "border-box",
                              padding: "var(--space-12) 0",
                              borderBottom: "1px solid var(--color-line)",
                              borderLeft: active ? "3px solid var(--color-accent)" : "3px solid transparent",
                              paddingLeft: "var(--space-8)",
                            }}
                          >
                            <Text as="span" size="11" muted>
                              {formatClock(chapter.start)}
                            </Text>
                            <span style={{ flex: 1, minWidth: 0 }}>
                              <Text size="14">{chapter.title}</Text>
                            </span>
                          </button>
                        );
                      })}
                    </div>
                  </Stack>
                )}
              </Stack>
              </div>
              </div>
            );
          }}
        </Async>
      </Stack>
    </Page>
    </div>
  );
}
