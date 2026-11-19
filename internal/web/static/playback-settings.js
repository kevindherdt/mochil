import { mountPlaybackSettings } from "./playback-settings-template.js?v=2";

export function normalizePlaybackQuality(value, fallback) {
  return ['maximum', 'high', 'medium', 'low'].includes(value) ? value : fallback
}

export function playbackSettingsValue(audioLanguage, subtitleLanguage, wifiQuality = 'maximum', mobileQuality = 'high') {
  return {
    audio_language: String(audioLanguage || "")
      .trim()
      .toLowerCase(),
    subtitle_language:
      String(subtitleLanguage || "")
        .trim()
        .toLowerCase() || "off",
    wifi_quality: normalizePlaybackQuality(wifiQuality, 'maximum'),
    mobile_quality: normalizePlaybackQuality(mobileQuality, 'high'),
  };
}

export function createPlaybackSettings({ document, $, onDirty = () => {} }) {
  mountPlaybackSettings(document);
  const audio = $("#playback-audio-language");
  const subtitles = $("#playback-subtitle-language");
  const wifiQuality = $("#playback-wifi-quality");
  const mobileQuality = $("#playback-mobile-quality");
  return {
    bind() {
      audio.addEventListener("input", onDirty);
      subtitles.addEventListener("input", onDirty);
      wifiQuality.addEventListener("input", onDirty);
      mobileQuality.addEventListener("input", onDirty);
      wifiQuality.addEventListener("change", () => globalThis.localStorage?.setItem('mochil-playback-wifi-quality', wifiQuality.value));
      mobileQuality.addEventListener("change", () => globalThis.localStorage?.setItem('mochil-playback-mobile-quality', mobileQuality.value));
    },
    load(settings) {
      audio.value = settings?.audio_language || "";
      subtitles.value = settings?.subtitle_language || "off";
      wifiQuality.value = normalizePlaybackQuality(settings?.wifi_quality, 'maximum');
      mobileQuality.value = normalizePlaybackQuality(settings?.mobile_quality, 'high');
      globalThis.localStorage?.setItem('mochil-playback-wifi-quality', wifiQuality.value);
      globalThis.localStorage?.setItem('mochil-playback-mobile-quality', mobileQuality.value);
    },
    write(settings) {
      settings.playback = playbackSettingsValue(audio.value, subtitles.value, wifiQuality.value, mobileQuality.value);
    },
  };
}
