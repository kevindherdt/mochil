// features/library/screens/ShowDetailScreen.tsx — fixes the literal bug reported: opening a TV/anime show did
// nothing, because no detail screen existed for the `shows` data source at all. A season is picked with the
// existing PillRow primitive (it is exactly a selectable row, same as the library's shelf switcher), and each
// season's episode list comes straight from `/api/videos?library=&show=&season=` — already a direct filter, no
// id-paging needed, unlike the show's own metadata (see api/mediaDetail.ts).
import { useState } from "react";
import { useNavigate, useParams, useRouter } from "@tanstack/react-router";
import { Pencil, ArrowLeft, Clapperboard } from "lucide-react";
import {
  Async,
  CoverImage,
  Heading,
  IconButton,
  Inline,
  MediaList,
  Page,
  PillRow,
  Stack,
  Text,
} from "../../../ui";
import { VideoMatchSheet } from "../VideoMatchSheet";
import { clearShowOverride, saveShowOverride } from "../api/videoMatch";
import { useEpisodes, useShowDetail } from "../useMediaDetail";

export function ShowDetailScreen() {
  const { libraryId, itemId } = useParams({ strict: false }) as {
    libraryId?: string;
    itemId?: string;
  };
  const router = useRouter();
  const showQuery = useShowDetail(libraryId, itemId);
  const [season, setSeason] = useState<number | undefined>(undefined);
  const [matching, setMatching] = useState(false);

  return (
    <Page>
      <Stack gap={16}>
        <IconButton
          icon={ArrowLeft}
          label="Back"
          onClick={() => router.history.back()}
        />
        <Async query={showQuery}>
          {(show) => {
            const title = show.metadata_override?.title ?? show.name;
            const overview = show.metadata_override?.overview ?? show.overview;
            const seasons = show.seasons.length > 1 ? show.seasons : [];
            const effectiveSeason =
              season ?? (seasons.length > 0 ? seasons[0]?.number : undefined);
            return (
              <Stack gap={16}>
                <Inline gap={16} align="start">
                  <div style={{ width: 120, flexShrink: 0 }}>
                    <CoverImage
                      src={
                        show.metadata_override?.artwork_url ??
                        show.artwork_url ??
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
                      {/* One pencil for the whole series: a per-episode TMDB match is not a thing anyone wants,
                          and the show-level override is what the cards and episodes read from. */}
                      <IconButton icon={Pencil} label="Fix the TMDB match" onClick={() => setMatching((open) => !open)} />
                    </Inline>
                    <Text size="13" muted>
                      {show.year !== undefined ? String(show.year) : ""}
                    </Text>
                  </Stack>
                </Inline>
                {matching && (
                  <VideoMatchSheet
                    libraryId={libraryId as string}
                    itemId={undefined}
                    show={show.source_name ?? show.name}
                    initialQuery={title}
                    onSave={async (override) => {
                      await saveShowOverride(libraryId as string, show.source_name ?? show.name, override);
                      await showQuery.refetch();
                    }}
                    onClear={async () => {
                      await clearShowOverride(libraryId as string, show.source_name ?? show.name);
                      await showQuery.refetch();
                    }}
                    onClose={() => setMatching(false)}
                  />
                )}

                {overview !== undefined && overview !== "" && (
                  <Text size="14">{overview}</Text>
                )}
                {seasons.length > 0 && (
                  <PillRow
                    items={seasons.map((item) => ({
                      id: String(item.number),
                      label: `Season ${item.number}`,
                    }))}
                    activeId={
                      effectiveSeason !== undefined
                        ? String(effectiveSeason)
                        : undefined
                    }
                    onSelect={(id) => setSeason(Number(id))}
                  />
                )}
                <Episodes
                  libraryId={libraryId}
                  showSourceName={show.source_name ?? show.name}
                  season={effectiveSeason}
                />
              </Stack>
            );
          }}
        </Async>
      </Stack>
    </Page>
  );
}

function Episodes({
  libraryId,
  showSourceName,
  season,
}: {
  libraryId: string | undefined;
  showSourceName: string | undefined;
  season: number | undefined;
}) {
  const navigate = useNavigate();
  const query = useEpisodes(libraryId, showSourceName, season);
  return (
    <Async
      query={query}
      empty={(items) => items.length === 0}
      emptyMessage="No episodes found for this season."
    >
      {(episodes) => (
        <MediaList
          items={episodes.map((episode) => {
            const duration = episode.progress?.duration_seconds ?? 0;
            const fraction =
              duration > 0
                ? (episode.progress?.position_seconds ?? 0) / duration
                : 0;
            const label =
              episode.episode !== undefined
                ? `Episode ${episode.episode}`
                : undefined;
            return {
              id: episode.id,
              title: episode.metadata_override?.title ?? episode.title,
              subtitle: label,
              coverUrl:
                episode.metadata_override?.artwork_url ??
                episode.artwork_url ??
                "",
              progress: fraction > 0.001 && fraction < 1 ? fraction : undefined,
              finished: fraction >= 0.98,
              onClick: () =>
                void navigate({
                  to: "/library/video/$libraryId/$itemId",
                  params: { libraryId: libraryId!, itemId: episode.id },
                }),
            };
          })}
        />
      )}
    </Async>
  );
}
