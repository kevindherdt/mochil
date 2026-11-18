import { mountReadingSyncSettings } from "./reading-sync-settings-template.js?v=2";

export function readingSyncValue({
  provider,
  bridge,
  serverURL,
  username,
  password,
}) {
  const selectedProvider = ["built-in", "external", "disabled"].includes(
    provider,
  )
    ? provider
    : "built-in";
  const value = {
    provider: selectedProvider,
    enabled: selectedProvider !== "disabled",
    web_reader_bridge: Boolean(bridge),
    server_url:
      selectedProvider === "external"
        ? String(serverURL || "")
            .trim()
            .replace(/\/$/, "")
        : "",
  };
  const cleanUsername = String(username || "").trim();
  if (cleanUsername) value.username = cleanUsername;
  if (password) value.password = String(password);
  return value;
}

export function createReadingSyncSettings({
  document,
  $,
  onDirty = () => {},
  fetchImpl = globalThis.fetch,
  origin = globalThis.location?.origin || "",
}) {
  mountReadingSyncSettings(document);
  const fields = {
    provider: $("#kosync-provider"),
    bridge: $("#kosync-webreader-bridge"),
    serverURL: $("#kosync-server-url"),
    username: $("#kosync-username"),
    password: $("#kosync-password"),
  };
  let publicURL = "";

  function render() {
    const external = fields.provider.value === "external";
    const enabled = fields.provider.value !== "disabled";
    fields.bridge.disabled = !enabled;
    $("#kosync-external-fields").hidden = !external;
    $("#kosync-credentials").hidden = !enabled;
    fields.serverURL.required = external;
    $("#kosync-local-help").hidden = external || !enabled;
    if (!external && enabled)
      $("#kosync-local-help").textContent =
        `KoSync server URL: ${publicURL || origin}`;
    $("#kosync-account-help").textContent = external
      ? "Use the external account you also use in KOReader. Leave a saved password blank to keep it unchanged."
      : "Use these details to log in from KOReader. Leave a saved password blank to keep it unchanged.";
    $("#plugin-setup").hidden = fields.provider.value !== "built-in";
  }

  async function downloadPlugin() {
    const button = $("#download-plugin"),
      status = $("#plugin-status");
    button.disabled = true;
    status.textContent = "Preparing your paired plugin…";
    try {
      const response = await fetchImpl("/api/koreader/plugin.zip", {
        method: "POST",
      });
      if (!response.ok)
        throw new Error(
          (await response.text()).trim() || `HTTP ${response.status}`,
        );
      const href = URL.createObjectURL(await response.blob());
      const link = document.createElement("a");
      link.href = href;
      link.download = "bobobooks.koplugin.zip";
      link.click();
      setTimeout(() => URL.revokeObjectURL(href), 30000);
      status.textContent =
        "Plugin ready. Extract it into koreader/plugins, then restart KOReader.";
    } catch (error) {
      status.textContent = `Could not create plugin: ${error.message}`;
    } finally {
      button.disabled = false;
    }
  }

  return {
    bind() {
      Object.values(fields).forEach((input) =>
        input.addEventListener("input", onDirty),
      );
      fields.provider.addEventListener("change", render);
      $("#download-plugin").onclick = downloadPlugin;
    },
    load(value = {}, valuePublicURL = "") {
      publicURL = valuePublicURL;
      fields.provider.value = value.provider || "built-in";
      fields.bridge.checked = value.web_reader_bridge !== false;
      fields.serverURL.value =
        value.server_url || "https://sync.koreader.rocks";
      fields.username.value = value.username || value.effective_username || "";
      fields.password.value = "";
      fields.password.placeholder = value.password_configured
        ? "Saved password"
        : "Password";
      render();
    },
    async checkAndroidDownload() {
      const link = $("#download-android"),
        version = $("#android-version"),
        status = $("#android-status");
      try {
        const response = await fetchImpl("/api/android/app.apk", {
          method: "HEAD",
          cache: "no-store",
        });
        if (!response.ok) throw new Error(`HTTP ${response.status}`);
        const build = response.headers.get("X-Mochil-App-Version");
        version.textContent = build
          ? `Version ${build} · Android 8 or newer`
          : "Android 8 or newer";
        link.href = "/api/android/app.apk";
        link.removeAttribute("aria-disabled");
        status.textContent = "";
      } catch {
        version.textContent =
          "No Android build is available on this server yet.";
        link.removeAttribute("href");
        link.setAttribute("aria-disabled", "true");
        status.textContent =
          "Ask the server administrator to publish an Android build.";
      }
    },
    write(settings) {
      settings.kosync = {
        ...(settings.kosync || {}),
        ...readingSyncValue({
          provider: fields.provider.value,
          bridge: fields.bridge.checked,
          serverURL: fields.serverURL.value,
          username: fields.username.value,
          password: fields.password.value,
        }),
      };
      if (!fields.username.value.trim()) delete settings.kosync.username;
      if (!fields.password.value) delete settings.kosync.password;
      delete settings.kosync.password_configured;
      delete settings.kosync.effective_username;
    },
  };
}
