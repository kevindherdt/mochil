// features/library/AudioGroupTracks.tsx — the tracklist inside one "Albums"/"Artists"/"Series"/"Authors" group.
// Kevin's correction: tapping a track here must play it immediately, the same as the flat shelf — it is not
// another subpage, and the big player's -15/play/+15 chrome has no place in a browsing list. Reuses BookCard
// (book-like groups) or MusicCard (music groups) exactly like MediaShelf's flat view, for the same reason: the
// card family is about what the artwork looks like, not where the list of items came from.
import { BookCard, MediaGrid, MediaList, MusicCard, Async, Inline, Button, Text } from "../../ui";
import { kindMeta } from "./kinds";
import { useAudioGroupTracks } from "./useMediaDetail";
import { usePlayer, type NowPlayingTrack } from "../player/PlayerContext";
import type { AudioGroupKind } from "./api/audioGroups";

export interface AudioGroupTracksProps {
  libraryId: string | undefined;
  kind: string;
  groupType: AudioGroupKind;
  groupId: string | undefined;
  layout: "grid" | "list";
  onBack: () => void;
  groupName: string;
}

export function AudioGroupTracks({ libraryId, kind, groupType, groupId, layout, onBack, groupName }: AudioGroupTracksProps) {
  const query = useAudioGroupTracks(libraryId, groupType, groupId);
  const player = usePlayer();
  const icon = kindMeta(kind).icon;
  const music = groupType === "albums" || groupType === "artists";

  const toTrack = (item: { id: string; title: string; subtitle: string; coverUrl: string; formatLabel?: string | undefined; streamUrl?: string | undefined }): NowPlayingTrack | undefined => {
    if (item.streamUrl === undefined || libraryId === undefined) return undefined;
    return { id: item.id, libraryId, title: item.title, subtitle: item.subtitle, coverUrl: item.coverUrl, formatLabel: item.formatLabel, streamUrl: item.streamUrl };
  };

  const play = (item: Parameters<typeof toTrack>[0]) => {
    const track = toTrack(item);
    if (track === undefined) return;
    player.play(track);
  };

  return (
    <>
      <Inline gap={8} align="center">
        <Button variant="secondary" onClick={onBack}>
          ← All {groupType}
        </Button>
        <Text size="13" muted>{groupName}</Text>
        {/* One tap to hear the whole album/series, in the order the group already lists them. Before this the
            player stopped dead at the end of every track, which is what "no queue/next-track yet" meant in
            practice. */}
        <Async query={query}>
          {(tracks) => {
            const playable = tracks.map(toTrack).filter((item): item is NowPlayingTrack => item !== undefined);
            if (playable.length === 0) return null;
            return (
              <Button variant="primary" onClick={() => player.playQueue(playable, 0)}>
                Play all · {playable.length}
              </Button>
            );
          }}
        </Async>
      </Inline>
      <Async query={query} empty={(items) => items.length === 0} emptyMessage="This group has no tracks.">
        {(items) =>
          music ? (
            <MediaGrid>
              {items.map((item) => (
                <MusicCard
                  key={item.id}
                  title={item.title}
                  artist={item.subtitle || "Unknown artist"}
                  coverUrl={item.coverUrl}
                  formatLabel={item.formatLabel}
                  kindIcon={icon}
                  onClick={() => play(item)}
                />
              ))}
            </MediaGrid>
          ) : layout === "list" ? (
            <MediaList
              items={items.map((item) => ({
                id: item.id,
                title: item.title,
                subtitle: item.subtitle,
                coverUrl: item.coverUrl,
                onClick: () => play(item),
              }))}
            />
          ) : (
            <MediaGrid>
              {items.map((item) => (
                <BookCard
                  key={item.id}
                  title={item.title}
                  author={item.subtitle}
                  coverUrl={item.coverUrl}
                  kindIcon={icon}
                  formatLabel={item.formatLabel}
                  onClick={() => play(item)}
                />
              ))}
            </MediaGrid>
          )
        }
      </Async>
    </>
  );
}
