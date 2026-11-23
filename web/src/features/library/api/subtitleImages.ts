// api/subtitleImages.ts — image-based (PGS) subtitle cues for the video player.
//
// A remux's subtitle tracks are bitmaps, not text, so they cannot be WebVTT and the player cannot hand them to a
// <track>. The server decodes the chosen track once into PNG cues plus a manifest (internal/video/subtitle_images.go)
// and this fetches that manifest. The first call for an uncached track answers 202 while the extraction runs, so the
// query keeps asking until it is ready — a two-hour film takes about a minute, and there is nothing to show until it
// finishes.
import { z } from "zod";

const cueSchema = z.looseObject({
  start: z.number(),
  end: z.number(),
  file: z.string(),
  // The caption's box inside the frame. Sizing scales the bitmap about this box, so the caption grows where it sits.
  x: z.number(),
  y: z.number(),
  w: z.number(),
  h: z.number(),
});

const manifestSchema = z.looseObject({
  width: z.number(),
  height: z.number(),
  cues: z.array(cueSchema),
});

export interface ImageSubtitleCue {
  start: number;
  end: number;
  file: string;
  x: number;
  y: number;
  w: number;
  h: number;
}

export interface ImageSubtitleManifest {
  width: number;
  height: number;
  cues: ImageSubtitleCue[];
}

/** `"preparing"` while the server is still extracting this track. */
export type SubtitleManifestResult = ImageSubtitleManifest | "preparing";

/** The cue covering `position`, by binary search — a two-hour film has ~1500 of them and the player asks four times a
    second, so a linear scan would be the wrong shape. */
export function cueAt(cues: ImageSubtitleCue[], position: number): ImageSubtitleCue | null {
  let low = 0;
  let high = cues.length - 1;
  while (low <= high) {
    const middle = (low + high) >> 1;
    const cue = cues[middle];
    if (cue === undefined) return null;
    if (position < cue.start) high = middle - 1;
    else if (position > cue.end) low = middle + 1;
    else return cue;
  }
  return null;
}

export function subtitleCueUrl(libraryId: string, itemId: string, streamIndex: number, file: string): string {
  return `/api/video-subtitles/${encodeURIComponent(libraryId)}/${encodeURIComponent(itemId)}/${streamIndex}/${encodeURIComponent(file)}`;
}

export async function fetchSubtitleManifest(
  libraryId: string,
  itemId: string,
  streamIndex: number,
  signal?: AbortSignal,
): Promise<SubtitleManifestResult> {
  const url = `/api/video-subtitles/${encodeURIComponent(libraryId)}/${encodeURIComponent(itemId)}/${streamIndex}/manifest.json`;
  const response = await fetch(url, { credentials: "include", ...(signal === undefined ? {} : { signal }) });
  if (response.status === 202) return "preparing";
  if (!response.ok) throw new Error("This subtitle track cannot be read.");
  return manifestSchema.parse(await response.json());
}
