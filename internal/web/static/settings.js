import { createAccountsSettings } from "./accounts-settings.js?v=13";
import { createModulesSettings } from "./modules-settings.js?v=19";
import { createHermesSettings } from "./hermes-settings.js?v=18";
import { createEditorSettings } from "./editor-settings.js?v=3";
import { createAvatarSettings } from "./avatar-settings.js?v=7";
import { offerRestart } from "./restart-prompt.js?v=1";
import { createNarrationVoices } from "./narration-voices-settings.js?v=22";
import { mountDiagnostics } from "./diagnostics-panel.js?v=4";
import { createManagedIngestUI } from "./managed-ingest-ui.js?v=1";
import { createLibrarySettings } from "./library-settings.js?v=27";
import { createPlaybackSettings } from "./playback-settings.js?v=13";
import { createMetadataProviderSettings } from "./metadata-provider-settings.js?v=13";
import { createHardcoverShelf } from "./hardcover-shelf-settings.js?v=14";
import { createReadingSyncSettings } from "./reading-sync-settings.js?v=12";
import { settingsTabFor } from "./settings-tab-aliases.js?v=1";
import { createStorageSettings } from "./storage-settings.js?v=13";
import {
  attachFolderBrowse,
  createFolderPicker,
} from "./folder-picker.js?v=2";

export function firstInvalidGlobalSetting(inputs) {
  return [...inputs].find(
    (input) =>
      input.closest("[data-settings-panel]")?.dataset.settingsPanel !==
        "accounts" &&
      // A field inside its own <form> (like "Make a new voice") is checked when that form is sent, not by Save changes.
      !input.form &&
      !input.disabled &&
      !input.checkValidity(),
  );
}

export { defaultPreferredFormats } from "./library-settings.js?v=27";

export function createSettings({
  document,
  $,
  $$,
  requestJSON,
  libraryVisibility,
  onLibraries,
  onSaved,
  onLibraryActivated = () => {},
}) {
  let settings;
  let settingsDirty = false;
  let persistedLibraryIDs = new Set();
  const folderPicker = createFolderPicker({ document, requestJSON });
  const accountsSettings = createAccountsSettings({ document, $, requestJSON });
  const managedIngestUI = createManagedIngestUI({ $, requestJSON });
  const playbackSettings = createPlaybackSettings({
    document,
    $,
    onDirty: markSettingsDirty,
  });
  const metadataProviderSettings = createMetadataProviderSettings({
    document,
    $,
    requestJSON,
    onDirty: markSettingsDirty,
  });
  const hardcoverShelf = createHardcoverShelf({ document, $, requestJSON });
  const modulesSettings = createModulesSettings({ document, $, onDirty: markSettingsDirty });
  const hermesSettings = createHermesSettings({ document, $, requestJSON, onDirty: markSettingsDirty });
  const editorSettings = createEditorSettings({ document, $, onDirty: markSettingsDirty });
  const avatarSettings = createAvatarSettings({ document, $ });
  const narrationVoices = createNarrationVoices({ document, $ });
  mountDiagnostics(document);
  let moduleList = [];
  const storageSettings = createStorageSettings({
    document,
    $,
    onDirty: markSettingsDirty,
  });
  const readingSyncSettings = createReadingSyncSettings({
    document,
    $,
    onDirty: markSettingsDirty,
  });
  const librarySettings = createLibrarySettings({
    document,
    $,
    $$,
    requestJSON,
    folderPicker,
    visibility: libraryVisibility,
    onLibraries,
    onLibraryActivated,
    onDirty: markSettingsDirty,
  });

  function markSettingsDirty() {
    settingsDirty = true;
    $("#settings-status").textContent = "Unsaved changes";
    $("#discard").hidden = false;
  }

  // Account mutations are independent authenticated operations with their own
  // forms and validation. They must never block or redirect the global settings
  // save merely because an unopened password form is empty.
  function invalidGlobalSetting() {
    return firstInvalidGlobalSetting(
      $$("#settings-view input, #settings-view select"),
    );
  }

  function renderSettings() {
    librarySettings.load(settings.libraries, settings.metadata_providers);
    storageSettings.load(settings);
    modulesSettings.load(moduleList, settings);
    hermesSettings.load(settings, moduleList);
    editorSettings.load(settings, moduleList);
    avatarSettings.load(settings, moduleList);
    narrationVoices.load(moduleList).catch(() => {});
    playbackSettings.load(settings.playback);
    readingSyncSettings.load(settings.kosync, settings.public_url);
    metadataProviderSettings.load(
      settings.metadata_providers,
      settings.libraries,
    );
    hardcoverShelf.load();
    onLibraries(settings.libraries);
  }

  async function openSettings() {
    try {
      settings = (await requestJSON("/api/settings")).data;
      moduleList = (await requestJSON("/api/modules").catch(() => ({ data: [] }))).data;
      persistedLibraryIDs = new Set(
        (settings.libraries || []).map((library) => library.id),
      );
      settingsDirty = false;
      $("#discard").hidden = true;
      $("#settings-status").textContent = "";
      renderSettings();
      readingSyncSettings.checkAndroidDownload();
      await managedIngestUI.load();
      await accountsSettings.load();
    } catch (error) {
      $("#settings-status").textContent = error.message;
    }
  }

  function isLoaded() {
    return Boolean(settings);
  }

  function bind() {
    accountsSettings.bind();
    managedIngestUI.bind();
    playbackSettings.bind();
    metadataProviderSettings.bind();
    hardcoverShelf.bind();
    readingSyncSettings.bind();
    storageSettings.bind();
    modulesSettings.bind();
    hermesSettings.bind();
    editorSettings.bind();
    avatarSettings.bind();
    for (const id of ["#managed-ingest-incoming", "#managed-ingest-target"])
      attachFolderBrowse($(id), folderPicker);
    librarySettings.bind();
    // Settings opens on its index: one card per page, and a page replaces the
    // index rather than sitting under a tab strip on an endless scroll.
    const settingsView = $("#settings-view");
    const settingsBack = $("#settings-back");
    const showSettingsIndex = () => {
      settingsView.dataset.settingsPage = "index";
      $$("[data-settings-tab]").forEach((tab) => {
        tab.classList.remove("active");
        tab.setAttribute("aria-pressed", "false");
      });
      $$("#settings-view > [data-settings-panel]").forEach((panel) => {
        panel.hidden = true;
      });
      settingsBack.hidden = true;
      // Nothing to save until a page with fields is open.
      $(".settings-save").hidden = true;
      try { sessionStorage.removeItem("bobobooks-settings-tab"); } catch { /* private mode */ }
    };
    const openSettingsPage = (button) => {
      settingsView.dataset.settingsPage = button.dataset.settingsTab;
      $$("[data-settings-tab]").forEach((tab) => {
        const active = tab === button;
        tab.classList.toggle("active", active);
        tab.setAttribute("aria-pressed", active);
      });
      $$("#settings-view > [data-settings-panel]").forEach((panel) => {
        panel.hidden = panel.dataset.settingsPanel !== button.dataset.settingsTab;
      });
      $(".settings-save").hidden = ["accounts", "narration", "diagnostics"].includes(
        button.dataset.settingsTab,
      );
      settingsBack.hidden = false;
      settingsView.scrollIntoView?.({ block: "start" });
      try { sessionStorage.setItem("bobobooks-settings-tab", button.dataset.settingsTab); } catch { /* private mode */ }
    };
    $$("[data-settings-tab]").forEach(
      (button) => (button.onclick = () => openSettingsPage(button)),
    );
    settingsBack.onclick = showSettingsIndex;
    try {
      const remembered = sessionStorage.getItem("bobobooks-settings-tab");
      const target = remembered && $(`[data-settings-tab="${CSS.escape(settingsTabFor(remembered))}"]`);
      if (target) openSettingsPage(target);
      else showSettingsIndex();
    } catch { showSettingsIndex(); }
    $$(".settings-accordion").forEach((section) =>
      section.addEventListener("toggle", () => {
        if (!section.open) return;
        section.parentElement
          .querySelectorAll(":scope > .settings-accordion[open]")
          .forEach((other) => {
            if (other !== section) other.open = false;
          });
        requestAnimationFrame(() =>
          section.scrollIntoView({ behavior: "smooth", block: "start" }),
        );
      }),
    );
    $("#discard").onclick = () => openSettings();
    $("#save").onclick = async () => {
      librarySettings.commit();
      const invalid = invalidGlobalSetting();
      if (invalid) {
        const panel = invalid.closest("#settings-view > [data-settings-panel]");
        const tab =
          panel && $(`[data-settings-tab="${panel.dataset.settingsPanel}"]`);
        tab?.click();
        // Open every collapsed section around the field so the problem is on screen, not hidden in a closed list.
        for (let el = invalid.parentElement; el; el = el.parentElement)
          if (el.tagName === "DETAILS") el.open = true;
        invalid.scrollIntoView?.({ block: "center" });
        invalid.reportValidity();
        invalid.focus();
        $("#settings-status").textContent = "Complete the highlighted field";
        return;
      }
      storageSettings.write(settings);
      const modulesChanged = modulesSettings.changed();
      modulesSettings.write(settings);
      hermesSettings.write(settings);
      playbackSettings.write(settings);
      metadataProviderSettings.write(settings);
      readingSyncSettings.write(settings);
      const status = $("#settings-status");
      const save = $("#save");
      save.disabled = true;
      save.textContent = "Saving…";
      try {
        const addedLibraryIDs = settings.libraries
          .map((library) => library.id)
          .filter((id) => !persistedLibraryIDs.has(id));
        const result = (
          await requestJSON("/api/settings", {
            method: "PUT",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(settings),
          })
        ).data;
        settingsDirty = false;
        $("#discard").hidden = true;
        onLibraryActivated(addedLibraryIDs);
        persistedLibraryIDs = new Set(
          settings.libraries.map((library) => library.id),
        );
        await onSaved?.(result);
        await managedIngestUI.load();
        if (modulesChanged) offerRestart({ document, status });
        else status.textContent = result.restart_required?.length
          ? `Saved and applied. Restart only to change ${result.restart_required.join(", ")}.`
          : "Saved and applied. Libraries are ready.";
        save.textContent = "Saved";
        setTimeout(() => {
          save.textContent = "Save changes";
          save.disabled = false;
        }, 1400);
      } catch (error) {
        status.textContent = `Could not save: ${error.message}`;
        save.disabled = false;
        save.textContent = "Save changes";
      }
    };

    addEventListener("beforeunload", (event) => {
      if (!settingsDirty) return;
      event.preventDefault();
      event.returnValue = "";
    });
  }

  return { bind, isLoaded, open: openSettings };
}
