export function audioRenderPolicy({
  kind,
  browse,
  itemCount,
  total,
  startOffset = 0,
  visibleItemCount = 0,
  podcast = {},
  audiobook = {},
  music = {},
}) {
  const groupedMusic = kind === "music" && ["albums", "artists"].includes(browse);
  const groupedAudiobooks =
    kind === "audiobooks" && ["series", "authors"].includes(browse);

  const loaded =
    kind === "podcasts" && podcast.show
      ? podcast.episodeCount
      : groupedAudiobooks && audiobook.group
        ? audiobook.bookCount
        : groupedAudiobooks
          ? audiobook.groupCount
          : groupedMusic && !music.selectedGroup
            ? music.groupCount
            : kind === "music" && browse === "tracks"
              ? startOffset + itemCount
              : itemCount;

  const available =
    kind === "podcasts" && podcast.show
      ? podcast.total
      : groupedAudiobooks && audiobook.group
        ? audiobook.total
        : groupedAudiobooks
          ? audiobook.groupCount
          : groupedMusic && !music.selectedGroup
            ? music.total
            : total;

  const staticMusicView =
    kind === "music" &&
    (["discover", "recently-played", "favorites", "playlists", "queue"].includes(
      browse,
    ) ||
      (groupedMusic && music.selectedGroup));

  const loadMoreHidden =
    (kind === "podcasts" && !podcast.show) ||
    (kind === "audiobooks" && !audiobook.group) ||
    staticMusicView ||
    loaded >= available;

  return {
    loaded,
    available,
    loadMoreHidden,
    loadMoreLabel:
      kind === "podcasts"
        ? "Load more episodes"
        : kind === "audiobooks"
          ? "Load more audiobooks"
          : groupedMusic
            ? "Load more collections"
            : "Load more",
  };
}
