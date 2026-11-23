// features/library/PodcastShows.tsx — the "Shows" browse mode for a podcast shelf: series stacking.
//
// Kevin: "the podcast still has no podcast series stacking support". Before this, a podcast shelf was one flat
// list of every episode of every subscribed show — 1836 rows here — which is unusable when you want one podcast.
//
// Drawn with the same GroupCard the Series/Authors views use, so a show stacks like a series does rather than
// inventing a fourth card family. Tapping one drills into that show's episodes, which is the *same* episode list,
// filtered by `feed=` on the server.
import { useQuery } from "@tanstack/react-query";
import { Async, MediaGrid, MediaList } from "../../ui";
import { Mic } from "lucide-react";
import { GroupCard } from "../../ui";
import { fetchPodcastShows } from "./api/podcastShows";

export interface PodcastShowsProps {
  layout: "grid" | "list";
  onOpen: (id: string, title: string) => void;
}

export function PodcastShows({ layout, onOpen }: PodcastShowsProps) {
  const query = useQuery({
    queryKey: ["library", "podcast-shows"],
    queryFn: fetchPodcastShows,
    staleTime: 60_000,
  });

  return (
    <Async query={query} empty={(shows) => shows.length === 0} emptyMessage="No podcasts subscribed yet.">
      {(shows) =>
        layout === "list" ? (
          <MediaList
            items={shows.map((show) => ({
              id: show.id,
              title: show.title,
              subtitle: show.episodeCount === 1 ? "1 episode" : `${show.episodeCount} episodes`,
              coverUrl: show.coverUrl,
              onClick: () => onOpen(show.id, show.title),
            }))}
          />
        ) : (
          <MediaGrid>
            {shows.map((show) => (
              <GroupCard
                key={show.id}
                name={show.title}
                count={show.episodeCount}
                countLabel={show.episodeCount === 1 ? "1 episode" : `${show.episodeCount} episodes`}
                coverUrl={show.coverUrl}
                kindIcon={Mic}
                // Podcast artwork is square, and a show holds many episodes, so it stacks exactly like a series.
                ratio="1x1"
                stack
                onClick={() => onOpen(show.id, show.title)}
              />
            ))}
          </MediaGrid>
        )
      }
    </Async>
  );
}
