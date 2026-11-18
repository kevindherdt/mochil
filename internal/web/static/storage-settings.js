import { managedIngestConfig } from "./settings-managed-ingest.js?v=1";
import { mountStorageSettings } from "./storage-settings-template.js?v=4";

export function createStorageSettings({ document, $, onDirty = () => {} }) {
  mountStorageSettings(document);
  const fields = {
    scanEvery: $("#scan-every"),
    coverMiB: $("#cover-cache-mib"),
    incoming: $("#managed-ingest-incoming"),
    target: $("#managed-ingest-target"),
    ingestGiB: $("#managed-ingest-gib"),
  };

  function renderManagedIngestState() {
    const configured = Boolean(
      fields.incoming.value.trim() || fields.target.value.trim(),
    );
    fields.incoming.required = configured;
    fields.target.required = configured;
    fields.ingestGiB.disabled = !configured;
  }

  return {
    bind() {
      Object.values(fields).forEach((input) =>
        input.addEventListener("input", onDirty),
      );
      fields.incoming.addEventListener("input", renderManagedIngestState);
      fields.target.addEventListener("input", renderManagedIngestState);
    },
    load(settings = {}) {
      fields.scanEvery.value = settings.scan_every || "10m";
      fields.coverMiB.value = Math.round(
        (settings.cover_cache?.maximum_bytes || 536870912) / 1048576,
      );
      fields.incoming.value = settings.managed_ingest?.incoming_directory || "";
      fields.target.value = settings.managed_ingest?.managed_directory || "";
      fields.ingestGiB.value = Math.round(
        (settings.managed_ingest?.maximum_file_bytes || 8589934592) /
          1073741824,
      );
      renderManagedIngestState();
    },
    write(settings) {
      settings.scan_every = fields.scanEvery.value;
      settings.cover_cache ??= {};
      settings.cover_cache.maximum_bytes =
        Number(fields.coverMiB.value) * 1048576;
      settings.cover_cache.maximum_width ||= 600;
      settings.cover_cache.maximum_height ||= 900;
      settings.cover_cache.jpeg_quality ||= 82;
      const ingest = managedIngestConfig(
        fields.incoming.value,
        fields.target.value,
        fields.ingestGiB.value,
      );
      if (ingest) settings.managed_ingest = ingest;
      else delete settings.managed_ingest;
    },
  };
}
