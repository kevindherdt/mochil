export function ingestPreviewItemHTML(item, escapeHTML) {
  const changes = (item.changes || []).map(change => `<li><span><strong>${escapeHTML(change.field.replaceAll("_", " "))}</strong><small>${escapeHTML(change.source)} · ${escapeHTML(change.confidence)} confidence</small></span><b>${escapeHTML(change.proposed)}</b></li>`).join("");
  const fallback = String(item.path || "").split("/").pop() || "Untitled";
  return `<article class="ingest-preview-item"><header><strong>${escapeHTML(item.title || fallback)}</strong><small>${escapeHTML(item.path || "")}</small></header><ul>${changes}</ul></article>`;
}

export function createIngestPreview({ $, escapeHTML, requestJSON, IntersectionObserverClass = globalThis.IntersectionObserver }) {
  let offset = 0;
  let loading = false;
  let observer;
  function reset() {
    offset = 0;
    $("#ingest-preview-results").innerHTML = "";
    $("#ingest-preview-status").textContent = "";
    $("#ingest-preview-more").hidden = true;
    observer?.disconnect();
  }
  async function load(resetFirst = false) {
    if (loading) return;
    if (resetFirst) reset();
    const libraryID = $("#metadata-library").value;
    if (!libraryID) return;
    loading = true;
    observer?.disconnect();
    $("#preview-ingest").disabled = true;
    $("#ingest-preview-status").textContent = offset ? "Loading more proposals…" : "Inspecting structured paths…";
    try {
      const preview = (await requestJSON(`/api/metadata/ingest-preview?library=${encodeURIComponent(libraryID)}&offset=${offset}&limit=100`)).data;
      $("#ingest-preview-results").insertAdjacentHTML("beforeend", (preview.items || []).map(item => ingestPreviewItemHTML(item, escapeHTML)).join(""));
      offset = preview.next_offset || 0;
      const shown = $("#ingest-preview-results").children.length;
      $("#ingest-preview-status").textContent = `${preview.inspected.toLocaleString()} items checked · ${preview.matched.toLocaleString()} have safe fill-in proposals${preview.matched ? ` · ${shown.toLocaleString()} shown` : ""}`;
      $("#ingest-preview-more").hidden = !preview.next_offset;
      if (preview.next_offset && IntersectionObserverClass) {
        observer = new IntersectionObserverClass(entries => { if (entries.some(entry => entry.isIntersecting)) void load(); }, { rootMargin: "500px 0px" });
        observer.observe($("#ingest-preview-sentinel"));
        $("#ingest-preview-more").hidden = true;
      }
    } catch (error) {
      $("#ingest-preview-status").textContent = `Could not preview this shelf: ${error.message}`;
      $("#ingest-preview-more").hidden = offset === 0;
    } finally {
      loading = false;
      $("#preview-ingest").disabled = false;
    }
  }
  return { load, reset };
}
