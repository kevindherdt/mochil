import { mountMetadataProviderSettings } from "./metadata-provider-settings-template.js?v=1";
import { effectiveMetadataProvider } from "./library-metadata.js?v=1";

const providers = { hardcover: "Hardcover", tmdb: "TMDB" };
const providerNames = { hardcover: "Hardcover", tmdb: "TMDB" };

// Which libraries the server will actually look up through this provider. The
// answer comes from the same derivation the card shows and the server applies;
// reading the raw saved field would say "not assigned" for every library that
// simply lets the provider pick itself.
export function metadataProviderUsage(libraries = [], provider, metadataProviders = {}) {
  const names = libraries
    .filter(
      (library) =>
        effectiveMetadataProvider(library, metadataProviders) ===
        providerNames[provider],
    )
    .map((library) => library.name || library.id);
  return names.length
    ? `Used by ${names.join(", ")}.`
    : "Not assigned to a library.";
}

export function createMetadataProviderSettings({
  document,
  $,
  requestJSON,
  onDirty = () => {},
}) {
  mountMetadataProviderSettings(document);
  let metadataProviders = {};
  let libraries = [];
  const field = (provider) => $(`#${provider}-token`);
  const status = (provider) => $(`#${provider}-provider-status`);
  const usage = (provider) =>
    metadataProviderUsage(libraries, provider, metadataProviders);

  function renderStatus(provider) {
    const output = status(provider);
    delete output.dataset.state;
    const connection = metadataProviders[`${provider}_token_configured`]
      ? `${providers[provider]} token saved.`
      : `${providers[provider]} is not connected.`;
    output.textContent = `${connection} ${usage(provider)}`;
  }

  async function testConnection(provider) {
    const button = $(`[data-test-provider="${provider}"]`);
    const input = field(provider);
    const output = status(provider);
    button.disabled = true;
    button.textContent = "Testing…";
    output.textContent = input.value.trim()
      ? "Testing the pasted token…"
      : "Testing the saved token…";
    try {
      const { data: result } = await requestJSON(
        "/api/metadata/provider-connection",
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ provider, token: input.value.trim() }),
        },
      );
      const resultUsage = result.libraries?.length
        ? ` Used by ${result.libraries.map((library) => library.name).join(", ")}.`
        : " Not assigned to a library.";
      output.textContent = `${result.message || "Connection verified."}${resultUsage}`;
      output.dataset.state = result.connected ? "connected" : "error";
    } catch (error) {
      output.textContent = `${error.message || "Mochil could not reach the provider. Check the connection and try again."} ${usage(provider)}`;
      output.dataset.state = "error";
    } finally {
      button.disabled = false;
      button.textContent = "Test connection";
    }
  }

  return {
    bind() {
      Object.keys(providers).forEach((provider) => {
        field(provider).addEventListener("input", onDirty);
        $(`[data-test-provider="${provider}"]`).onclick = () =>
          testConnection(provider);
      });
    },
    load(value = {}, valueLibraries = []) {
      metadataProviders = value;
      libraries = valueLibraries;
      Object.keys(providers).forEach((provider) => {
        const input = field(provider);
        input.value = "";
        input.placeholder = value[`${provider}_token_configured`]
          ? "Saved token"
          : "Not configured";
        renderStatus(provider);
      });
    },
    write(settings) {
      settings.metadata_providers ??= {};
      Object.keys(providers).forEach((provider) => {
        settings.metadata_providers[`${provider}_token`] =
          field(provider).value.trim();
        delete settings.metadata_providers[`${provider}_token_configured`];
      });
    },
  };
}
