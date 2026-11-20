import { mountStorageSettings } from "./storage-settings-template.js?v=5";

// The drop folder for Import lives on the Import page itself (a folder picker,
// not a pair of typed paths), since 0.1.20. This card used to also offer those
// same two fields, and saving it with its own copy empty silently deleted
// whatever Import had just set — the two fell out of sync by nothing more than
// opening Settings and clicking Save. managed_ingest is intentionally never
// read or written here now; Import owns it exclusively.
// The card is now one action ("scan now"). The two knobs it used to hold — the default scan interval and
// the cover-cache size — were settings nobody needed to think about, on a page about libraries; both keep
// their stored values and defaults, and nothing here reads or writes them any more.
export function createStorageSettings({ document, $, onDirty = () => {} }) {
  mountStorageSettings(document);

  return {
    bind() {},
    load() {},
    write() {},
  };
}
