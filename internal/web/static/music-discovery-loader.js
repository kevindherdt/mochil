import { musicLoadingHTML } from "./music-loading.js?v=1";

export function createMusicDiscoveryLoader({
  requestJSON,
  container,
  setStatus,
  getLibrary,
  getFavorites,
  onDiscovery,
  onRecent,
}) {
  let revision = 0;
  const cache = new Map();
  const begin = () => ++revision;
  const isCurrent = (ticket) => ticket === revision;
  const cancel = () => revision++;
  const safe = async (path) => {
    try {
      return { data: (await requestJSON(path)).data || [], failed: false };
    } catch {
      return { data: [], failed: true };
    }
  };

  async function loadDiscovery(ticket) {
    if (!isCurrent(ticket)) return false;
    setStatus("");
    const libraryID = getLibrary();
    const cached = cache.get(libraryID);
    if (cached) onDiscovery({ ...cached, favorites: getFavorites().slice(0, 8) });
    else container.innerHTML = musicLoadingHTML();
    const library = encodeURIComponent(libraryID);
    const [played, recent, albums, artists] = await Promise.all([
      safe(`/api/audio-progress?music_library=${library}&limit=8`),
      safe(`/api/audio?library=${library}&limit=8&sort=recent`),
      safe(`/api/audio-groups?library=${library}&group_type=albums&limit=10`),
      safe(`/api/audio-groups?library=${library}&group_type=artists&limit=10`),
    ]);
    if (!isCurrent(ticket)) return false;
    const discovery = {
      played: played.failed ? cached?.played || [] : played.data,
      recent: recent.failed ? cached?.recent || [] : recent.data,
      favorites: getFavorites().slice(0, 8),
      albums: albums.failed ? cached?.albums || [] : albums.data,
      artists: artists.failed ? cached?.artists || [] : artists.data,
      failures: {
        played: played.failed,
        recent: recent.failed,
        albums: albums.failed,
        artists: artists.failed,
      },
    };
    cache.set(libraryID, discovery);
    onDiscovery(discovery);
    return true;
  }

  async function loadRecent(ticket) {
    if (!isCurrent(ticket)) return false;
    setStatus("");
    const libraryID = getLibrary();
    const cached = cache.get(libraryID);
    if (cached) onRecent(cached.played);
    try {
      const library = encodeURIComponent(libraryID);
      const played =
        (await requestJSON(
          `/api/audio-progress?music_library=${library}&limit=20`,
        )).data || [];
      if (!isCurrent(ticket)) return false;
      cache.set(libraryID, { ...(cached || {}), played });
      onRecent(played);
      return true;
    } catch (error) {
      if (!isCurrent(ticket)) return false;
      if (!cached) onRecent([]);
      setStatus(`Could not load recently played: ${error.message}`);
      return false;
    }
  }

  return { begin, isCurrent, cancel, loadDiscovery, loadRecent };
}
