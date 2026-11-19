import test from "node:test";
import assert from "node:assert/strict";
import {
  preferredVideoTrack,
  videoLanguageCode,
  videoTimeLabel,
  videoTrackLabel,
} from "../static/video-format.js";

test("video language aliases normalize reader preferences", () => {
  assert.equal(videoLanguageCode(" English "), "eng");
  assert.equal(videoLanguageCode("JA"), "jpn");
  assert.equal(videoLanguageCode("ita"), "ita");
  assert.equal(
    preferredVideoTrack([{ language: "eng" }, { language: "jpn" }], "Japanese")
      .language,
    "jpn",
  );
});

test("video track labels remain informative with a safe fallback", () => {
  assert.equal(
    videoTrackLabel({ title: "Main", language: "eng", codec: "aac" }, "Audio"),
    "Main · ENG · AAC",
  );
  assert.equal(videoTrackLabel({}, "Audio 1"), "Audio 1");
});

test("video time labels cover short and long media", () => {
  assert.equal(videoTimeLabel(65.9), "1:05");
  assert.equal(videoTimeLabel(3661), "1:01:01");
  assert.equal(videoTimeLabel(0), "");
});

test("cleanVideoTitle drops release tags but keeps the year and plain titles", async () => {
  const { cleanVideoTitle } = await import("../static/video-format.js");
  assert.equal(cleanVideoTitle("The World of Kanako (2014) Bluray-1080p"), "The World of Kanako (2014)");
  assert.equal(cleanVideoTitle("Movie Name 2160p HDR"), "Movie Name");
  assert.equal(cleanVideoTitle("Blue Ray of Hope"), "Blue Ray of Hope");
  assert.equal(cleanVideoTitle("1080p"), "1080p");
});
