// Opens the note editor. The live editor loads on first use; the plain-text one
// stays as the safety net if it cannot load.
import { mountVaultEditor } from "./vault-editor.js?v=51";

export async function launchVaultEditor({ libraryName, onDone, ...editing }) {
  try {
    const { mountLiveEditor } = await import("./vault-live-editor.js?v=59");
    mountLiveEditor({ ...editing, libraryName, onClose: onDone });
  } catch (error) {
    console.error("Live editor failed to load", error);
    mountVaultEditor({ ...editing, onCancel: onDone, onSaved: onDone });
  }
}
