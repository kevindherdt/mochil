// An install with nothing configured has no shelf to show, so the app opens Settings, where a folder
// can be added. It cannot be a server redirect: "/" and "/#/settings" are the same request because a
// browser never sends the fragment, and that is what looped Chrome into ERR_TOO_MANY_REDIRECTS.
export function openSettingsWhenEmpty({ location, fetchImpl = (...args) => fetch(...args) } = {}) {
  if (!location || location.pathname !== "/" || location.hash) return;
  fetchImpl("/api/libraries", { headers: { Accept: "application/json" } })
    .then((response) => (response.ok ? response.json() : []))
    .then((list) => {
      if (Array.isArray(list) && list.length === 0) location.hash = "#/settings";
    })
    .catch(() => {});
}
