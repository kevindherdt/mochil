// features/library/AudioGroupList.tsx — "Albums"/"Artists" (music) and "Series"/"Authors" (audiobooks): the audio
// equivalent of LibraryScreen's existing series/authors browse list, same GroupCard, same Show-more pattern.
// Kept separate from AudioGroupTracks.tsx on purpose: a group (an album, a series) and a group's tracks are two
// different screens with two different click behaviours (drill in vs. play), and folding them into one component
// with an `isTrackList` flag is exactly the kind of branch that grows a third state nobody asked for.
import { Async, Button, GroupCard, Inline, MediaGrid, MediaList, Stack } from "../../ui";
import { UserRound } from "lucide-react";
import { useAudioGroups } from "./useAudioGroups";
import type { AudioGroupKind } from "./api/audioGroups";

export interface AudioGroupListProps {
  libraryId: string | undefined;
  groupType: AudioGroupKind;
  q: string;
  layout: "grid" | "list";
  onOpen: (groupId: string, groupName: string) => void;
}

export function AudioGroupList({ libraryId, groupType, q, layout, onOpen }: AudioGroupListProps) {
  const groups = useAudioGroups(libraryId, groupType, q);
  // Series/authors (audiobooks) read as books — 2:3 covers, a pile for a series with several volumes. Albums and
  // artists are music — square artwork, one cover per group, same as MusicCard.
  const bookLike = groupType === "series" || groupType === "authors";
  const noun = groupType === "albums" ? "track" : groupType === "artists" ? "track" : "book";

  return (
    <Stack gap={12}>
      <Async
        query={groups.query}
        empty={(items) => items.length === 0}
        emptyMessage={`No ${groupType} in this shelf.`}
      >
        {(items) =>
          layout === "list" ? (
            <MediaList
              items={items.map((group) => ({
                id: group.id,
                title: group.name,
                subtitle: group.subtitle !== "" ? `${group.subtitle} · ${group.count} ${noun}${group.count === 1 ? "" : "s"}` : `${group.count} ${noun}${group.count === 1 ? "" : "s"}`,
                coverUrl: group.coverUrl ?? "",
                onClick: () => onOpen(group.id, group.name),
              }))}
            />
          ) : (
            <MediaGrid>
              {items.map((group) => (
                <GroupCard
                  key={group.id}
                  name={group.name}
                  count={group.count}
                  coverUrl={group.coverUrl}
                  ratio={bookLike ? "2x3" : "1x1"}
                  stack={groupType === "series"}
                  kindIcon={groupType === "authors" || groupType === "artists" ? UserRound : undefined}
                  countLabel={`${group.count} ${noun}${group.count === 1 ? "" : "s"}`}
                  onClick={() => onOpen(group.id, group.name)}
                />
              ))}
            </MediaGrid>
          )
        }
      </Async>
      {groups.hasMore && (
        <Inline gap={8}>
          <Button variant="secondary" onClick={groups.loadMore} disabled={groups.isLoadingMore}>
            {groups.isLoadingMore ? "Loading…" : "Show more"}
          </Button>
        </Inline>
      )}
    </Stack>
  );
}
