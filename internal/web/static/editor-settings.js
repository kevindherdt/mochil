// Settings → Modules → Editor: whether the library editor appears in the Home/library menu.
// The section and its #editor-menu-toggle checkbox are static in index.html (not inserted by this
// module) because editor-link.js wires that checkbox up eagerly at page load, before this settings module
// — which is lazy-loaded only once the Settings view opens — would otherwise exist to insert it.
// The Hermes-assisted editor tools ("Ask about Hardcover matches", "Ask about ISBNs") are Hermes
// abilities like any other, so they are drawn and saved by hermes-settings.js under
// settings.hermes.features; this module only owns the editor section's own visibility.
import { effectiveModuleState } from "./modules-settings.js?v=19";

export function createEditorSettings({ $ }) {
  return {
    bind() {},
    // The section (inside Settings, Modules) shows only while the editor module is on.
    load(settings, modules = []) {
      const editorModule = modules.find((item) => item.id === "editor");
      $('[data-settings-panel="editor"]').hidden = !(Boolean(editorModule) && effectiveModuleState(editorModule, settings));
    },
  };
}
