import { createContinuousPager } from "./continuous-pager.js?v=4";
import { createBoundedWindow } from "./bounded-window.js?v=1";
import { createVideoMetadataReview } from "./video-metadata-review.js?v=3";
import { mountVideoMetadataMatch } from "./video-metadata-match.js?v=2";
import { createVideoPlayer } from "./video-player.js?v=16";
import { mountVideoLibrary } from "./video-library-template.js?v=2";
import { bindVideoShowPreferences } from "./video-show-preferences.js?v=2";
import { cleanVideoTitle } from "./video-format.js?v=2";
import { bindEpisodeOverview } from "./video-episode-overview.js?v=2";
import { bindTitleDetails, episodeSynopsisHTML, titleDetailsSummaryHTML } from "./video-details-presentation.js?v=1";
import {
  applySeasonItems,
  isCurrentVideoSeasonView,
} from "./video-season-state.js?v=1";

function addArtwork(element, url) {
  if (!element || !url) return;
  const image = new Image();
  image.alt = "";
  image.className = "art-img";
  image.loading = "lazy";
  image.decoding = "async";
  image.fetchPriority = "low";
  image.onerror = () => image.remove();
  image.src = url;
  element.append(image);
}

export function createVideoLibrary({
  document,
  $,
  escapeHTML,
  requestJSON,
  setNavigation = () => {},
  isLibraryVisible = () => true,
}) {
  mountVideoLibrary(document);
  // Artwork fades in when it arrives (load does not bubble, so it is caught on the way down).
  document.addEventListener("load", (event) => { if (event.target?.classList?.contains("art-img")) event.target.classList.add("loaded"); }, true);
  // A poster that is missing is dropped, leaving the quiet placeholder (inline handlers are blocked by the CSP).
  document.addEventListener("error", (event) => { if (event.target?.classList?.contains("art-img")) event.target.remove(); }, true);
  let request;
  let loading = false;
  let activeLibrary = "";
  let activeSearch = "";
  let activeKind = "movies";
  let activeMetadataProvider = "";
  let items = [];
  let itemIDs = new Set();
  let total = 0;
  let startOffset = 0;
  let prepending = false;
  let layout = localStorage.getItem("mochil-video-layout") || "covers";
  const pageSize = 120;
  const videoWindow = createBoundedWindow({
    container: $("#videos"),
    topSpacer: $("#video-top-spacer"),
    bottomSpacer: $("#video-bottom-spacer"),
    maximum: 360,
    trim: pageSize,
  });
  const metadataReview = createVideoMetadataReview({
    requestJSON,
    escapeHTML,
    refresh: () =>
      load(
        activeLibrary,
        activeSearch,
        true,
        activeKind,
        activeMetadataProvider,
      ),
  });
  const playerController = createVideoPlayer({
    document,
    $,
    escapeHTML,
    requestJSON,
  });
  const playGlyph =
    '<svg viewBox="0 0 24 24"><path d="m9 7 8 5-8 5V7Z"/></svg>';
  const artworkMarkup = (url, className = "video-art") =>
    `<span class="${className}" aria-hidden="true">${playGlyph}${url ? `<img class="art-img" src="${escapeHTML(url)}" loading="lazy" decoding="async" fetchpriority="low" alt="">` : ""}</span>`;

  const applyLayout = () => {
    $("#video-layout").value = layout;
    $("#videos").classList.toggle("video-cover-grid", layout === "covers");
    $("#videos").classList.toggle("video-list", layout === "list");
    $("#videos")
      .querySelectorAll(".video-season-episodes")
      .forEach((target) =>
        target.classList.toggle("video-cover-grid", layout === "covers"),
      );
  };

  $("#video-layout").onchange = (event) => {
    layout = event.target.value;
    localStorage.setItem("mochil-video-layout", layout);
    $("#view-menu").open = false;
    applyLayout();
  };

  const sizeLabel = (bytes) => {
    if (!Number.isFinite(bytes) || bytes <= 0) return "";
    const units = ["B", "KB", "MB", "GB", "TB"];
    const unit = Math.min(
      Math.floor(Math.log(bytes) / Math.log(1024)),
      units.length - 1,
    );
    const value = bytes / 1024 ** unit;
    return `${value >= 10 || unit === 0 ? value.toFixed(0) : value.toFixed(1)} ${units[unit]}`;
  };

  const videoMarkup = (item, episodeIndex = null) => {
    const progress = item.progress;
    const fraction =
      progress?.duration_seconds > 0
        ? Math.min(
            100,
            (progress.position_seconds / progress.duration_seconds) * 100,
          )
        : 0;
    const resume =
      fraction > 0 && fraction < 95
        ? `Resume at ${Math.floor(progress.position_seconds / 60)} min`
        : "";
    const episodeCode =
      item.episode > 0
        ? `S${String(item.season).padStart(2, "0")}E${String(item.episode).padStart(2, "0")}`
        : "";
    const displayTitle = item.episode_title || cleanVideoTitle(item.title);
    const playable = item.stream_url || item.hls_url;
    if (episodeIndex !== null) return `<article class="video-item video-episode-card"><button type="button" class="video-episode-select${playable ? ' video-open' : ''}" data-episode-index="${episodeIndex}" ${playable ? `data-item="${escapeHTML(item.id)}" data-title="${escapeHTML(displayTitle)}"` : 'disabled'} aria-label="${playable ? 'Play' : 'Unavailable'} ${escapeHTML(displayTitle)}">${artworkMarkup(item.artwork_url)}<span class="video-copy"><strong title="${escapeHTML(displayTitle)}">${escapeHTML(displayTitle)}</strong><small>${escapeHTML([episodeCode, resume].filter(Boolean).join(" · "))}</small>${fraction > 0 ? `<i class="video-progress" style="--progress:${fraction}%"></i>` : ""}</span></button></article>`;
    return `<article class="video-item">${playable ? `<button class="video-open" data-item="${escapeHTML(item.id)}" data-title="${escapeHTML(displayTitle)}">` : '<span class="video-open video-open--disabled">'}
      ${artworkMarkup(item.artwork_url)}
      <span class="video-copy"><strong title="${escapeHTML(displayTitle)}">${escapeHTML(displayTitle)}</strong><small>${escapeHTML([episodeCode, resume, item.container?.toUpperCase(), sizeLabel(item.size), item.direct_play ? "Direct play" : "Compatibility playback", item.metadata_override ? "TMDB details" : ""].filter(Boolean).join(" · "))}</small>${episodeSynopsisHTML(item, escapeHTML)}${fraction > 0 ? `<i class="video-progress" style="--progress:${fraction}%"></i>` : ""}</span>
      ${playable ? "</button>" : "</span>"}${activeKind === "movies" && activeMetadataProvider === "tmdb" ? `<details class="video-title-details" data-title-details data-item-id="${escapeHTML(item.id)}">${titleDetailsSummaryHTML(displayTitle, escapeHTML)}<div data-title-details-content></div></details>` : ""}${activeMetadataProvider === "tmdb" && activeKind === "movies" ? `<details class="video-more"><summary aria-label="More options for ${escapeHTML(displayTitle)}"><svg viewBox="0 0 24 24" aria-hidden="true"><circle cx="12" cy="5" r="1.4"/><circle cx="12" cy="12" r="1.4"/><circle cx="12" cy="19" r="1.4"/></svg></summary><div><button type="button" class="video-metadata-open" data-metadata-item="${escapeHTML(item.id)}">${item.metadata_override ? "Manage details" : "Find details"}</button></div></details><div class="video-metadata-results" data-metadata-results="${escapeHTML(item.id)}" role="status" hidden></div>` : ""}</article>`;
  };

  const bindVideos = (target) =>
    target.querySelectorAll(".video-open[data-item]").forEach((button) => {
      button.onclick = () =>
        playerController.open(
          items.find((item) => item.id === button.dataset.item),
          button.dataset.title,
        );
    });

  const bindMetadata = (target) =>
    target.querySelectorAll("[data-metadata-item]").forEach((button) => {
      button.addEventListener("click", () => button.closest("details")?.removeAttribute("open"));
      const item = items.find(
        (candidate) => candidate.id === button.dataset.metadataItem,
      );
      if (item)
        metadataReview.bind(button, {
          libraryID: activeLibrary,
          itemID: item.id,
          previous: item.metadata_override,
        });
    });

  const bindDetails = (target, libraryID) => {
    target.querySelectorAll('[data-title-details]').forEach(panel => {
      if (panel.dataset.detailsBound) return;
      panel.dataset.detailsBound = 'true';
      bindTitleDetails(panel, { libraryID, itemID: panel.dataset.itemId, show: panel.dataset.showName, requestJSON, escapeHTML });
    });
  };

  const render = (newItems, mode = "replace") => {
    $("#status").textContent = "";
    const markup = newItems.map(item => videoMarkup(item)).join("");
    if (mode === "replace") {
      videoWindow.reset();
      $("#videos").innerHTML =
        markup || '<p class="empty-state">No supported video files found.</p>';
    } else {
      const insertion = videoWindow.begin(mode);
      $("#videos").insertAdjacentHTML(
        mode === "append" ? "beforeend" : "afterbegin",
        markup,
      );
      const removed = videoWindow.finish(insertion);
      if (removed.start) {
        items.splice(0, removed.start);
        startOffset += removed.start;
      }
      if (removed.end) items.splice(-removed.end, removed.end);
    }
    applyLayout();
    $("#video-load-more").hidden = startOffset + items.length >= total;
    $("#video-load-more").disabled = false;
    $("#video-load-more").textContent = "Load more videos";
    bindVideos($("#videos"));
    bindMetadata($("#videos"));
    bindDetails($("#videos"), activeLibrary);
    setNavigation(
      [...$("#videos").children].map((element) => ({
        element,
        label: element.querySelector("strong")?.textContent || "",
      })),
    );
    pager?.refresh();
    previousPager?.refresh();
  };

  async function loadSeason(libraryID, show, season, target, search, other) {
    const view = { request, items, knownIDs: itemIDs, target };
    target.innerHTML = '<p class="video-season-status">Loading episodes…</p>';
    const query = new URLSearchParams({
      library: libraryID,
      show,
      season,
      limit: 500,
    });
    if (search) query.set("q", search);
    if (other) query.set("other", "1");
    try {
      const result = await requestJSON(`/api/videos?${query}`);
      if (!applySeasonItems(view, request, items, itemIDs, result.data)) return;
      target.innerHTML = result.data.length
        ? `<section class="video-episode-overview" aria-label="Episode preview"><span>Episode preview</span><h3 data-episode-title></h3><p data-episode-description></p><button type="button" data-episode-toggle aria-expanded="false" aria-label="Show full description" class="video-episode-toggle" hidden><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M6 9l6 6 6-6"/></svg></button></section>${result.data.map((item, index) => videoMarkup(item, index)).join("")}`
        : '<p class="empty-state">No episodes found.</p>';
      if (result.data.length) bindEpisodeOverview(target, result.data.map(item => ({...item, displayTitle: item.episode_title || cleanVideoTitle(item.title)})));
      target.classList.toggle("video-cover-grid", layout === "covers");
      bindVideos(target);
      bindMetadata(target);
    } catch (error) {
      if (!isCurrentVideoSeasonView(view, request, items, itemIDs)) return;
      // Clearing the guard lets the same season be reopened instead of staying broken.
      delete details.dataset.loaded;
      target.innerHTML = `<p class="empty-state" role="alert">Could not load this season: ${escapeHTML(error.message)}</p>`;
    }
  }

  async function loadShows(libraryID, search) {
    request?.abort();
    items = [];
    itemIDs = new Set();
    startOffset = 0;
    videoWindow.reset();
    $("#videos").innerHTML = "";
    $("#video-load-previous").hidden = true;
    $("#video-load-more").hidden = true;
    $("#status").textContent = "Opening shows…";
    const current = new AbortController();
    request = current;
    const query = new URLSearchParams({ library: libraryID });
    if (search) query.set("q", search);
    try {
      const result = await requestJSON(`/api/video-shows?${query}`, {
        signal: current.signal,
      });
      if (request !== current) return;
      $("#status").textContent =
        `${result.data.length.toLocaleString()} ${result.data.length === 1 ? "show" : "shows"}`;
      $("#videos").innerHTML =
        result.data
          .map(
            (show) =>
              `<details class="video-show"><summary><span class="video-show-art" aria-hidden="true"><svg viewBox="0 0 24 24"><path d="m9 7 8 5-8 5V7Z"/></svg></span><span class="video-show-copy"><strong>${escapeHTML(show.name)}</strong><small>${show.count} ${show.count === 1 ? "episode" : "episodes"}${show.other ? "" : ` · ${show.seasons.length} ${show.seasons.length === 1 ? "season" : "seasons"}${show.year ? ` · ${show.year}` : ""}${show.metadata_override ? " · TMDB details" : ""}`}</small>${show.overview ? `<p class="video-show-overview">${escapeHTML(show.overview)}</p>` : ""}</span><svg class="video-disclosure" viewBox="0 0 24 24" aria-hidden="true"><path d="m8 10 4 4 4-4"/></svg></summary><div class="video-seasons">${show.other ? "" : `${activeMetadataProvider === "tmdb" ? `<details class="video-title-details" data-title-details data-show-name="${escapeHTML(show.source_name)}">${titleDetailsSummaryHTML(show.name, escapeHTML)}<div data-title-details-content></div></details>` : ""}<div class="video-show-metadata"><button type="button" class="video-metadata-open" data-show-metadata>${show.metadata_override ? "Manage show details" : "Find show details"}</button><div class="video-metadata-results" data-metadata-results role="status" hidden></div></div><details class="show-playback" data-show="${escapeHTML(show.source_name)}"><summary>Playback for this show</summary><div><label>Audio<select data-show-audio><option value="">Use default</option></select></label><label>Subtitles<select data-show-subtitles><option value="">Use default</option><option value="off">Off</option></select></label><button type="button">Save</button><small role="status"></small></div></details>`}${show.seasons.map((season) => `<details class="video-season" data-show="${show.other ? "" : escapeHTML(show.source_name)}" data-other="${show.other ? "1" : ""}" data-season="${season.number}"><summary><span class="video-season-art" aria-hidden="true"></span><span>${show.other ? "Unmatched files" : season.number === 0 ? "Specials" : `Season ${season.number}`}</span><small>${season.count} ${season.count === 1 ? "episode" : "episodes"}</small><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m8 10 4 4 4-4"/></svg></summary><div class="video-season-episodes"></div></details>`).join("")}</div></details>`,
          )
          .join("") || '<p class="empty-state">No shows found.</p>';
      $("#videos")
        .querySelectorAll(".video-show-art")
        .forEach((art, index) =>
          addArtwork(art, result.data[index]?.artwork_url),
        );
      $("#videos")
        .querySelectorAll(".video-show")
        .forEach((element, showIndex) =>
          element
            .querySelectorAll(".video-season-art")
            .forEach((art, seasonIndex) =>
              addArtwork(
                art,
                result.data[showIndex]?.seasons?.[seasonIndex]?.artwork_url,
              ),
            ),
        );
      setNavigation(
        [...$("#videos").children].map((element) => ({
          element,
          label:
            element.querySelector(".video-show-copy strong")?.textContent || "",
        })),
      );
      applyLayout();
      $("#videos")
        .querySelectorAll(".video-show")
        .forEach((details, index) => {
          const show = result.data[index];
          const button = details.querySelector("[data-show-metadata]");
          if (button)
            metadataReview.bind(button, {
              libraryID,
              itemID: show.item_id,
              show: show.source_name,
              previous: show.metadata_override,
            });
        });
      $("#videos")
        .querySelectorAll(".video-season")
        .forEach((details) =>
          details.addEventListener("toggle", () => {
            if (!details.open || details.dataset.loaded) return;
            details.dataset.loaded = "true";
            loadSeason(
              libraryID,
              details.dataset.show,
              details.dataset.season,
              details.querySelector(".video-season-episodes"),
              search,
              details.dataset.other === "1",
            );
          }),
        );
      $("#videos")
        .querySelectorAll(".video-season>summary")
        .forEach((summary) =>
          summary.addEventListener("click", (event) => event.stopPropagation()),
        );
      bindDetails($("#videos"), libraryID);
      bindVideoShowPreferences({
        root: $("#videos"),
        libraryID,
        requestJSON,
        escapeHTML,
      });
    } catch (error) {
      if (error.name !== "AbortError")
        $("#status").textContent = `Could not open shows: ${error.message}`;
    }
  }

  async function load(
    libraryID,
    search,
    reset = true,
    libraryKind = "movies",
    metadataProvider = activeMetadataProvider,
  ) {
    if (!libraryID) return;
    if (loading && !reset) return;
    activeLibrary = libraryID;
    activeSearch = search || "";
    activeKind = libraryKind;
    activeMetadataProvider = metadataProvider;
    if (reset && metadataProvider === "tmdb")
      mountVideoMetadataMatch({
        host: $("#videos"),
        libraryID,
        requestJSON,
        escapeHTML,
        refresh: () => load(libraryID, activeSearch, true, activeKind, activeMetadataProvider),
      });
    if (reset && (libraryKind === "tv" || libraryKind === "anime"))
      return loadShows(libraryID, search);
    if (reset) {
      request?.abort();
      items = [];
      itemIDs = new Set();
      total = 0;
      startOffset = 0;
    }
    const current = new AbortController();
    request = current;
    loading = true;
    if (reset) {
      $("#videos").innerHTML = "";
      $("#status").textContent = "Opening the video shelf…";
    }
    const query = new URLSearchParams({
      library: libraryID,
      limit: pageSize,
      offset: startOffset + items.length,
    });
    if (search) query.set("q", search);
    try {
      const result = await requestJSON(`/api/videos?${query}`, {
        signal: current.signal,
      });
      if (request !== current) return;
      total =
        Number(result.response.headers.get("X-Total-Count")) ||
        result.data.length;
      items.push(...result.data);
      render(result.data, reset ? "replace" : "append");
    } catch (error) {
      if (error.name !== "AbortError") {
        $("#status").textContent =
          `Could not open the video shelf: ${error.message}`;
        return false;
      }
    } finally {
      if (request === current) loading = false;
      pager?.refresh();
      previousPager?.refresh();
    }
    return true;
  }

  async function loadPrevious() {
    if (loading || prepending || startOffset <= 0) return false;
    prepending = true;
    const offset = Math.max(0, startOffset - pageSize);
    const current = new AbortController();
    request = current;
    loading = true;
    try {
      const query = new URLSearchParams({
        library: activeLibrary,
        limit: startOffset - offset,
        offset,
      });
      if (activeSearch) query.set("q", activeSearch);
      const result = await requestJSON(`/api/videos?${query}`, {
        signal: current.signal,
      });
      if (request !== current || !result.data.length) return false;
      startOffset = offset;
      items.unshift(...result.data);
      render(result.data, "prepend");
      return true;
    } catch (error) {
      if (error.name !== "AbortError")
        $("#status").textContent = `Could not load earlier videos: ${error.message}`;
      return false;
    } finally {
      if (request === current) loading = false;
      prepending = false;
      previousPager?.refresh();
    }
  }

  const pager = createContinuousPager({
    sentinel: $("#video-sentinel"),
    button: $("#video-load-more"),
    label: "videos",
    canLoad: () =>
      !$("#video-index").hidden &&
      activeKind === "movies" &&
      !loading &&
      items.length < total,
    load: () =>
      load(
        activeLibrary,
        activeSearch,
        false,
        activeKind,
        activeMetadataProvider,
      ),
  });
  const previousPager = createContinuousPager({
    sentinel: $("#video-start-sentinel"),
    button: $("#video-load-previous"),
    label: "previous videos",
    rootMargin: "500px",
    canLoad: () =>
      !$("#video-index").hidden &&
      activeKind === "movies" &&
      !loading &&
      !prepending &&
      startOffset > 0,
    load: loadPrevious,
  });
  return { load, open: playerController.open, abort: () => { request?.abort(); playerController.abort(); } };
}
