// features/library/ImageCaptions.tsx — bitmap subtitles (PGS/VobSub) drawn over the picture.
//
// A remux's subtitles are images, so they cannot become WebVTT and a <track> cannot carry them. The server decodes the
// chosen track into cue PNGs (internal/video/subtitle_images.go); this shows whichever cue covers the current
// position. Nothing is scaled by hand: the cue frame is the same size as the video, so drawing the <img> with the same
// inset and `object-fit: contain` it puts the caption exactly where the studio placed it, letterbox included.
//
// Until the extraction finishes the server answers "preparing" and this polls; there is nothing to show before that,
// and the menu item is already selectable, so the caption appears mid-film the moment the track is ready.
import { useCallback, useRef, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { cueAt, fetchSubtitleManifest, subtitleCueUrl } from "./api/subtitleImages";

export interface ImageCaptionsProps {
  libraryId: string;
  itemId: string;
  streamIndex: number;
  /** Seconds of playback. */
  position: number;
}

export function ImageCaptions({ libraryId, itemId, streamIndex, position }: ImageCaptionsProps) {
  // The cue bitmap is the whole frame, and its only ink is the caption — so the caption can be resized by scaling the
  // image about the caption's own bottom centre. That keeps it where the studio put it instead of drifting, and it is
  // why the manifest carries a box per cue. `--cue-scale` is the same value the caption-appearance sheet writes for
  // text cues, so "Large" now means Large for both kinds.
  // Measured from a callback ref, not an effect: this component renders nothing until the manifest arrives, so on
  // mount there is no <img> to measure and an effect with `[]` deps would never run again — which is exactly how the
  // transform origin first shipped stuck at the default centre. A callback ref fires when the element really exists,
  // and again for each new cue, since the <img> is keyed by file.
  const [frame, setFrame] = useState({ width: 0, height: 0 });
  const observer = useRef<ResizeObserver | null>(null);
  const attach = useCallback((element: HTMLImageElement | null) => {
    observer.current?.disconnect();
    observer.current = null;
    if (element === null) return;
    const measure = () =>
      setFrame((previous) =>
        previous.width === element.clientWidth && previous.height === element.clientHeight
          ? previous
          : { width: element.clientWidth, height: element.clientHeight },
      );
    measure();
    observer.current = new ResizeObserver(measure);
    observer.current.observe(element);
  }, []);
  const query = useQuery({
    queryKey: ["subtitle-images", libraryId, itemId, streamIndex],
    queryFn: ({ signal }) => fetchSubtitleManifest(libraryId, itemId, streamIndex, signal),
    // Keep asking while the server is still extracting; then never again for this track.
    refetchInterval: (reference) => (reference.state.data === "preparing" ? 4_000 : false),
    staleTime: 10 * 60_000,
    retry: 1,
  });

  const manifest = query.data;
  if (manifest === undefined || manifest === "preparing") return null;
  const cue = cueAt(manifest.cues, position);
  if (cue === null) return null;

  // The image is letterboxed inside the player box exactly like the video, so the caption's origin has to be mapped
  // from video pixels into box pixels before it can be used as a transform origin.
  const origin = (() => {
    if (frame.width === 0 || frame.height === 0 || manifest.width === 0 || manifest.height === 0) return undefined;
    const scale = Math.min(frame.width / manifest.width, frame.height / manifest.height);
    const offsetX = (frame.width - manifest.width * scale) / 2;
    const offsetY = (frame.height - manifest.height * scale) / 2;
    return {
      x: offsetX + (cue.x + cue.w / 2) * scale,
      y: offsetY + (cue.y + cue.h) * scale,
    };
  })();

  return (
    <img
      // The src changes per cue; the key keeps React from animating a stale image into place.
      key={cue.file}
      ref={attach}
      src={subtitleCueUrl(libraryId, itemId, streamIndex, cue.file)}
      alt=""
      aria-hidden="true"
      style={{
        position: "absolute",
        inset: 0,
        width: "100%",
        height: "100%",
        objectFit: "contain",
        pointerEvents: "none",
        transform: "scale(var(--cue-scale, 1))",
        ...(origin === undefined ? {} : { transformOrigin: `${origin.x}px ${origin.y}px` }),
      }}
    />
  );
}
