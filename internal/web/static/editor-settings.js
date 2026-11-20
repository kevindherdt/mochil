// Removed: the Settings → Editor page held one switch, "show the editor in the menu on this device", and the
// editor is now a normal page in the menu like Hermes. Nothing here to configure, so nothing to render.
export function createEditorSettings() {
  return { bind() {}, load() {} };
}
