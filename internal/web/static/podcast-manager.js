export function createPodcastManager({ $, escapeHTML, requestJSON, onChanged }) {
  let feeds = []
  const subscribed = new Set()
  const moreIcon = '<circle cx="12" cy="5" r="1.4"/><circle cx="12" cy="12" r="1.4"/><circle cx="12" cy="19" r="1.4"/>'
  const bytes = value => {
    const amount = Number(value) || 0
    if (amount < 1024) return `${amount} B`
    const units = ['KiB', 'MiB', 'GiB', 'TiB']
    let size = amount / 1024, index = 0
    while (size >= 1024 && index < units.length - 1) { size /= 1024; index++ }
    return `${size >= 10 ? size.toFixed(0) : size.toFixed(1)} ${units[index]}`
  }
  const retainOptions = value => [1, 3, 5, 10, 20, 50].map(count => `<option value="${count}" ${count === value ? 'selected' : ''}>${count}</option>`).join('')
	const playedRetentionOptions = value => [
		[0, 'Never'], [1, '1 day'], [3, '3 days'], [7, '7 days'], [14, '14 days'], [30, '30 days'], [90, '90 days']
	].map(([days, label]) => `<option value="${days}" ${days === value ? 'selected' : ''}>${label}</option>`).join('')
	const checkedLabel = value => {
		const date = new Date(value)
		if (!value || Number.isNaN(date.getTime())) return ''
		return `Checked ${new Intl.RelativeTimeFormat(undefined, {numeric:'auto'}).format(Math.round((date.getTime() - Date.now()) / 3600000), 'hour')}`
	}

  function updateSearchResults() {
    $('#podcast-results').querySelectorAll('[data-url]').forEach(result => {
      const button = result.querySelector('button'), saved = subscribed.has(result.dataset.url)
      button.textContent = saved ? 'Subscribed' : 'Subscribe'
      button.disabled = saved
      button.classList.toggle('subscribed', saved)
    })
  }

  async function load() {
    try {
      const [feedResult, downloadResult] = await Promise.all([requestJSON('/api/podcast-feeds'), requestJSON('/api/podcast-downloads')])
      feeds = feedResult.data
      subscribed.clear()
      feeds.forEach(feed => subscribed.add(feed.url))
      const usage = `${bytes(downloadResult.data.bytes)} of ${bytes(downloadResult.data.max_bytes)} used · ${downloadResult.data.files} ${downloadResult.data.files === 1 ? 'episode' : 'episodes'} saved`
	  const failed = Number(downloadResult.data.failed) || 0
      $('#podcast-feeds').innerHTML = `<p class="podcast-storage"><span>${escapeHTML(usage)}</span>${failed ? `<button data-retry-downloads>Retry ${failed} failed ${failed === 1 ? 'download' : 'downloads'}</button>` : ''}</p>` + (feeds.map(feed => {
		const policy = feed.policy || {}, retain = Number(policy.retain_downloads) || 3, playedDays = Number(policy.delete_played_after_days) || 0
		const health = feed.last_error ? '<small class="podcast-feed-error">Refresh failed · Saved episodes are still available</small>' : `<small class="podcast-feed-checked">${escapeHTML(checkedLabel(feed.updated))}</small>`
		const policySummary = [policy.auto_download ? `Auto · keep ${retain}` : '', playedDays ? `remove played after ${playedDays}d` : ''].filter(Boolean).join(' · ')
		return `<article data-feed="${escapeHTML(feed.id)}"><span><strong>${escapeHTML(feed.title)}</strong><small>${feed.episode_count} episodes${policySummary ? ` · ${policySummary}` : ''}</small>${health}</span><details class="podcast-feed-options"><summary aria-label="Options for ${escapeHTML(feed.title)}"><svg viewBox="0 0 24 24" aria-hidden="true">${moreIcon}</svg></summary><div><label><input type="checkbox" data-auto ${policy.auto_download ? 'checked' : ''}> Automatic downloads</label><label>Keep newest<select data-retain>${retainOptions(retain)}</select></label><label>Remove played<select data-played-retention aria-label="Remove played downloads after">${playedRetentionOptions(playedDays)}</select></label><button data-save-policy>Save download policy</button><button data-refresh>${feed.last_error ? 'Retry refresh' : 'Refresh now'}</button><button data-remove>Unsubscribe</button></div></details></article>`
      }).join('') || '<small>No subscribed shows yet.</small>')
      updateSearchResults()
    } catch (error) { $('#podcast-status').textContent = error.message }
    return feeds
  }

  async function subscribe(url, button) {
    button.disabled = true
    const previous = button.textContent
    button.textContent = 'Subscribing…'
    try {
      await requestJSON('/api/podcast-feeds', {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify({url})})
      button.textContent = 'Subscribed'
      button.classList.add('subscribed')
      $('#podcast-status').textContent = 'Subscribed.'
      await load()
      onChanged(feeds)
    } catch (error) {
      button.textContent = previous
      button.disabled = false
      $('#podcast-status').textContent = error.message
    }
  }

  $('#podcast-feeds').onclick = async event => {
	const retry = event.target.closest('[data-retry-downloads]')
	if (retry) {
		retry.disabled = true
		retry.textContent = 'Retrying…'
		try {
			const result = await requestJSON('/api/podcast-downloads/retry', {method:'POST'})
			$('#podcast-status').textContent = `${result.data.retried} ${result.data.retried === 1 ? 'download' : 'downloads'} queued again.`
		} catch (_) { $('#podcast-status').textContent = 'Some downloads could not be retried. Try again shortly.' }
		await load()
		return
	}
    const article = event.target.closest('article')
    if (!article) return
    const endpoint = `/api/podcast-feeds/${encodeURIComponent(article.dataset.feed)}`
    if (event.target.closest('[data-save-policy]')) {
	  const button = event.target.closest('[data-save-policy]')
      const auto = article.querySelector('[data-auto]').checked, retain = Number(article.querySelector('[data-retain]').value), playedDays = Number(article.querySelector('[data-played-retention]').value)
	  button.disabled = true
	  try {
		await requestJSON(`${endpoint}/policy`, {method:'PUT', headers:{'Content-Type':'application/json'}, body:JSON.stringify({auto_download:auto, retain_downloads:retain, delete_played_after_days:playedDays})})
		const playedMessage = playedDays ? ` Played downloads are removed after ${playedDays} ${playedDays === 1 ? 'day' : 'days'}.` : ' Played downloads are kept.'
		$('#podcast-status').textContent = `${auto ? `Automatic downloads enabled; keeping the newest ${retain}.` : 'Automatic downloads disabled.'}${playedMessage}`
	  } catch (_) {
		$('#podcast-status').textContent = 'The download policy could not be saved. Try again.'
	  } finally { button.disabled = false }
	} else if (event.target.closest('[data-refresh]')) {
		const button = event.target.closest('[data-refresh]')
		button.disabled = true
		button.textContent = 'Refreshing…'
		$('#podcast-status').textContent = `Refreshing ${article.querySelector('strong').textContent}…`
		try {
			await requestJSON(`${endpoint}/refresh`, {method:'POST'})
			$('#podcast-status').textContent = 'Podcast refreshed.'
		} catch (_) {
			$('#podcast-status').textContent = 'Refresh failed. Saved episodes are still available; retry when the feed is reachable.'
		}
    }
    else if (event.target.closest('[data-remove]') && confirm('Remove this podcast and its saved episodes?')) await requestJSON(endpoint, {method:'DELETE'})
    else return
    await load()
    onChanged(feeds)
  }
  $('#podcast-manage-toggle').onclick = () => {
    const panel = $('#podcast-manage'), open = panel.hidden
    panel.hidden = !open
    $('#podcast-manage-toggle').setAttribute('aria-expanded', open)
  }
  $('#podcast-search').onsubmit = async event => {
    event.preventDefault()
    const button = event.submitter
    button.disabled = true
    $('#podcast-status').textContent = 'Searching…'
    try {
      const {data} = await requestJSON(`/api/podcast-search?q=${encodeURIComponent($('#podcast-query').value.trim())}`)
      $('#podcast-results').innerHTML = data.map(result => {
        const saved = subscribed.has(result.feed_url)
        return `<article data-url="${escapeHTML(result.feed_url)}"><span class="podcast-result-art">${result.artwork_url ? `<img src="${escapeHTML(result.artwork_url)}" alt="" loading="lazy">` : ''}</span><span><strong>${escapeHTML(result.title)}</strong><small>${escapeHTML([result.author,result.genre,result.episode_count ? `${result.episode_count} episodes` : ''].filter(Boolean).join(' · '))}</small></span><button class="${saved ? 'subscribed' : ''}" ${saved ? 'disabled' : ''}>${saved ? 'Subscribed' : 'Subscribe'}</button></article>`
      }).join('') || '<p>No podcasts found.</p>'
      $('#podcast-status').textContent = ''
    } catch (error) { $('#podcast-status').textContent = error.message }
    finally { button.disabled = false }
  }
  $('#podcast-results').onclick = event => {
    const button = event.target.closest('button'), result = button?.closest('article')
    if (result) subscribe(result.dataset.url, button)
  }
  $('#podcast-add').onsubmit = event => {
    event.preventDefault()
    subscribe($('#podcast-url').value.trim(), event.submitter).then(() => { $('#podcast-url').value = '' })
  }

  return { load, get feeds() { return feeds } }
}
