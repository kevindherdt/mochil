import assert from "node:assert/strict";
import test from "node:test";
import {
  createPlaybackSettings,
  playbackSettingsValue,
} from "../static/playback-settings.js";
import {
  mountPlaybackSettings,
  playbackSettingsHTML,
} from "../static/playback-settings-template.js";

test("playback values normalize languages and preserve explicit subtitle-off", () => {
  assert.deepEqual(playbackSettingsValue(" Japanese ", " English "), {
    audio_language: "japanese",
    subtitle_language: "english",
    wifi_quality: "maximum",
    mobile_quality: "high",
  });
  assert.deepEqual(playbackSettingsValue("", ""), {
    audio_language: "",
    subtitle_language: "off",
    wifi_quality: "maximum",
    mobile_quality: "high",
  });
});

test("playback template retains language and settings contracts", () => {
  for (const id of [
    "playback-audio-language",
    "playback-subtitle-language",
    "playback-wifi-quality",
    "playback-mobile-quality",
    "media-languages",
    "subtitle-languages",
  ])
    assert.match(playbackSettingsHTML, new RegExp(`id="${id}"`));
  assert.match(playbackSettingsHTML, /data-settings-panel="playback" hidden/);
  assert.match(playbackSettingsHTML, /value="Japanese"/);
  assert.match(playbackSettingsHTML, /value="off"/);
});

test("playback settings mount once inside Settings", () => {
  const insertions = [];
  let mounted = false;
  const settings = {
    insertAdjacentHTML(position, markup) {
      insertions.push({ position, markup });
      mounted = true;
    },
  };
  const document = {
    querySelector(selector) {
      return selector === '[data-settings-panel="playback"]'
        ? mounted
          ? {}
          : null
        : settings;
    },
  };
  mountPlaybackSettings(document);
  mountPlaybackSettings(document);
  assert.deepEqual(insertions, [
    { position: "beforeend", markup: playbackSettingsHTML },
  ]);
});

test("playback controller loads, tracks, and writes its complete slice", () => {
  let dirty = 0;
  const fields = {
    "#playback-audio-language": {
      value: "",
      addEventListener(type, handler) {
        this[type] = handler;
      },
    },
    "#playback-subtitle-language": {
      value: "",
      addEventListener(type, handler) {
        this[type] = handler;
      },
    },
    "#playback-wifi-quality": { value: "", addEventListener(type, handler) { this[type] = handler; } },
    "#playback-mobile-quality": { value: "", addEventListener(type, handler) { this[type] = handler; } },
  };
  const controller = createPlaybackSettings({
    document: { querySelector: () => ({}) },
    $: (selector) => fields[selector],
    onDirty: () => dirty++,
  });
  controller.bind();
  controller.load({ audio_language: "ja", subtitle_language: "nl", wifi_quality: "high", mobile_quality: "low" });
  assert.equal(fields["#playback-audio-language"].value, "ja");
  assert.equal(fields["#playback-subtitle-language"].value, "nl");
  assert.equal(fields["#playback-wifi-quality"].value, "high");
  assert.equal(fields["#playback-mobile-quality"].value, "low");
  fields["#playback-audio-language"].input();
  assert.equal(dirty, 1);
  fields["#playback-audio-language"].value = " English ";
  fields["#playback-subtitle-language"].value = " ";
  fields["#playback-wifi-quality"].value = "medium";
  fields["#playback-mobile-quality"].value = "low";
  const settings = { unrelated: true };
  controller.write(settings);
  assert.deepEqual(settings, {
    unrelated: true,
    playback: { audio_language: "english", subtitle_language: "off", wifi_quality: "medium", mobile_quality: "low" },
  });
});
