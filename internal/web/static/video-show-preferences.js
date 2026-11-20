import {
  videoLanguageCode as languageCode,
  videoTrackLabel as trackLabel,
} from "./video-format.js?v=2";

export function videoLanguageOptions(episodes) {
  const audio = new Map();
  const subtitles = new Map();
  episodes.forEach((item) => {
    (item.audio_streams || []).forEach((track) => {
      if (track.language)
        audio.set(languageCode(track.language), trackLabel(track, track.language));
    });
    (item.subtitle_streams || [])
      .filter((track) => track.text)
      .forEach((track) => {
        if (track.language)
          subtitles.set(
            languageCode(track.language),
            trackLabel(track, track.language),
          );
      });
  });
  const sorted = (values) =>
    [...values].sort((left, right) => left[1].localeCompare(right[1]));
  return { audio: sorted(audio), subtitles: sorted(subtitles) };
}

export function bindVideoShowPreferences({
  root,
  libraryID,
  requestJSON,
  escapeHTML,
}) {
  root.querySelectorAll(".show-playback").forEach((details) =>
    details.addEventListener("toggle", async () => {
      if (!details.open || details.dataset.loaded) return;
      details.dataset.loaded = "true";
      const query = new URLSearchParams({
        library: libraryID,
        show: details.dataset.show,
      });
      const status = details.querySelector("[role=status]");
      try {
        const [preferenceResult, episodesResult] = await Promise.all([
          requestJSON(`/api/video-preferences?${query}`),
          requestJSON(
            `/api/videos?${new URLSearchParams({ library: libraryID, show: details.dataset.show, limit: 500 })}`,
          ),
        ]);
        const preference = preferenceResult.data.override;
        const options = videoLanguageOptions(episodesResult.data);
        const audio = details.querySelector("[data-show-audio]");
        const subtitles = details.querySelector("[data-show-subtitles]");
        audio.insertAdjacentHTML("beforeend", optionMarkup(options.audio, escapeHTML));
        subtitles.insertAdjacentHTML(
          "beforeend",
          optionMarkup(options.subtitles, escapeHTML),
        );
        audio.value = languageCode(preference.audio_language);
        subtitles.value =
          preference.subtitle_language === "off"
            ? "off"
            : languageCode(preference.subtitle_language);
      } catch (error) {
        status.textContent = error.message;
      }
      details.querySelector("button").onclick = () =>
        saveVideoShowPreferences(details, query, status, requestJSON);
    }),
  );
}

function optionMarkup(options, escapeHTML) {
  return options
    .map(
      ([value, label]) =>
        `<option value="${escapeHTML(value)}">${escapeHTML(label)}</option>`,
    )
    .join("");
}

async function saveVideoShowPreferences(details, query, status, requestJSON) {
  status.textContent = "Saving…";
  try {
    await requestJSON(`/api/video-preferences?${query}`, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        audio_language: details.querySelector("[data-show-audio]").value,
        subtitle_language: details.querySelector("[data-show-subtitles]").value,
      }),
    });
    status.textContent = "Saved for this show.";
  } catch (error) {
    status.textContent = error.message;
  }
}
