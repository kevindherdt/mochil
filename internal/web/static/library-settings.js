import { mountLibraryFormTemplate } from "./library-form-template.js?v=8";
import { attachFolderBrowse } from "./folder-picker.js?v=4";
import { iconForKind } from "./settings-icons.js?v=1";
import {
  defaultPreferredFormats,
  normalizePreferredFormats,
  pathGet,
  pathSet,
  sameFormats,
} from "./library-form-model.js?v=1";
import { metadataLineText } from "./library-metadata.js?v=1";
import { inspectFolderLibrary } from "./library-inspect.js?v=1";
import { createLibraryOrderPanel } from "./library-order.js?v=1";
import { libraryHue } from "./library-identity.js?v=2";

// Re-exported so the settings page and its tests keep importing one module.
export {
  defaultPreferredFormats,
  normalizePreferredFormats,
} from "./library-form-model.js?v=1";
export {
  derivedMetadataProvider,
  effectiveMetadataProvider,
  metadataLineText,
} from "./library-metadata.js?v=1";
export {
  inspectionEvidence,
  inspectFolderLibrary,
  libraryFromInspection,
  libraryNameFromPath,
} from "./library-inspect.js?v=1";

// Calibre is a book-library format; it has no meaning for anything that isn't
// a publication (movies, TV, music, etc. are never "a Calibre library"), so the
// Storage choice itself is hidden for these kinds rather than merely narrowed.
export const MEDIA_KINDS = [
  "movies",
  "tv",
  "anime",
  "audiobooks",
  "podcasts",
  "music",
  "documents",
  "vault",
];

export function applyLibraryKind(library, kind) {
  const previousKind = library.kind;
  const previousFormats = [...(library.preferred_formats || [])];
  library.kind = kind;
  if (sameFormats(previousFormats, defaultPreferredFormats(previousKind)))
    library.preferred_formats = defaultPreferredFormats(kind);
  // A content change can make a saved provider impossible. This field is no
  // longer in the form, so a stale name would fail validation on save and there
  // would be no control to clear it: drop it and let the server derive again.
  delete library.metadata_provider;
  if (kind === "manga" && library.inference.folder_layout === "auto")
    library.inference.folder_layout = "series";
  if (MEDIA_KINDS.includes(kind) && library.source.type === "calibre")
    library.source.type = "filesystem";
}

export { newLibrary } from "./library-form-model.js?v=1";

const pendingIdPrefix = "new-library-";

// Library ids are an internal key, so they are derived from the name once and
// then never change; users never see or type them.
export function libraryIdFromName(name, taken = new Set()) {
  let base = String(name || "")
    .normalize("NFKD")
    .replace(/[\u0300-\u036f]/g, "")
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 56)
    .replace(/-+$/, "");
  if (!base) base = "library";
  let id = base;
  for (let n = 2; taken.has(id); n++) id = `${base}-${n}`;
  return id;
}

// A first install has one obvious next step, so say it in plain words instead of leaving the reader
// to work out whether "Calibre", "Local folder" or "vault" is the thing they want.
export function shouldShowFirstLibraryHint(libraries) {
  return Array.isArray(libraries) && libraries.length === 0;
}

function kindLabel(kind = "books") {
  return kind.charAt(0).toUpperCase() + kind.slice(1);
}

function storageLabel(type) {
  if (type === "calibre") return "Calibre";
  return "Local folder";
}

function renderSource(card, type) {
  card.querySelectorAll("[data-source]").forEach((field) => {
    const active = field.dataset.source === type;
    field.hidden = !active;
    field.querySelectorAll("input,select").forEach((input) => {
      input.disabled = !active;
    });
  });
}

// Preferred formats and folder layout only apply to publications, so the nested
// Advanced section would otherwise sit open and empty for a video library.
function renderAdvancedGroup(card) {
  const group = card.querySelector(".library-advanced-more");
  if (!group) return;
  const visible = [...group.querySelectorAll("[data-publication-only]")].some(
    (field) => !field.hidden,
  );
  group.hidden = !visible;
  if (!visible) group.open = false;
}

function renderContentKind(card, kind, library) {
  const media = MEDIA_KINDS.includes(kind);
  const storage = card.querySelector('[data-field="source.type"]');
  // Media kinds have exactly one valid storage: there is nothing to choose,
  // so normalize the model and hide the control rather than show a dropdown
  // with a disabled option that can never actually be picked.
  const storageField = storage.closest("label");
  storageField.hidden = media;
  storage.disabled = media;
  if (media && library.source.type === "calibre") {
    library.source.type = "filesystem";
    storage.value = "filesystem";
    renderSource(card, "filesystem");
  }
  for (const option of storage.options)
    option.disabled =
      (["documents", "vault"].includes(kind) &&
        option.value !== "filesystem") ||
      option.textContent.includes("planned");
  card.querySelectorAll("[data-publication-only]").forEach((field) => {
    const active =
      !media &&
      (!field.dataset.source || field.dataset.source === storage.value);
    field.hidden = !active;
    field.querySelectorAll("input,select").forEach((input) => {
      input.disabled = !active;
    });
  });
  card.querySelectorAll("[data-vault-only]").forEach((field) => {
    const active = kind === "vault" && storage.value === "filesystem";
    field.hidden = !active;
    field.querySelectorAll("input,select").forEach((input) => {
      input.disabled = !active;
    });
  });
  const streaming = card.querySelector("[data-page-streaming]");
  const supportsStreaming = ["comics", "manga", "mixed"].includes(kind);
  streaming.hidden = !supportsStreaming;
  streaming.querySelector("input").disabled = !supportsStreaming;
  renderAdvancedGroup(card);
  const note = card.querySelector(".media-support-note");
  note.hidden = !media;
  if (["movies", "tv", "anime"].includes(kind))
    note.textContent =
      "Local video folders support direct and compatibility playback, seasons, artwork, and resume.";
  else if (kind === "audiobooks")
    note.textContent =
      "Local audiobook folders support embedded tags and chapters, resume, speed, sleep timer, artwork, streaming, and downloads.";
  else if (kind === "podcasts")
    note.textContent =
      "Local podcast audio is indexed here. RSS feeds can also be added from this shelf after saving it.";
  else if (kind === "music")
    note.textContent =
      "Local music folders use embedded title, artist, album, disc, track, duration, and artwork tags.";
  else if (kind === "documents")
    note.textContent =
      "Local folders support read-only document indexing. Mochil never rewrites document sources.";
  else if (kind === "vault")
    note.textContent =
      library.vault_writes && storage.value === "filesystem"
        ? "This local vault allows guarded Markdown editing. Mochil preserves raw files and never indexes or changes hidden configuration such as .obsidian."
        : "Obsidian vaults preserve their folder structure and stay read-only until Wiki editing is explicitly enabled for a local folder.";
}

// Library scans finish in the background; poll for a while, but not after the
// card that showed the status has left the page (a save, a tab switch away).
export const scanPollAttempts = 180;
export function stillScanning(card, attempt, attempts = scanPollAttempts) {
  return attempt < attempts && card?.isConnected !== false;
}

export function createLibrarySettings({
  document,
  $,
  $$,
  requestJSON,
  folderPicker,
  visibility = { isVisible: () => true, setVisible: () => {}, count: () => undefined },
  onLibraries,
  onLibraryActivated = () => {},
  onDirty = () => {},
  confirmRemoval = globalThis.confirm,
}) {
  mountLibraryFormTemplate(document);
  let libraries = [];
  let metadataProviders = {};
  // What the detection endpoint saw, kept out of the library document itself.
  // Keys are the card's library objects, so it follows the card, and it is
  // dropped as soon as the reader changes a field.
  const inspectionNotes = new WeakMap();

  function clearEvidence(card) {
    const line = card.querySelector("[data-evidence]");
    if (!line) return;
    line.textContent = "";
    line.hidden = true;
  }

  function syncMetadataLine(card, library) {
    const line = card.querySelector("[data-metadata-line]");
    if (line) line.textContent = metadataLineText(library, metadataProviders);
  }

  async function scanLibrary(card, library) {
    const button = card.querySelector(".scan-library");
    const status = card.querySelector(".library-scan-status");
    button.disabled = true;
    button.textContent = "Starting…";
    try {
      onLibraryActivated([library.id]);
      const response = await requestJSON("/api/library-scan", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ library_id: library.id }),
      });
      const jobKey = response.data.job_key;
      button.textContent = "Scanning…";
      status.textContent = "Checking for new files and changed metadata…";
      for (let attempt = 0; stillScanning(card, attempt); attempt += 1) {
        await new Promise((resolve) => globalThis.setTimeout(resolve, 1000));
        const jobs = await requestJSON("/api/jobs");
        const job = (jobs.data || []).find(
          (value) => value.queue === "library-scans" && value.key === jobKey,
        );
        if (!job || ["queued", "running"].includes(job.status)) continue;
        if (job.status === "ready") {
          status.textContent =
            "Scan complete. New and changed files are available.";
          onLibraries(libraries);
        } else status.textContent = job.error || "The scan did not complete.";
        return;
      }
      status.textContent = "The scan is still running in the background.";
    } catch (error) {
      status.textContent = error.message;
    } finally {
      button.disabled = false;
      button.textContent = "Scan now";
    }
  }

  function commit() {
    $$("#library-settings .library-form").forEach((card, index) => {
      const library = libraries[index];
      if (!library) return;
      card.querySelectorAll("[data-field]:not(:disabled)").forEach((input) => {
        const value =
          input.dataset.field === "preferred_formats"
            ? normalizePreferredFormats(input.value)
            : input.type === "checkbox"
              ? input.checked
              : input.value;
        pathSet(library, input.dataset.field, value);
      });
      library.name = String(library.name || "").trim();
    });
    libraries.forEach((library) => {
      if (!library.id.startsWith(pendingIdPrefix)) return;
      const taken = new Set(
        libraries.filter((other) => other !== library).map((other) => other.id),
      );
      library.id = libraryIdFromName(library.name, taken);
    });
  }

  // The switcher lists libraries in the order they appear in the settings, and Home is not one of
  // them, so it is always first for free. Reordering the array here is the whole feature: the save
  // sends the array in this order and the server keeps it.
  function renderOrder() {
    const panel = createLibraryOrderPanel({
      libraries,
      iconForKind,
      onChange: moveLibrary,
    });
    if (panel) $("#library-settings").append(panel);
  }

  function moveLibrary(from, to) {
    if (from === to || to < 0 || to >= libraries.length) return;
    const [moved] = libraries.splice(from, 1);
    libraries.splice(to, 0, moved);
    onDirty();
    render();
    // Keep the row you just moved in reach, so a keyboard or a thumb can move it again.
    const row = $(`#library-settings .library-order-row[data-id="${moved.id}"]`);
    row?.querySelector(".library-order-move:not(:disabled)")?.focus({ preventScroll: true });
  }

  function render() {
    const host = $("#library-settings");
    host.innerHTML = "";
    renderOrder();
    const hint = $("#first-library-hint");
    if (hint) hint.hidden = !shouldShowFirstLibraryHint(libraries);
    libraries.forEach((library, index) => {
      if (
        ["comics", "manga"].includes(library.kind) &&
        sameFormats(library.preferred_formats, defaultPreferredFormats("books"))
      ) {
        // Upgrade to the formats this kind actually ships, without treating it as an edit: the page
        // used to mark itself dirty the moment it opened because of this, and warn on the way out of a
        // page nobody had touched. The value is normalised again on the next open, so nothing is lost.
        library.preferred_formats = defaultPreferredFormats(library.kind);
      }
      const node = $("#library-form").content.cloneNode(true);
      const card = node.querySelector(".library-form");
      const updateSummary = () => {
        card.querySelector(".library-name").textContent =
          library.name || "New library";
        // The icon shape says what kind of library it is; the colour is the library's own hue, the
        // same one the switcher paints it with, so one shelf is one colour everywhere it appears.
        const icon = card.querySelector(".library-icon");
        icon.dataset.icon = iconForKind(library.kind);
        icon.dataset.library = library.id;
        icon.style.setProperty("--kind-h", String(libraryHue(library.id)));
        const items = visibility.count(library.id);
        card.querySelector(".library-summary-meta").textContent =
          [kindLabel(library.kind), storageLabel(library.source.type), Number.isFinite(items) ? `${items.toLocaleString()} items` : ""].filter(Boolean).join(" · ");
      };
      updateSummary();
      if (folderPicker)
        card
          .querySelectorAll('[data-field="source.path"]')
          .forEach((input) => attachFolderBrowse(input, folderPicker));
      // A saved library can be hidden on this device; one still being created cannot yet.
      const advanced = card.querySelector(".library-advanced");
      const optionsToggle = card.querySelector(".library-options-toggle");
      optionsToggle.onclick = () => (advanced.open = !advanced.open);
      advanced.addEventListener("toggle", () => optionsToggle.setAttribute("aria-expanded", advanced.open));
      // The help line stays for screen readers but shows as a hover hint, which keeps the options grid compact.
      advanced.querySelectorAll("label").forEach((label) => {
        const help = label.querySelector(".field-help, .check-field small");
        if (help) label.title = help.textContent;
      });
      const showToggle = card.querySelector(".library-show-toggle");
      const showLabel = card.querySelector(".library-show");
      showLabel.hidden = library.id.startsWith(pendingIdPrefix);
      showToggle.checked = visibility.isVisible(library.id);
      showToggle.setAttribute("aria-label", `Show ${library.name || library.id} on this device`);
      showToggle.onchange = () => visibility.setVisible(library.id, showToggle.checked);
      const evidence = inspectionNotes.get(library) || "";
      const evidenceLine = card.querySelector("[data-evidence]");
      evidenceLine.textContent = evidence;
      evidenceLine.hidden = !evidence;
      card.querySelectorAll("[data-field]").forEach((input) => {
        if (input.dataset.field === "scan_every" && library.scan_every && ![...input.options].some((option) => option.value === library.scan_every))
          input.add(new Option(`Every ${library.scan_every}`, library.scan_every));
        if (input.type === "checkbox")
          input.checked = Boolean(pathGet(library, input.dataset.field));
        else if (input.dataset.field === "preferred_formats")
          input.value = (
            pathGet(library, input.dataset.field) ??
            defaultPreferredFormats(library.kind)
          ).join(", ");
        else input.value = pathGet(library, input.dataset.field) || "";
        input.oninput = () => {
          const value =
            input.dataset.field === "preferred_formats"
              ? normalizePreferredFormats(input.value)
              : input.type === "checkbox"
                ? input.checked
                : input.value;
          // The evidence described the folder as it was found; once a field moves
          // it no longer describes what the reader is about to save.
          inspectionNotes.delete(library);
          clearEvidence(card);
          if (input.dataset.field === "kind")
            applyLibraryKind(library, input.value);
          else pathSet(library, input.dataset.field, value);
          if (input.dataset.field === "kind") {
            card.querySelector('[data-field="preferred_formats"]').value =
              library.preferred_formats.join(", ");
            card.querySelector('[data-field="inference.folder_layout"]').value =
              library.inference.folder_layout;
            card.querySelector('[data-field="source.type"]').value =
              library.source.type;
          }
          renderSource(
            card,
            input.dataset.field === "source.type"
              ? input.value
              : library.source.type,
          );
          renderContentKind(card, library.kind, library);
          syncMetadataLine(card, library);
          updateSummary();
          onDirty();
        };
      });
      card.querySelector(".remove").onclick = () => {
        if (
          !confirmRemoval(
            `Remove “${library.name || "this library"}” from Mochil settings?`,
          )
        )
          return;
        libraries.splice(index, 1);
        render();
        onDirty();
      };
      const scanButton = card.querySelector(".scan-library");
      scanButton.disabled = library.id.startsWith(pendingIdPrefix);
      scanButton.title = scanButton.disabled
        ? "Save this library before scanning it"
        : "Find new files and refresh changed metadata";
      scanButton.onclick = () => scanLibrary(card, library);
      card.addEventListener("toggle", () => {
        if (!card.open) return;
        host
          .querySelectorAll(":scope > .library-form[open]")
          .forEach((other) => {
            if (other !== card) other.open = false;
          });
        requestAnimationFrame(() =>
          card.scrollIntoView({ behavior: "smooth", block: "start" }),
        );
      });
      renderSource(card, library.source.type);
      renderContentKind(card, library.kind, library);
      syncMetadataLine(card, library);
      host.append(node);
    });
  }

  function appendLibrary(library) {
    libraries.push(library);
    render();
    onDirty();
    const added = $("#library-settings .library-form:last-child");
    if (!added) return;
    added.open = true;
    added.scrollIntoView?.({ block: "center", behavior: "smooth" });
    added.querySelector('[data-field="name"]')?.focus({ preventScroll: true });
  }

  // Adding a library starts with the folder, because the folder already says
  // what the library is. The card opens prefilled from the detection endpoint.
  async function addLibraryFromFolder() {
    if (!folderPicker) {
      appendLibrary(newLibrary(libraries.length + 1));
      return;
    }
    const path = await folderPicker.pick("");
    if (!path) {
      // No browsable locations on the server: the picker can only be closed, so
      // fall back to a card whose path is typed by hand instead of a dead end.
      if (folderPicker.outcome?.() === "empty") {
        appendLibrary(newLibrary(libraries.length + 1));
        const added = $("#library-settings .library-form:last-child");
        const status = added?.querySelector(".library-scan-status");
        if (status)
          status.textContent =
            "Type the folder path: no browsable locations are configured on the server.";
        added?.querySelector('[data-field="source.path"]')?.focus({ preventScroll: true });
      }
      return; // cancelled: nothing to add
    }
    const button = $("#add-library");
    if (button) button.disabled = true;
    let result;
    try {
      result = await inspectFolderLibrary(path, {
        requestJSON,
        number: libraries.length + 1,
      });
    } finally {
      if (button) button.disabled = false;
    }
    if (result.evidence) inspectionNotes.set(result.library, result.evidence);
    appendLibrary(result.library);
  }

  return {
    bind() {
      $("#add-library").onclick = () => addLibraryFromFolder();
    },
    load(nextLibraries = [], nextMetadataProviders = {}) {
      libraries = nextLibraries;
      metadataProviders = nextMetadataProviders || {};
      render();
    },
    commit,
    values: () => libraries,
  };
}
