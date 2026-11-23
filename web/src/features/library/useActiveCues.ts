// features/library/useActiveCues.ts — the caption lines that are on screen right now.
//
// Why this exists at all: the browser's own rendering of a <track> (`::cue`) cannot be trusted with the things Kevin
// asked for. Measured in Chromium on the dev box — `color` and `background-color` do apply, but custom properties do
// not reach the cue pseudo-element (so a settings panel cannot restyle them without rewriting a stylesheet), `font-size`
// is inconsistent across engines, there is no supported way to move a cue, and the cue is positioned against the
// video *element* box rather than its letterboxed contents, so on a portrait phone it lands exactly where our transport
// row is. Every player that takes captions seriously draws them itself; this is that, without a subtitle parser:
//
// The track is set to "hidden", which stops the browser painting cues while keeping `cues` and `activeCues` populated,
// and this hook reports the active ones on `cuechange`. Plain paragraphs of text, in the order the cues appear.
import { useEffect, useState } from "react";

/** Strips the WebVTT inline markup (<i>, <b>, <c.yellow>, <v Speaker>, timestamps) and drops empty cues. */
function plainText(cue: VTTCue): string {
  return cue.text
    .replace(/<[^>]*>/g, "")
    .replace(/&amp;/g, "&")
    .replace(/&lt;/g, "<")
    .replace(/&gt;/g, ">")
    .replace(/&nbsp;/g, " ")
    .trim();
}

export function useActiveCues(trackIndex: number | undefined): string[] {
  // The lines are tagged with the track they came from, so switching subtitles off (or to another language) shows
  // nothing until the new track reports — no effect-body setState, and no stale captions in between.
  const [state, setState] = useState<{ track: number | undefined; lines: string[] }>({ track: undefined, lines: [] });

  useEffect(() => {
    if (trackIndex === undefined) return;
    const video = document.querySelector("video");
    const tracks = video?.textTracks;
    if (tracks === undefined || tracks === null) return;

    let track: TextTrack | undefined;
    for (let i = 0; i < tracks.length; i += 1) {
      const candidate = tracks[i];
      if (candidate === undefined || candidate === null) continue;
      if (candidate.id === String(trackIndex)) track = candidate;
      // Native rendering is off for every track: the browser's cue layer is what we are replacing.
      candidate.mode = candidate.id === String(trackIndex) ? "hidden" : "disabled";
    }
    if (track === undefined) return;

    function read() {
      const active = track?.activeCues;
      if (active === undefined || active === null) {
        setState({ track: trackIndex, lines: [] });
        return;
      }
      const texts: string[] = [];
      for (let i = 0; i < active.length; i += 1) {
        const entry = active[i] as VTTCue | undefined;
        if (entry === undefined || typeof entry.text !== "string") continue;
        const text = plainText(entry);
        if (text !== "") texts.push(text);
      }
      setState({ track: trackIndex, lines: texts });
    }

    read();
    track.addEventListener("cuechange", read);
    // Seeking can cross several cues between two `cuechange` events in some engines; `seeked` catches that.
    video?.addEventListener("seeked", read);
    return () => {
      track?.removeEventListener("cuechange", read);
      video?.removeEventListener("seeked", read);
    };
  }, [trackIndex]);

  return state.track === trackIndex ? state.lines : [];
}
