const labels = {
  unknown_artist: "Unknown artists",
  duplicate_track: "Duplicate tracks",
  duplicate_file_candidate: "Likely file copies",
  track_identity_conflict: "Track identity conflicts",
  inconsistent_album_tags: "Inconsistent albums",
  missing_artwork: "No known artwork",
};

function compactPath(path) {
  return String(path || "").split("/").filter(Boolean).slice(-2).join("/");
}

export function musicQualityView(report, escapeHTML, filter = "all") {
  const counts = Object.entries(report.counts || {}).filter(([, count]) => count > 0);
  const summary = [
    `<span><strong>${Number(report.tracks || 0).toLocaleString()}</strong>Tracks</span>`,
    `<span><strong>${Number(report.albums || 0).toLocaleString()}</strong>Album folders</span>`,
    ...counts.map(([code, count]) => `<span><strong>${count.toLocaleString()}</strong>${escapeHTML(labels[code] || code)}</span>`),
  ].join("");
  const matching = (report.issues || []).filter(issue => filter === "all" || issue.code === filter);
  const issues = matching.map(issue => {
    const fullPaths = (issue.paths || []).join(" · ");
    const paths = (issue.paths || []).map(compactPath).join(" · ");
    return `<article><div class="metadata-book"><strong>${escapeHTML(issue.title || "Untitled")}</strong><small title="${escapeHTML(fullPaths)}">${escapeHTML(paths)}</small></div><span>${escapeHTML(labels[issue.code] || issue.code)}</span><p>${escapeHTML(issue.detail || "")}</p></article>`;
  }).join("");
  const empty = (report.issues || []).length ? "No findings match this category." : "No obvious music tag problems found.";
  return { summary, issues: issues || `<p class="empty-state">${empty}</p>` };
}

export function createMusicQualityReview({ $, escapeHTML, requestJSON }) {
  let report;
  function render() {
    if (!report) return;
    const markup = musicQualityView(report, escapeHTML, $("#music-quality-filter").value);
    $("#metadata-summary").hidden = false;
    $("#metadata-summary").innerHTML = markup.summary;
    $("#metadata-issues").innerHTML = markup.issues;
  }
  async function inspect(libraryID) {
    report = (await requestJSON(`/api/metadata/music-quality?library=${encodeURIComponent(libraryID)}`)).data;
    const categories = Object.entries(report.counts || {}).filter(([, count]) => count > 0);
    const filter = $("#music-quality-filter");
    filter.innerHTML = '<option value="all">All findings</option>' + categories.map(([code, count]) => `<option value="${escapeHTML(code)}">${escapeHTML(labels[code] || code)} (${count.toLocaleString()})</option>`).join("");
    filter.value = categories[0]?.[0] || "all";
    $("#music-quality-filter-wrap").hidden = categories.length === 0;
    render();
    $("#metadata-status").textContent = `${Number(report.tracks || 0).toLocaleString()} tracks checked · embedded-artwork cache and folder sidecars checked without decoding media files.`;
  }
  function reset() {
    report = undefined;
    $("#music-quality-filter-wrap").hidden = true;
  }
  return { inspect, render, reset };
}
