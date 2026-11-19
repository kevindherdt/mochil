// Whether one optional module (by id, e.g. "hermes" or "editor") is turned on, per /api/modules. Shared by every
// place that offers something only when its module is on, so the check and its fallback live in exactly one spot.
export async function moduleEnabled(id, fetchImpl = fetch) {
  try {
    const modules = await (await fetchImpl("/api/modules")).json();
    const list = Array.isArray(modules) ? modules : modules.data || [];
    return list.some((module) => module.id === id && module.enabled);
  } catch (_) {
    return false;
  }
}
