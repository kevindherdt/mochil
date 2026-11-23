// features/library/screens/VideoPlayerScreen.tsx — the route for /play/{library}/{item}. Deliberately thin: it reads
// the route, loads the item, and hands everything to VideoPlayback. Screens compose; features own layout (law L2).
//
// The shell hides its chrome for this path (app/router.tsx's ShellLayout sets `bare`), which is what makes the player
// own the whole screen instead of sharing it with the dock — the thing "Plex styled" was actually asking for.
import { useNavigate, useParams, useRouter, useSearch } from "@tanstack/react-router";
import { Text } from "../../../ui";
import { useVideoDetail } from "../useMediaDetail";
import { VideoPlayback } from "../VideoPlayback";

export function VideoPlayerScreen() {
  const { libraryId, itemId } = useParams({ strict: false }) as { libraryId?: string; itemId?: string };
  // `?restart=1` is the detail page's "From the start" next to "Resume from 12:34".
  const search = useSearch({ strict: false }) as { restart?: string };
  const router = useRouter();
  const navigate = useNavigate();
  const query = useVideoDetail(libraryId, itemId);

  if (query.data === undefined) {
    return (
      <div
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          width: "100%",
          height: "100%",
          background: "var(--color-bg)",
        }}
      >
        <Text size="13" muted>{query.isError ? "This video could not be loaded." : "Opening…"}</Text>
      </div>
    );
  }

  const item = query.data;
  return (
    <VideoPlayback
      libraryId={libraryId!}
      itemId={itemId!}
      title={item.metadata_override?.title ?? item.title}
      mediaRevision={item.revision}
      streamUrl={item.stream_url}
      hlsUrl={item.hls_url}
      positionSeconds={search.restart === "1" ? 0 : (item.progress?.position_seconds ?? 0)}
      durationSeconds={item.progress?.duration_seconds ?? item.duration_seconds ?? 0}
      audioTracks={item.audio_streams}
      subtitleTracks={item.subtitle_streams}
      onBack={() => router.history.back()}
      onOpenDetail={() =>
        void navigate({ to: "/library/video/$libraryId/$itemId", params: { libraryId: libraryId!, itemId: itemId! } })
      }
    />
  );
}
