// features/library/api/podcastShows.ts — the podcast "Shows" list, i.e. series stacking.
//
// Kevin: "the podcast still has no podcast series stacking support ... often there are podcasts with 1000s of
// episodes". A podcast shelf is not a flat list of episodes any more than a TV library is a flat list of 40 loose
// episodes — it is a shelf of shows, each of which is a list of episodes.
//
// Source choice, deliberate: `/api/podcast-feeds` rather than `/api/audio-groups`. The grouping endpoint only ever
// sees `library.Audio.Items()`, which for podcasts is the handful of *downloaded* episode files — a show list built
// from it would silently shrink to "shows you have downloaded something from". `/api/podcast-feeds`
// (summarizePodcastFeeds, internal/web/podcast_summary.go) is the subscription list itself, every feed with its own
// episode count and artwork, which is the actual shelf. Episodes for one show then come from
// `/api/audio?library=podcasts&feed=<id>`, which the server already filters (apiAudio's `feed` parameter).
import { z } from "zod";
import { request } from "../../../lib/http";

const feedSchema = z.looseObject({
  id: z.string(),
  title: z.string(),
  description: z.string().optional(),
  artwork_url: z.string().optional(),
  episode_count: z.number().optional(),
  updated: z.string().optional(),
});

export interface PodcastShow {
  id: string;
  title: string;
  description: string;
  coverUrl: string;
  episodeCount: number;
}

export async function fetchPodcastShows(): Promise<PodcastShow[]> {
  const raw = await request<unknown>("/api/podcast-feeds");
  return z.array(feedSchema).parse(raw ?? []).map((feed): PodcastShow => ({
    id: feed.id,
    title: feed.title,
    description: feed.description ?? "",
    coverUrl: feed.artwork_url ?? "",
    episodeCount: feed.episode_count ?? 0,
  }));
}
