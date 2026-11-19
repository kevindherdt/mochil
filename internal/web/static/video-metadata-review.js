export function createVideoMetadataReview({ requestJSON, escapeHTML, refresh }) {
	function bind(button, context) {
		button.onclick = () => context.previous ? manage(button, context) : search(button, context)
	}

	function endpoint(context) {
		return context.show
			? `/api/video-show-metadata/${encodeURIComponent(context.libraryID)}?show=${encodeURIComponent(context.show)}`
			: `/api/video-metadata/${encodeURIComponent(context.libraryID)}/${encodeURIComponent(context.itemID)}`
	}

	function manage(button, context) {
		const target = button.parentElement.querySelector('[data-metadata-results]')
		const value = context.previous
		const mediaType = context.show ? 'tv' : 'movie'
		const source = `https://www.themoviedb.org/${mediaType}/${encodeURIComponent(value.provider_id)}`
		const fields = [value.title && `Title: ${value.title}`, value.year && `Year: ${value.year}`, value.overview && 'Synopsis', value.artwork_url && 'Poster'].filter(Boolean)
		target.hidden = false
		target.innerHTML = `<div class="video-metadata-current"><small>Saved from TMDB (${escapeHTML(value.provider_id)})${fields.length ? ` · ${escapeHTML(fields.filter(field => !field.startsWith('Title') && !field.startsWith('Year')).join(', ').toLowerCase())}` : ''}</small><div class="video-metadata-actions"><button type="button" data-find-replacement>Find replacement</button><a href="${escapeHTML(source)}" target="_blank" rel="noopener noreferrer">Review source</a><button type="button" class="remove" data-remove-metadata>Remove saved details</button></div><div class="video-metadata-confirm" hidden><p>Restore the metadata inferred from your files? Cached artwork may be reclaimed later.</p><button type="button" class="danger" data-confirm-remove>Confirm removal</button><button type="button" data-cancel-remove>Cancel</button></div><small role="status"></small></div>`
		target.querySelector('[data-find-replacement]').onclick = () => search(button, context)
		const confirmation = target.querySelector('.video-metadata-confirm')
		target.querySelector('[data-remove-metadata]').onclick = () => { confirmation.hidden = false; confirmation.querySelector('[data-confirm-remove]').focus() }
		target.querySelector('[data-cancel-remove]').onclick = () => { confirmation.hidden = true; button.focus() }
		target.querySelector('[data-confirm-remove]').onclick = () => remove(context, confirmation, target.querySelector('[role=status]'))
	}

	async function remove(context, confirmation, status) {
		confirmation.querySelectorAll('button').forEach(button => { button.disabled = true })
		status.textContent = 'Removing saved details…'
		try {
			await requestJSON(endpoint(context), { method: 'DELETE' })
			status.textContent = 'Removed. Restoring inferred metadata…'
			await refresh()
		} catch (error) {
			status.textContent = `Could not remove these details: ${error.message}`
			confirmation.querySelectorAll('button').forEach(button => { button.disabled = false })
		}
	}

	async function search(button, context) {
		const target = button.parentElement.querySelector('[data-metadata-results]')
		button.disabled = true
		target.hidden = false
		target.textContent = 'Searching TMDB…'
		try {
			const query = new URLSearchParams({ library: context.libraryID, video: context.itemID })
			const response = (await requestJSON(`/api/metadata/suggestions?${query}`)).data
			const results = response.results || []
			target.innerHTML = results.length ? resultMarkup(response, results) : '<p>No close matches were returned. Your catalogue has not changed.</p>'
			target.querySelectorAll('[data-tmdb-result]').forEach(row => {
				row.querySelector('[data-apply-metadata]').onclick = () => apply(row, results[Number(row.dataset.tmdbResult)], context)
			})
			button.textContent = 'Search again'
		} catch (error) {
			target.textContent = `Could not search TMDB: ${error.message}`
			button.textContent = 'Try again'
		} finally { button.disabled = false }
	}

	function resultMarkup(response, results) {
		return `<p>Shared with TMDB: <b>${escapeHTML(response.shared.title)}</b></p>${results.map((result, index) => `<article data-tmdb-result="${index}">${result.cover_url ? `<img src="/api/metadata/suggestion-artwork?url=${encodeURIComponent(result.cover_url)}" alt="" loading="lazy" decoding="async">` : ''}<div><strong>${escapeHTML(result.title)}${result.year ? ` <small>(${result.year})</small>` : ''}</strong>${result.overview ? `<p>${escapeHTML(result.overview)}</p>` : '<p>No synopsis supplied.</p>'}<div class="video-metadata-fields"><label><input type="checkbox" data-field="title" checked> Title</label>${result.year ? '<label><input type="checkbox" data-field="year" checked> Year</label>' : ''}${result.overview ? '<label><input type="checkbox" data-field="overview" checked> Synopsis</label>' : ''}${result.cover_url ? '<label><input type="checkbox" data-field="artwork_url" checked> Poster</label>' : ''}</div><div class="video-metadata-actions"><button type="button" data-apply-metadata>Apply selected fields</button><a href="${escapeHTML(result.source_url)}" target="_blank" rel="noopener noreferrer">Review source</a></div><small role="status"></small></div></article>`).join('')}<small>Only checked fields become Mochil catalogue overrides. Your source videos remain untouched.</small>`
	}

	async function apply(row, result, context) {
		const button = row.querySelector('[data-apply-metadata]')
		const status = row.querySelector('[role=status]')
		const selected = field => row.querySelector(`[data-field="${field}"]`)?.checked
		if (!row.querySelector('[data-field]:checked')) { status.textContent = 'Choose at least one field.'; return }
		const previous = context.previous || {}
		const value = {
			title: selected('title') ? result.title : previous.title || '',
			year: selected('year') ? result.year || 0 : previous.year || 0,
			overview: selected('overview') ? result.overview || '' : previous.overview || '',
			artwork_url: selected('artwork_url') ? result.cover_url || '' : previous.artwork_url || '',
			provider: result.provider,
			provider_id: result.provider_id
		}
		button.disabled = true
		status.textContent = 'Saving selected fields…'
		try {
			await requestJSON(endpoint(context), { method: 'PUT', headers: {'Content-Type':'application/json'}, body: JSON.stringify(value) })
			status.textContent = 'Saved. Refreshing this shelf…'
			await refresh()
		} catch (error) {
			status.textContent = `Could not save these fields: ${error.message}`
			button.disabled = false
		}
	}

	return { bind }
}
