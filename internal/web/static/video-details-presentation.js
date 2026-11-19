// Only a TMDB filename is accepted; the server proxy never receives an arbitrary URL.
function portraitURL(path) {
  return typeof path === 'string' && /^\/[a-zA-Z0-9_-]+\.(?:jpg|png|webp)$/.test(path) && path.length <= 200
    ? `/api/tmdb-portrait?path=${encodeURIComponent(path)}` : '';
}

export function castHTML(cast, escapeHTML) {
  if (!Array.isArray(cast)) return '';
  const members = cast.filter(member => typeof member?.name === 'string' && member.name.trim());
  if (!members.length) return '';
  return `<section class="video-cast" aria-label="Cast"><h3>Cast</h3><ul>${members.map(member => {
    const portrait = portraitURL(member.profile_path);
    return `<li><span class="video-cast-portrait"><span class="video-cast-placeholder" aria-hidden="true">●</span>${portrait ? `<img class="art-img" src="${escapeHTML(portrait)}" alt="" loading="lazy" decoding="async">` : ''}</span><strong>${escapeHTML(member.name)}</strong>${member.character ? `<small>${escapeHTML(member.character)}</small>` : ''}</li>`;
  }).join('')}</ul></section>`;
}

export function titleDetailsSummaryHTML(title, escapeHTML) {
  return `<summary aria-label="Cast and details for ${escapeHTML(title)}">Cast and details</summary>`;
}

export function episodeSynopsisHTML(item, escapeHTML) {
  return item.overview ? `<span class="video-episode-synopsis">${escapeHTML(item.overview)}</span>` : '';
}

export function bindTitleDetails(panel, {libraryID, itemID, show, requestJSON, escapeHTML}) {
  let loaded = false, loading = false;
  panel.addEventListener('toggle', async () => {
    if (!panel.open || loaded || loading) return;
    loading = true;
    const content = panel.querySelector('[data-title-details-content]');
    content.innerHTML = '<p role="status">Loading details…</p>';
    try {
      const query = new URLSearchParams(show ? {show} : {item:itemID});
      const result = await requestJSON(`/api/video-details/${encodeURIComponent(libraryID)}?${query}`);
      const html = castHTML(result.data?.cast, escapeHTML);
      content.innerHTML = html || '<p>No cast details available. Close and reopen to retry.</p>';
      loaded = Boolean(html);
    } catch {
      content.innerHTML = '<p role="status">Details unavailable. Close and reopen to retry.</p>';
    } finally { loading = false; }
  });
}
