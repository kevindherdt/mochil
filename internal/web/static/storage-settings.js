import { mountStorageSettings } from "./storage-settings-template.js?v=5";

// The drop folder for Import lives on the Import page itself (a folder picker,
// not a pair of typed paths), since 0.1.20. This card used to also offer those
// same two fields, and saving it with its own copy empty silently deleted
// whatever Import had just set — the two fell out of sync by nothing more than
// opening Settings and clicking Save. managed_ingest is intentionally never
// read or written here now; Import owns it exclusively.
export function createStorageSettings({ document, $, onDirty = () => {} }) {
  mountStorageSettings(document);
  const fields = {
    scanEvery: $("#scan-every"),
    coverMiB: $("#cover-cache-mib"),
  };

  return {
    bind() {
      Object.values(fields).forEach((input) =>
        input.addEventListener("input", onDirty),
      );
    },
    load(settings = {}) {
      fields.scanEvery.value = settings.scan_every || "10m";
      fields.coverMiB.value = Math.round(
        (settings.cover_cache?.maximum_bytes || 536870912) / 1048576,
      );
    },
    write(settings) {
      settings.scan_every = fields.scanEvery.value;
      settings.cover_cache ??= {};
      settings.cover_cache.maximum_bytes =
        Number(fields.coverMiB.value) * 1048576;
      settings.cover_cache.maximum_width ||= 600;
      settings.cover_cache.maximum_height ||= 900;
      settings.cover_cache.jpeg_quality ||= 82;
    },
  };
}
