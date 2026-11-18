import { renderMarkdown } from "./markdown-renderer.js?v=1";
import { createVaultMarkdownRenderer } from "./vault-markdown.js?v=6";
import { parseVaultReference, jumpToVaultHeading } from "./vault-heading.js?v=1";
import { createVaultPageResolver } from "./vault-page-resolver.js?v=1";
import { bindVaultTasks } from "./vault-task.js?v=1";
import { isVaultPage, vaultPageEntries } from "./vault-pages.js?v=1";
import { createContinuousPager } from "./continuous-pager.js?v=4";
import { createBoundedWindow } from "./bounded-window.js?v=1";
import { createVaultTree } from "./vault-tree.js?v=3";
import { bindCodeCopy } from "./code-copy.js?v=1";
import { mountDocumentLibrary } from "./document-library-template.js?v=2";
import { mountVaultCreate } from "./vault-create.js?v=1";
import { vaultHomeHTML } from "./vault-home.js?v=3";
import { vaultNoteHTML, mountVaultOutline, dropRepeatedTitle } from "./vault-note.js?v=4";
import { launchVaultEditor } from "./vault-edit-launch.js?v=36";
import { mountBacklinks } from "./vault-backlinks.js?v=1";
import { mountVaultExtras, graphButtonHTML } from "./vault-extras.js?v=17";
import { createVaultAssetResolver } from "./vault-assets.js?v=1";

export function createDocumentLibrary({
  document,
  $,
  escapeHTML,
  requestJSON,
  setNavigation = () => {},
}) {
  mountDocumentLibrary(document);
  let request,
    textRequest,
    items = [],
    vaultEntries = [],
    total = 0,
    startOffset = 0,
    nextCursor = "",
    loading = false,
    prepending = false,
    library = "",
    search = "",
    kind = "documents",
    vaultWritable = false,
    activeVaultItem = "";
  const renderVaultTree = createVaultTree({ escapeHTML });
  let renderVaultMarkdown = () => "";
  let resolveVaultPage = () => null;
  const vaultWidth = () =>
    localStorage.getItem("bobobooks-vault-width") === "wide"
      ? "wide"
      : "centered";
  let vaultSidebarOpen =
    localStorage.getItem("bobobooks-vault-sidebar") !== "closed";
  const phoneLayout = () => globalThis.matchMedia?.("(max-width: 760px)").matches === true;
  function setVaultSidebar(workspace, open, remember = true) {
    workspace.dataset.sidebar = open ? "open" : "closed";
    vaultSidebarOpen = open;
    if (remember) localStorage.setItem("bobobooks-vault-sidebar", open ? "open" : "closed");
    workspace.querySelectorAll("[data-vault-sidebar]").forEach((button) => {
      button.setAttribute("aria-expanded", String(open));
      button.setAttribute("aria-label", `${open ? "Collapse" : "Open"} vault sidebar`);
    });
  }
  const pageSize = 120;
  const documentWindow = createBoundedWindow({
    container: $("#documents"),
    topSpacer: $("#document-top-spacer"),
    bottomSpacer: $("#document-bottom-spacer"),
    maximum: 360,
    trim: pageSize,
  });
  const icons = {
    more: '<circle cx="12" cy="5" r="1.4"/><circle cx="12" cy="12" r="1.4"/><circle cx="12" cy="19" r="1.4"/>',
    download: '<path d="M12 3v12m-5-5 5 5 5-5M5 21h14"/>',
  };
  const folder = (item) =>
    item.path.includes("/")
      ? item.path.slice(0, item.path.lastIndexOf("/"))
      : "Top level";
  const size = (bytes) =>
    bytes < 1024
      ? `${bytes} B`
      : bytes < 1048576
        ? `${Math.round(bytes / 1024)} KB`
        : `${(bytes / 1048576).toFixed(bytes < 10485760 ? 1 : 0)} MB`;
  const menu = (item) =>
    `<details class="document-more"><summary aria-label="More options for ${escapeHTML(item.title)}"><svg viewBox="0 0 24 24" aria-hidden="true">${icons.more}</svg></summary><div><a href="${escapeHTML(item.download_url)}" download><svg viewBox="0 0 24 24" aria-hidden="true">${icons.download}</svg>Download</a></div></details>`;
  const opener = (item, content) =>
    item.text_url
      ? `<button class="document-open" data-document="${escapeHTML(item.id)}">${content}</button>`
      : `<a class="document-open" href="${escapeHTML(item.open_url)}" target="_blank" rel="noopener">${content}</a>`;
  const context = (item) => item.match_snippet || folder(item);
  const card = (item) =>
    `<article class="document-card">${opener(item, `<span class="document-paper"><b>${escapeHTML(item.format)}</b><i></i><i></i><i></i></span><strong>${escapeHTML(item.title)}</strong><small${item.match_snippet ? ' class="document-match"' : ""}>${escapeHTML(context(item))}</small>`)}${menu(item)}</article>`;
  const row = (item) =>
    `<article class="document-row">${opener(item, `<span class="document-format">${escapeHTML(item.format)}</span><span><strong>${escapeHTML(item.title)}</strong><small${item.match_snippet ? ' class="document-match"' : ""}>${escapeHTML(context(item))}${item.match_snippet ? "" : ` · ${size(item.size)}`}</small></span>`)}${menu(item)}</article>`;
  const officeText = (value) =>
    `<div class="document-office-text">${value
      .split(/\n+/)
      .filter((line) => line.trim())
      .map((line) => `<p>${escapeHTML(line)}</p>`)
      .join("")}</div>`;
  const preformatted = new Set([
    "txt",
    "log",
    "csv",
    "tsv",
    "json",
    "jsonl",
    "yaml",
    "yml",
    "toml",
    "tex",
  ]);
  const sidebarIcon =
    '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 4h16v16H4zM9 4v16"/></svg>';
  const sidebarToggle = (sidebarOpen) =>
    `<button class="vault-sidebar-toggle" type="button" data-vault-sidebar aria-label="${sidebarOpen ? "Collapse" : "Open"} vault sidebar" aria-expanded="${sidebarOpen}">${sidebarIcon}</button>`;
  const vaultCreateActions = () =>
    vaultWritable
      ? `<div class="vault-create-actions" aria-label="Create in vault">
          <button type="button" data-vault-create="page" aria-label="New page" title="New page"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M6 3h8l4 4v14H6zM14 3v5h5M12 11v6m-3-3h6"/></svg></button>
          <button type="button" data-vault-daily aria-label="Today's note" title="Today's note"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M5 5h14v15H5zM5 9h14M9 3v4m6-4v4"/></svg></button>
          <button type="button" data-vault-create="folder" aria-label="New folder" title="New folder"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M3 6h7l2 2h9v11H3zM12 11v5m-2.5-2.5h5"/></svg></button>
        </div>`
      : "";
  function setVaultReadingMode(enabled) {
    document.body?.classList.toggle("vault-reading", enabled);
    $("#library-view")?.classList.toggle("vault-library-view", enabled);
  }
  function vaultWorkspace(visible) {
    const sidebar = vaultSidebarOpen;
    const message = visible.some(isVaultPage)
      ? "Choose a page from the vault."
      : search ? "No matching vault pages found." : "No pages yet.";
    return `<section class="vault-workspace" data-writable="${vaultWritable}" data-sidebar="${sidebar ? "open" : "closed"}" data-width="${vaultWidth()}">${sidebarToggle(false)}<aside><header class="vault-sidebar-tools">${sidebarToggle(true)}<strong class="vault-sidebar-title">Pages <small>${visible.filter(isVaultPage).length.toLocaleString()}</small></strong><button type="button" class="vault-switch" data-vault-switch aria-label="Jump to page" title="Jump to page (Ctrl+O)"><svg viewBox="0 0 24 24" aria-hidden="true"><circle cx="11" cy="11" r="6"/><path d="m16 16 4 4"/></svg></button>${graphButtonHTML}${vaultCreateActions()}</header>${renderVaultTree(visible, activeVaultItem)}</aside><main class="vault-pane">${vaultHomeHTML(visible, { message, writable: vaultWritable, escapeHTML })}</main></section>`;
  }
  function render(newItems = items, mode = "rerender") {
    if (kind === "vault") {
      setVaultReadingMode(true);
      const pages = vaultPageEntries(vaultEntries);
      const visible = search
        ? pages.filter((item) =>
            `${item.title || ""} ${item.path}`
              .toLowerCase()
              .includes(search.toLowerCase()),
          )
        : pages;
      $("#documents").className = "vault-browser";
      $("#documents").innerHTML = vaultWorkspace(visible);
      $("#status").textContent = "";
      $("#document-load-more").hidden = true;
      $("#document-load-previous").hidden = true;
      setNavigation([]);
      return;
    }
    setVaultReadingMode(false);
    const layout = $("#document-layout").value;
    $("#documents").className =
      layout === "covers" ? "document-grid" : "document-list";
    const markup = newItems.map(layout === "covers" ? card : row).join("");
    if (mode === "append" || mode === "prepend") {
      const insertion = documentWindow.begin(mode);
      $("#documents").insertAdjacentHTML(
        mode === "append" ? "beforeend" : "afterbegin",
        markup,
      );
      const removed = documentWindow.finish(insertion);
      if (removed.start) {
        items.splice(0, removed.start);
        startOffset += removed.start;
      }
      if (removed.end) {
        items.splice(-removed.end, removed.end);
        nextCursor = "";
      }
    } else {
      if (mode === "replace") documentWindow.reset();
      $("#documents").innerHTML =
        markup || '<p class="empty-state">No matching documents found.</p>';
    }
    const loadedEnd = startOffset + items.length;
    $("#status").textContent =
      startOffset === 0 && loadedEnd === total
        ? `${total.toLocaleString()} ${total === 1 ? "document" : "documents"}`
        : `Showing ${(startOffset + 1).toLocaleString()}–${loadedEnd.toLocaleString()} of ${total.toLocaleString()} documents`;
    $("#document-load-more").hidden = loadedEnd >= total;
    $("#document-load-more").disabled = false;
    setNavigation(
      $("#document-sort").value === "title"
        ? [...$("#documents").children].map((element) => ({
            element,
            label: element.querySelector("strong")?.textContent || "",
          }))
        : [],
    );
    pager?.refresh();
    previousPager?.refresh();
  }
  async function openText(item, heading = "", scrollTop = 0) {
    textRequest?.abort();
    const current = new AbortController();
    textRequest = current;
    if (kind === "vault") {
      activeVaultItem = item.id;
      const workspace = $("#documents .vault-workspace");
      const pane = $("#documents .vault-pane");
      if (!workspace || !pane) render();
      $("#documents .vault-pane").innerHTML =
        '<p class="empty-state">Opening page…</p>';
      $("#documents .vault-page.active")?.classList.remove("active");
      const pageButton = $(
        `#documents [data-document="${CSS.escape(item.id)}"]`,
      );
      pageButton?.classList.add("active");
      pageButton?.setAttribute("aria-current", "page");
      if (matchMedia("(max-width: 760px)").matches)
        setVaultSidebar($("#documents .vault-workspace"), false, false);
    } else {
      $("#documents").className = "document-reader";
      $("#documents").innerHTML =
        '<p class="empty-state">Opening document…</p>';
    }
    $("#document-toolbar").hidden = true;
    $("#document-load-more").hidden = true;
    try {
      const result = await requestJSON(item.text_url, {
        signal: current.signal,
      });
      if (textRequest !== current) return;
      const content = preformatted.has(item.format)
        ? `<pre>${escapeHTML(result.data.content)}</pre>`
        : ["docx", "odt", "rtf"].includes(item.format)
          ? officeText(result.data.content)
          : kind === "vault"
            ? renderVaultMarkdown(result.data.content, item.path, {writable: result.data.writable === true && isVaultPage(item)})
            : renderMarkdown(result.data.content);
      if (kind === "vault") {
        const pane = $("#documents .vault-pane");
        pane.className = "vault-pane document-reader vault-document";
        const canEdit = result.data.writable === true && isVaultPage(item);
        pane.innerHTML = vaultNoteHTML({ item, libraryName: item.library_name, content, canEdit, escapeHTML });
        dropRepeatedTitle(pane, item.title);
        mountVaultOutline(pane);
        extras.noteOpened(item, pane);
        mountBacklinks({ pane, item, library, requestJSON, escapeHTML, isCurrent: () => textRequest === current });
        pane.scrollTop = scrollTop;
        bindCodeCopy(pane);
        bindVaultTasks({pane, item:{...item, library_id:library}, content:result.data.content,
          revision:result.data.edit_revision, writable:canEdit, requestJSON,
          isCurrent:()=>textRequest === current,
          onSaved:async offset=>{
            const position = pane.scrollTop;
            const loaded = await openText(item, "", position);
            if (loaded && textRequest === loaded)
              pane.querySelector(`input[data-vault-task="${offset}"]`)?.focus({preventScroll:true});
          },
        });
        if (heading) jumpToVaultHeading(pane, heading);
        // Hermes "Polish": offered only to administrators with Hermes set up; it loads when a note opens.
        if (canEdit) import("./vault-polish.js?v=4").then(({ mountVaultPolish }) => mountVaultPolish({ pane, item: { ...item, library_id: library }, requestJSON, onDone: () => openText(item) })).catch(() => {});
        pane.querySelector("[data-vault-edit]")?.addEventListener("click", () =>
          launchVaultEditor({
            pane,
            item: { ...item, library_id: library },
            libraryName: item.library_name,
            content: result.data.content,
            revision: result.data.edit_revision,
            escapeHTML,
            requestJSON,
            pages: vaultPageEntries(vaultEntries).filter(isVaultPage),
            pickTemplate: extras.pickTemplate,
            resolveImage: ((find) => (target) => find(item.path, target)?.url ?? null)(createVaultAssetResolver(vaultEntries, library)),
            onDone: () => openText(item),
          }),
        );
        return current;
      }
      $("#documents").innerHTML =
        `<header><button class="document-back" aria-label="Back to documents"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m15 5-7 7 7 7"/></svg></button><div><h1>${escapeHTML(item.title)}</h1><p>${escapeHTML(folder(item))}</p></div><a href="${escapeHTML(item.download_url)}" download>Download</a></header><article>${content}</article>`;
      bindCodeCopy($("#documents"));
      $(".document-back").onclick = () => {
        $("#document-toolbar").hidden = kind === "vault";
        render();
      };
    } catch (error) {
      if (textRequest !== current) return;
      if (error.name !== "AbortError") {
        if (kind === "vault") {
          $("#documents .vault-pane").innerHTML =
            `<section class="document-reader-error"><p>${escapeHTML(error.message)}</p><button type="button" data-document="${escapeHTML(item.id)}">Try again</button></section>`;
          return;
        }
        $("#documents").innerHTML =
          `<section class="document-reader-error"><p>${escapeHTML(error.message)}</p><button class="document-back">Back to documents</button></section>`;
        $(".document-back").onclick = () => {
          $("#document-toolbar").hidden = kind === "vault";
          render();
        };
      }
    }
  }
  async function load(
    selectedLibrary,
    query,
    reset = true,
    selectedKind = kind,
  ) {
    if (!selectedLibrary || (loading && !reset)) return;
    library = selectedLibrary;
    search = query || "";
    kind = selectedKind;
    if (reset) {
      request?.abort();
      textRequest?.abort();
      textRequest = null;
      activeVaultItem = "";
      prepending = false;
      items = [];
      vaultEntries = [];
      total = 0;
      startOffset = 0;
      nextCursor = "";
      documentWindow.reset();
      $("#documents").innerHTML = "";
      $("#status").textContent =
        kind === "vault" ? "Opening vault…" : "Opening documents…";
      $("#document-toolbar").hidden = kind === "vault";
    }
    const current = new AbortController();
    request = current;
    loading = true;
    const params = new URLSearchParams({
      library,
      limit: pageSize,
      sort: $("#document-sort").value,
    });
    if (kind === "vault") params.set("tree", "1");
    if (!reset)
      nextCursor
        ? params.set("cursor", nextCursor)
        : params.set("offset", startOffset + items.length);
    if (search) params.set("q", search);
    try {
      const result = await requestJSON(`/api/documents?${params}`, {
        signal: current.signal,
      });
      if (request !== current) return;
      total =
        Number(result.response.headers.get("X-Total-Count")) ||
        result.data.length;
      nextCursor = result.response.headers.get("X-Next-Cursor") || "";
      if (kind === "vault") {
        vaultWritable = result.response.headers.get("X-Vault-Writable") === "1";
        vaultEntries.push(...result.data);
        renderVaultMarkdown = createVaultMarkdownRenderer(vaultEntries, library);
        resolveVaultPage = createVaultPageResolver(vaultEntries, library);
        items.push(...result.data.filter((item) => item.kind !== "folder"));
        render(result.data, reset ? "replace" : "append");
      } else {
        items.push(...result.data);
        render(result.data, reset ? "replace" : "append");
      }
    } catch (error) {
      if (request !== current) return false;
      if (error.status === 409 && !reset) {
        loading = false;
        return load(library, search, true, kind);
      }
      if (error.name !== "AbortError") {
        $("#status").textContent = `Could not open documents: ${error.message}`;
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
    if (loading || prepending || startOffset <= 0 || kind === "vault")
      return false;
    prepending = true;
    const offset = Math.max(0, startOffset - pageSize);
    const current = new AbortController();
    request = current;
    loading = true;
    const params = new URLSearchParams({
      library,
      limit: startOffset - offset,
      offset,
      sort: $("#document-sort").value,
    });
    if (search) params.set("q", search);
    try {
      const result = await requestJSON(`/api/documents?${params}`, {
        signal: current.signal,
      });
      if (request !== current || !result.data.length) return false;
      startOffset = offset;
      items.unshift(...result.data);
      render(result.data, "prepend");
      return true;
    } catch (error) {
      if (request !== current) return false;
      if (error.name !== "AbortError")
        $("#status").textContent = `Could not load earlier documents: ${error.message}`;
      return false;
    } finally {
      if (request === current) {
        loading = false;
        prepending = false;
      }
      previousPager?.refresh();
    }
  }
  $("#document-sort").value =
    localStorage.getItem("bobobooks-document-sort") || "title";
  $("#document-sort").onchange = () => {
    localStorage.setItem("bobobooks-document-sort", $("#document-sort").value);
    load(library, search, true);
  };
  $("#document-layout").value =
    localStorage.getItem("bobobooks-document-layout") || "covers";
  $("#document-layout").onchange = () => {
    localStorage.setItem(
      "bobobooks-document-layout",
      $("#document-layout").value,
    );
    $("#view-menu").open = false;
    render();
  };
  $("#documents").onclick = (event) => {
    const create = event.target.closest("[data-vault-create]");
    if (create) {
      const host = $("#documents .vault-workspace aside");
      if (host)
        mountVaultCreate({
          host,
          type: create.dataset.vaultCreate,
          library,
          requestJSON,
          onCreated: () => load(library, search, true),
        });
      return;
    }
    if (event.target.closest("[data-vault-switch]")) {
      extras.openSwitcher();
      return;
    }
    const tagLink = event.target.closest('a[href^="#tag:"]');
    if (tagLink) {
      event.preventDefault();
      extras.showTag(decodeURIComponent(tagLink.getAttribute("href").slice(5)));
      return;
    }
    const sidebar = event.target.closest("[data-vault-sidebar]");
    const workspace = $("#documents .vault-workspace");
    // On a phone the sidebar is a drawer: the dimmed page behind it closes it.
    const scrim = workspace && event.target === workspace && phoneLayout();
    if (sidebar || scrim) {
      setVaultSidebar(workspace, scrim ? false : workspace.dataset.sidebar !== "open", !scrim);
      return;
    }
    const link = event.target.closest('a[href^="#vault:"]');
    if (link) {
      event.preventDefault();
      const reference = parseVaultReference(link.getAttribute("href"));
      if (!reference) return;
      if (!reference.page) {
        jumpToVaultHeading($("#documents .vault-pane"), reference.heading);
        return;
      }
      const currentPage = items.find((candidate) => candidate.id === activeVaultItem);
      const item = resolveVaultPage(currentPage?.path || "", reference.page);
      if (item) {
        if (item.id === activeVaultItem && reference.heading)
          jumpToVaultHeading($("#documents .vault-pane"), reference.heading);
        else openText(item, reference.heading);
      }
      return;
    }
    const button = event.target.closest("[data-document]");
    if (!button) return;
    const item = items.find(
      (candidate) => candidate.id === button.dataset.document,
    );
    if (item) {
      // Choosing a page on a phone puts the drawer away so the page is what you see.
      const workspace = $("#documents .vault-workspace");
      if (workspace && phoneLayout() && workspace.dataset.sidebar === "open") setVaultSidebar(workspace, false, false);
      openText(item);
    }
  };
  const extras = mountVaultExtras({
    document, $, escapeHTML, requestJSON,
    library: () => library,
    items: () => items,
    entries: () => vaultEntries,
    writable: () => vaultWritable,
    active: () => activeVaultItem,
    openText,
    reload: () => load(library, search, true),
    expandFolder: renderVaultTree.expand,
  });
  const pager = createContinuousPager({
    sentinel: $("#document-sentinel"),
    button: $("#document-load-more"),
    label: "documents",
    canLoad: () =>
      !$("#document-index").hidden &&
      kind !== "vault" &&
      !loading &&
      startOffset + items.length < total,
    load: () => load(library, search, false),
  });
  const previousPager = createContinuousPager({
    sentinel: $("#document-start-sentinel"),
    button: $("#document-load-previous"),
    label: "previous documents",
    rootMargin: "500px",
    canLoad: () =>
      !$("#document-index").hidden &&
      kind !== "vault" &&
      !loading &&
      !prepending &&
      startOffset > 0,
    load: loadPrevious,
  });
  return {
    load,
    abort: () => {
      request?.abort();
      textRequest?.abort();
      request = null;
      textRequest = null;
      loading = false;
      prepending = false;
      setVaultReadingMode(false);
    },
  };
}
