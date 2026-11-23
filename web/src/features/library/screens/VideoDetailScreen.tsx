// features/library/screens/VideoDetailScreen.tsx — a film's detail page. `stream_url` only exists when the file
// is `DirectPlay` (`internal/web/video_catalogue_http.go`'s `streamVideo` 404s on anything that needs the
// transcoder); everything else answers `hls_url` instead, which a bare `<video>` can't read without hls.js. That
// decoder, plus a real player chrome (seek bar, resume-on-leave, subtitle tracks), is `PlayerSlot`'s job and is
// tracked separately (docs/rewrite-progress.md's Blocked section) — this screen plays what it honestly can and
// says so, rather than wiring something that silently fails on half the library.
import { useParams, useRouter, useNavigate } from "@tanstack/react-router";
import { useState } from "react";
import { Pencil, ArrowLeft, Clapperboard, Play } from "lucide-react";
import {
  Async,
  Button,
  CoverImage,
  Heading,
  IconButton,
  Inline,
  Page,
  Stack,
  Text,
} from "../../../ui";
import { VideoMatchSheet } from "../VideoMatchSheet";
import { clearItemOverride, saveItemOverride } from "../api/videoMatch";
import { resolutionLabel } from "../api/media";
import { useVideoDetail } from "../useMediaDetail";

const RESUME_THRESHOLD_S = 5;
const RESUME_END_GUARD_S = 10;

/** Where playback would start. Mirrors the player's own rule, so the button and the player cannot disagree about
 *  whether this is a resume or a fresh start. */
function resumeAt(item: { progress?: { position_seconds: number; duration_seconds: number } | undefined; duration_seconds?: number | undefined }): number {
  const position = item.progress?.position_seconds ?? 0;
  const duration = item.progress?.duration_seconds ?? item.duration_seconds ?? 0;
  if (position <= RESUME_THRESHOLD_S) return 0;
  if (duration > 0 && position >= duration - RESUME_END_GUARD_S) return 0;
  return position;
}

function clock(seconds: number): string {
  const total = Math.max(0, Math.floor(seconds));
  const h = Math.floor(total / 3600);
  const m = Math.floor((total % 3600) / 60);
  const s = total % 60;
  const pad = (value: number) => String(value).padStart(2, "0");
  return h > 0 ? `${h}:${pad(m)}:${pad(s)}` : `${m}:${pad(s)}`;
}

function resumeLabel(item: Parameters<typeof resumeAt>[0]): string {
  const at = resumeAt(item);
  return at > 0 ? `Resume from ${clock(at)}` : "Play";
}

export function VideoDetailScreen() {
  const { libraryId, itemId } = useParams({ strict: false }) as {
    libraryId?: string;
    itemId?: string;
  };
  const router = useRouter();
  const navigate = useNavigate();
  const query = useVideoDetail(libraryId, itemId);
  const [matching, setMatching] = useState(false);

  return (
    <Page>
      <Stack gap={16}>
        <IconButton
          icon={ArrowLeft}
          label="Back"
          onClick={() => router.history.back()}
        />
        <Async query={query}>
          {(item) => {
            const title = item.metadata_override?.title ?? item.title;
            const overview = item.metadata_override?.overview ?? item.overview;
            const year = item.metadata_override?.year ?? item.year;
            return (
              <Stack gap={16}>
                <Inline gap={16} align="start">
                  <div style={{ width: 120, flexShrink: 0 }}>
                    <CoverImage
                      src={
                        item.metadata_override?.artwork_url ??
                        item.artwork_url ??
                        ""
                      }
                      alt={title}
                      kindIcon={Clapperboard}
                      radius="8"
                    />
                  </div>
                  <Stack gap={4}>
                    <Inline gap={8} align="center">
                      <Heading size="22">{title}</Heading>
                      {/* The pencil Kevin asked for: TMDB matching lives behind it, on the title of the thing
                          being fixed, rather than as a library-wide button. */}
                      <IconButton icon={Pencil} label="Fix the TMDB match" onClick={() => setMatching((open) => !open)} />
                    </Inline>
                    <Text size="13" muted>
                      {[year?.toString(), resolutionLabel(item)]
                        .filter(Boolean)
                        .join(" · ")}
                    </Text>
                  </Stack>
                </Inline>
                {matching && (
                  <VideoMatchSheet
                    libraryId={libraryId as string}
                    itemId={item.id}
                    show={undefined}
                    initialQuery={title}
                    onSave={async (override) => {
                      await saveItemOverride(libraryId as string, item.id, override);
                      await query.refetch();
                    }}
                    onClear={async () => {
                      await clearItemOverride(libraryId as string, item.id);
                      await query.refetch();
                    }}
                    onClose={() => setMatching(false)}
                  />
                )}

                {item.stream_url !== undefined || item.hls_url !== undefined ? (
                  /* A play action, not an embedded video: the film opens the full-screen player at /play/…, which is
                     what "Plex styled" meant here — the detail page is a page, the player is a player. Resume and
                     restart are separate choices because "Play" on something half-watched is ambiguous otherwise. */
                  <Inline gap={8}>
                    <Button
                      variant="primary"
                      onClick={() =>
                        void navigate({
                          to: "/play/$libraryId/$itemId",
                          params: { libraryId: libraryId!, itemId: item.id },
                        })
                      }
                    >
                      <Play size={16} aria-hidden="true" />
                      {resumeLabel(item)}
                    </Button>
                    {resumeAt(item) > 0 && (
                      <Button
                        variant="secondary"
                        onClick={() =>
                          void navigate({
                            to: "/play/$libraryId/$itemId",
                            params: { libraryId: libraryId!, itemId: item.id },
                            search: { restart: "1" },
                          })
                        }
                      >
                        From the start
                      </Button>
                    )}
                  </Inline>
                ) : (
                  <Text size="13" muted>
                    This file has no playable source — the server could not find a stream for it. Use the classic app
                    for now.
                  </Text>
                )}
                {overview !== undefined && overview !== "" && (
                  <Text size="14">{overview}</Text>
                )}
              </Stack>
            );
          }}
        </Async>
      </Stack>
    </Page>
  );
}
