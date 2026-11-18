import assert from "node:assert/strict";
import test from "node:test";
import { videoLanguageOptions } from "../static/video-show-preferences.js";

test("show playback options deduplicate, normalize, and sort languages", () => {
  const options = videoLanguageOptions([
    {
      audio_streams: [
        { language: "ja", title: "Japanese" },
        { language: "jpn", title: "Original" },
        { language: "en", title: "English" },
      ],
      subtitle_streams: [
        { language: "en", title: "Signs", text: true },
        { language: "ja", title: "Image subtitles", text: false },
      ],
    },
  ]);

  assert.deepEqual(options.audio, [
    ["eng", "English · EN"],
    ["jpn", "Original · JPN"],
  ]);
  assert.deepEqual(options.subtitles, [["eng", "Signs · EN"]]);
});
