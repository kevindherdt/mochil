// features/library/api/videoProgress.ts — reading/writing a video's resume position.
//
// The item payload from /api/videos already carries `progress`, so opening a screen usually needs nothing extra; this
// module exists for the write side and for re-reading after a save, and it sends `expected_media_revision` so the
// server can refuse a position that belongs to a different version of the file (a replaced episode would otherwise
// resume at a stranger's timestamp).
import { z } from "zod";
import { request } from "../../../lib/http";

const progressSchema = z.object({
  library_id: z.string().optional(),
  item_id: z.string().optional(),
  position_seconds: z.number(),
  duration_seconds: z.number(),
  updated_at: z.number().optional(),
  revision: z.number().optional(),
});

export type VideoProgress = z.infer<typeof progressSchema>;

export async function fetchVideoProgress(libraryId: string, itemId: string): Promise<VideoProgress> {
  const raw = await request<unknown>(
    `/api/video-progress/${encodeURIComponent(libraryId)}/${encodeURIComponent(itemId)}`,
  );
  return progressSchema.parse(raw);
}

export interface ProgressWrite {
  libraryId: string;
  itemId: string;
  positionSeconds: number;
  durationSeconds: number;
  /** The file revision the position belongs to — the server 412s if it no longer matches. */
  mediaRevision?: string;
}

/** A plain checkpoint. The server refuses one that moves the position *backwards* (it is protecting against a stale
 *  device reporting an old timestamp), which is correct for a passive save and wrong for a deliberate seek. */
export async function saveVideoProgress(write: ProgressWrite): Promise<void> {
  await request<unknown>(
    `/api/video-progress/${encodeURIComponent(write.libraryId)}/${encodeURIComponent(write.itemId)}`,
    {
      method: "PUT",
      body: {
        position_seconds: write.positionSeconds,
        duration_seconds: write.durationSeconds,
        ...(write.mediaRevision === undefined ? {} : { expected_media_revision: write.mediaRevision }),
      },
    },
  );
}

/** A deliberate seek — the reader dragged the bar or jumped back. Sends the revision it is replacing plus
 *  `explicit_seek`, which is the only way the server accepts a position earlier than the stored one
 *  (`video.ProgressStore.put`'s monotonic guard). */
export async function saveVideoSeek(write: ProgressWrite): Promise<void> {
  const current = await fetchVideoProgress(write.libraryId, write.itemId);
  await request<unknown>(
    `/api/video-progress/${encodeURIComponent(write.libraryId)}/${encodeURIComponent(write.itemId)}`,
    {
      method: "PUT",
      body: {
        position_seconds: write.positionSeconds,
        duration_seconds: write.durationSeconds,
        expected_revision: current.revision ?? 0,
        explicit_seek: true,
        ...(write.mediaRevision === undefined ? {} : { expected_media_revision: write.mediaRevision }),
      },
    },
  );
}
