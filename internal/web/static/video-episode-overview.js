// Hover and focus preview; only the episode button starts playback.
let nextDescriptionID = 0;
export function bindEpisodeOverview(root, episodes) {
  const buttons = [...root.querySelectorAll('[data-episode-index]')];
  const title = root.querySelector('[data-episode-title]');
  const description = root.querySelector('[data-episode-description]');
  const toggle = root.querySelector('[data-episode-toggle]');
  const overview = root.querySelector('.video-episode-overview');
  description.id = `episode-preview-description-${++nextDescriptionID}`;
  toggle.setAttribute('aria-describedby', description.id);
  toggle.hidden = false;
  let current = null;
  const hasOverflow = () => description.scrollHeight > description.clientHeight;
  const applyToggle = () => {
    // Keep the toggle in the layout always and hide it with visibility (not display:none)
    // so the preview height never shifts when switching episodes.
    toggle.hidden = false;
    toggle.classList.toggle('is-hidden', !hasOverflow());
  };
  const setExpanded = expanded => {
    overview.classList.toggle('is-expanded', expanded);
    toggle.setAttribute('aria-expanded', String(expanded));
    // Hide only when collapsed and nothing needs expanding.
    toggle.classList.toggle('is-hidden', !expanded && !hasOverflow());
    toggle.setAttribute('aria-label', expanded ? 'Hide full description' : 'Show full description');
  };
  const preview = button => {
    const index = Number(button?.dataset.episodeIndex);
    const episode = button && Number.isInteger(index) ? episodes[index] : null;
    if (!episode || button === current) return;
    current = button;
    setExpanded(false);
    title.textContent = episode.displayTitle;
    description.textContent = typeof episode.overview === 'string' && episode.overview.trim() ? episode.overview : 'No description available.';
    applyToggle();
  };
  const fromEvent = event => event.target.closest?.('[data-episode-index]') || event.target.closest?.('.video-episode-card')?.querySelector('[data-episode-index]');
  root.addEventListener('focusin', event => { const b = fromEvent(event); if (b) preview(b); });
  root.addEventListener('pointerover', event => { if (event.pointerType === 'mouse') { const b = fromEvent(event); if (b) preview(b); } });
  root.addEventListener('click', event => {
    const target = event.target.closest?.('[data-episode-toggle]');
    if (target) setExpanded(target.getAttribute('aria-expanded') !== 'true');
  });
  preview(buttons[0]);
}
