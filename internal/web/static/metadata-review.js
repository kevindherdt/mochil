import { mountMetadataApplyDialog } from './metadata-apply-dialog-template.js?v=1'
import { mountMetadataReview } from './metadata-review-template.js?v=2'
import { createMusicQualityReview } from './music-quality-review.js?v=2'
import { createIngestPreview } from './ingest-preview.js?v=1'
import { metadataApplyImpact, metadataApplyReceipt } from './metadata-apply-summary.js?v=2'
import { createPublicationMatcher } from './publication-matcher.js?v=2'

export function createMetadataReview({ document, $, $$, escapeHTML, requestJSON }) {
mountMetadataReview(document)
mountMetadataApplyDialog(document)
let metadataJob
let metadataApplyPreview
let metadataLibraries = []
const musicQuality = createMusicQualityReview({ $, escapeHTML, requestJSON })
const ingestPreview = createIngestPreview({ $, escapeHTML, requestJSON })
const publicationMatcher = createPublicationMatcher({ escapeHTML, requestJSON, getContext: () => ({ job: metadataJob, libraries: metadataLibraries }) })

const metadataIssueLabels = {
	missing_author: 'Missing author',
	missing_title: 'Missing title',
	missing_series_index: 'Missing volume number',
	duplicate_series_index: 'Reading-order conflict',
	duplicate_edition: 'Possible duplicate edition'
}

function selectedMetadataLibrary() {
	return metadataLibraries.find(library => library.id === $('#metadata-library').value)
}

async function inspectMetadata() {
	metadataApplyPreview = undefined
	const libraryID = $('#metadata-library').value
	if (!libraryID) {
		$('#metadata-status').textContent = 'Add a publication shelf before inspecting metadata.'
		return
	}
	const button = $('#inspect-metadata')
	button.disabled = true
	$('#metadata-status').textContent = 'Inspecting the catalogue…'
	$('#metadata-summary').hidden = true
	$('#metadata-issues').innerHTML = ''
	try {
		if (selectedMetadataLibrary()?.kind === 'music') {
			await musicQuality.inspect(libraryID)
			return
		}
		metadataJob = (await requestJSON(`/api/metadata/jobs?library=${encodeURIComponent(libraryID)}`, { method: 'POST' })).data
		renderMetadataJob()
		await loadMetadataJobs()
	} catch (error) {
		$('#metadata-status').textContent = `Inspection failed: ${error.message}`
	} finally {
		button.disabled = false
	}
}

function metadataDecision(issue) { return issue.decision || 'pending' }

function metadataFieldLabel(issue) {
	if (issue.code === 'missing_author') return 'Author'
	if (issue.code === 'missing_title') return 'Title'
	return 'Volume number'
}

function metadataCurrentValue(issue) {
	if (issue.code === 'missing_author') return issue.authors?.filter(Boolean).join(' & ') || 'Not set'
	if (issue.code === 'missing_title') return issue.title || 'Not set'
	return issue.series_index === null || issue.series_index === undefined ? 'Not set' : String(issue.series_index)
}

function duplicateConflicts(issue) {
	if (issue.code !== 'duplicate_series_index') return []
	const series = String(issue.series || '').trim().toLocaleLowerCase()
	return metadataJob.items.filter(candidate => candidate.code === 'duplicate_series_index' && candidate.book_id !== issue.book_id && String(candidate.series || '').trim().toLocaleLowerCase() === series && Number(candidate.series_index) === Number(issue.series_index))
}

function duplicateGroupKey(issue) {
	return `${String(issue.series || '').trim().toLocaleLowerCase()}\u0000${metadataCurrentValue(issue)}`
}

function duplicateGroups(items) {
	const groups = new Map()
	items.filter(issue => issue.code === 'duplicate_series_index').forEach(issue => {
		const key = duplicateGroupKey(issue)
		if (!groups.has(key)) groups.set(key, [])
		groups.get(key).push(issue)
	})
	return groups
}

function authorProposalGroupKey(issue) {
	if (issue.code !== 'missing_author' || issue.proposal?.source !== 'consistent series siblings' || !issue.proposal?.proposed) return ''
	return `${String(issue.series || '').trim().toLocaleLowerCase()}\u0000${issue.proposal.proposed.toLocaleLowerCase()}`
}

function authorProposalGroups(items) {
	const groups = new Map()
	items.forEach(issue => {
		const key = authorProposalGroupKey(issue)
		if (!key) return
		if (!groups.has(key)) groups.set(key, [])
		groups.get(key).push(issue)
	})
	for (const [key, group] of groups) if (group.length < 2) groups.delete(key)
	return groups
}

function authorProposalGroupMarkup(group, locked) {
	const first = group[0]
	const author = first.proposal.proposed
	const pending = group.filter(issue => metadataDecision(issue) === 'pending').length
	const rows = group.map(issue => `<div class="metadata-author-row" data-book="${escapeHTML(issue.book_id)}"><div><strong>${escapeHTML(issue.title || 'Untitled')}</strong><small class="metadata-path">${escapeHTML(issue.path)}</small></div><span>${metadataDecision(issue) === 'accepted' ? 'Correction saved' : metadataDecision(issue) === 'ignored' ? 'Keeping Unknown' : 'Unknown'}</span></div>`).join('')
	const actions = locked ? '' : `<div class="metadata-group-actions"><button data-author-action="save">Use ${escapeHTML(author)} for these books</button><button data-author-action="keep">Keep all as Unknown</button></div>`
	return `<article class="metadata-author-group" data-author="${escapeHTML(author)}"><header><div><strong>${escapeHTML(first.series || 'Unnamed series')}</strong><small>${group.length.toLocaleString()} books are missing an author</small></div><span>${pending ? `${pending.toLocaleString()} to review` : 'Reviewed'}</span></header><div class="metadata-author-proposal"><small>Consistent author on other books in this series</small><strong>${escapeHTML(author)}</strong></div><div class="metadata-author-rows">${rows}</div>${actions}<p class="metadata-group-status" role="status"></p></article>`
}

function duplicateGroupMarkup(group, locked) {
	const first = group[0]
	const current = metadataCurrentValue(first)
	const pending = group.filter(issue => metadataDecision(issue) === 'pending').length
	const suggestions = group.filter(issue => issue.proposal?.proposed && issue.proposal.proposed !== metadataCurrentValue(issue)).length
	const provider = metadataLibraries.find(library => library.id === metadataJob.library_id)?.metadata_provider
	const providerName = provider === 'hardcover' ? 'Hardcover' : provider === 'openlibrary' ? 'Open Library' : ''
	const rows = group.map(issue => {
		const author = (issue.authors || []).filter(Boolean).join(' & ')
		const value = issue.decision === 'accepted' ? issue.decision_value : issue.proposal?.proposed || metadataCurrentValue(issue)
		const suggested = issue.proposal?.proposed && issue.proposal.proposed !== metadataCurrentValue(issue)
		const lookup = providerName ? `<div class="metadata-online"><button type="button" data-online-lookup>Inspect ${providerName} match</button><div class="metadata-suggestions" role="status"></div></div>` : ''
		return `<div class="metadata-group-row" data-book="${escapeHTML(issue.book_id)}" data-code="${escapeHTML(issue.code)}" data-current="${escapeHTML(metadataCurrentValue(issue))}"><div><strong>${escapeHTML(issue.title || 'Untitled')}</strong><small>${escapeHTML(author || 'Unknown author')}</small><small class="metadata-path">${escapeHTML(issue.path)}</small>${lookup}</div>${locked ? `<b>Volume ${escapeHTML(value)}</b>` : `<label><span>Volume</span><input inputmode="decimal" maxlength="20" value="${escapeHTML(value)}" aria-label="Volume number for ${escapeHTML(issue.title || 'Untitled')}">${suggested ? `<small>Title suggests ${escapeHTML(issue.proposal.proposed)}</small>` : ''}</label>`}</div>`
	}).join('')
	const actions = locked ? '' : `<div class="metadata-group-actions"><button data-group-action="save">Save these volume numbers</button><button data-group-action="keep">Keep all as-is</button></div>`
	const hint = suggestions ? ` · ${suggestions.toLocaleString()} title ${suggestions === 1 ? 'match' : 'matches'}` : ''
	return `<article class="metadata-duplicate-group" data-duplicate-group="${escapeHTML(duplicateGroupKey(first))}"><details><summary><span><strong>${escapeHTML(first.series || 'Unnamed series')}</strong><small>${group.length.toLocaleString()} books share reading position ${escapeHTML(current)}${hint}</small></span><b>${pending ? `${pending.toLocaleString()} to review` : 'Reviewed'}</b></summary><div class="metadata-duplicate-group-body"><p>These books share one series position. Mochil cannot safely choose their order; enter the intended positions or keep them unchanged.</p><div class="metadata-group-rows">${rows}</div>${actions}<p class="metadata-group-status" role="status"></p></div></details></article>`
}

function metadataReviewMarkup(issue, state, locked) {
	const fieldLabel = metadataFieldLabel(issue)
	const current = metadataCurrentValue(issue)
	const proposed = issue.decision_value || issue.proposal?.proposed || ''
	const conflicts = duplicateConflicts(issue)
	const conflictMarkup = conflicts.length ? `<div class="metadata-conflicts"><h4>Also marked volume ${escapeHTML(current)}</h4>${conflicts.map(conflict => `<div><strong>${escapeHTML(conflict.title || 'Untitled')}</strong><small>${escapeHTML((conflict.authors || []).join(' & ') || conflict.path)}</small><small class="metadata-path">${escapeHTML(conflict.path)}</small></div>`).join('')}</div>` : ''
	const comparison = `<div class="metadata-comparison"><div><small>Current ${escapeHTML(fieldLabel.toLowerCase())}</small><strong>${escapeHTML(current)}</strong></div><div><small>${proposed ? 'Proposed correction' : 'Correction'}</small><strong>${proposed ? escapeHTML(proposed) : 'Enter below'}</strong></div></div>`
	if (issue.code === 'duplicate_edition') {
		if (locked) return '<p>Kept as an intentional edition.</p>'
		const reviewed = state === 'ignored'
		const provider = metadataLibraries.find(library => library.id === metadataJob.library_id)?.metadata_provider
		const providerName = provider === 'hardcover' ? 'Hardcover' : provider === 'openlibrary' ? 'Open Library' : ''
		const lookup = providerName ? `<div class="metadata-online"><button type="button" data-online-lookup>Inspect ${providerName} match</button><small>This lookup does not merge or delete either edition.</small><div class="metadata-suggestions" role="status"></div></div>` : ''
		return `<details class="metadata-decision" ${reviewed ? 'open' : ''}><summary>${reviewed ? 'Intentional edition' : 'Choose what to do'}</summary><div>${lookup}<p>${reviewed ? 'This choice will be remembered by future inspections.' : 'Mochil will keep both files. This only classifies the finding.'}</p><div class="metadata-decision-actions"><button data-decision="${reviewed ? 'pending' : 'ignored'}">${reviewed ? 'Review again' : 'Keep both editions'}</button></div></div></details>`
	}
	if (locked) return comparison + conflictMarkup
	const inputMode = fieldLabel === 'Volume number' ? ' inputmode="decimal"' : ''
	const provider = metadataLibraries.find(library => library.id === metadataJob.library_id)?.metadata_provider
	const providerName = provider === 'hardcover' ? 'Hardcover' : provider === 'openlibrary' ? 'Open Library' : ''
	const lookup = providerName && ['missing_author', 'missing_title'].includes(issue.code) ? `<div class="metadata-online"><button type="button" data-online-lookup>Search ${providerName}</button><small>Shares this title${issue.authors?.some(author => author && author.toLowerCase() !== 'unknown') ? ' and author' : ''} only when pressed.</small><div class="metadata-suggestions" role="status"></div></div>` : ''
	return `${comparison}${conflictMarkup}<details class="metadata-decision" ${state === 'pending' ? '' : 'open'}><summary>${state === 'pending' ? 'Choose what to do' : 'Change decision'}</summary><div>${lookup}<label>${escapeHTML(fieldLabel)}<input value="${escapeHTML(proposed)}" maxlength="500"${inputMode} aria-label="Correct ${escapeHTML(fieldLabel.toLowerCase())}" placeholder="Enter the correct ${escapeHTML(fieldLabel.toLowerCase())}"></label><div class="metadata-decision-actions"><button data-decision="accepted" ${proposed ? '' : 'disabled'}>Save correction</button><button data-decision="ignored">Keep as-is</button></div></div></details>`
}

function renderMetadataJob() {
	if (!metadataJob) return
	const allDuplicateGroups = duplicateGroups(metadataJob.items)
	const counts = Object.entries(metadataJob.counts || {}).filter(([, count]) => count > 0)
	$('#metadata-summary').hidden = counts.length === 0
	$('#metadata-summary').innerHTML = counts.map(([code, count]) => code === 'duplicate_series_index' ? `<span><strong>${allDuplicateGroups.size.toLocaleString()}</strong>Reading-order conflict ${allDuplicateGroups.size === 1 ? 'group' : 'groups'}<small>${count.toLocaleString()} affected books</small></span>` : `<span><strong>${count.toLocaleString()}</strong>${escapeHTML(metadataIssueLabels[code] || code)}</span>`).join('')
	const filter = $('#metadata-filter').value
	const matching = metadataJob.items.filter(issue => filter === 'all' || metadataDecision(issue) === filter)
	const matchingDuplicateKeys = new Set(matching.filter(issue => issue.code === 'duplicate_series_index').map(duplicateGroupKey))
	const matchingAuthorGroups = authorProposalGroups(matching)
	const groupedAuthorBooks = new Set([...matchingAuthorGroups.values()].flat().map(issue => issue.book_id))
	const individual = matching.filter(issue => issue.code !== 'duplicate_series_index' && !groupedAuthorBooks.has(issue.book_id))
	const unresolved = metadataJob.items.filter(issue => metadataDecision(issue) === 'pending').length
	const batchable = metadataJob.items.filter(isHighConfidenceVolumeCorrection).slice(0, 500)
	const locked = metadataJob.status === 'applied'
	$('#metadata-filter-wrap').hidden = locked || metadataJob.issue_count === 0
	const remembered = metadataJob.inherited_decisions ? ` · ${metadataJob.inherited_decisions.toLocaleString()} remembered` : ''
	$('#metadata-status').textContent = locked ? `${metadataJob.books_inspected.toLocaleString()} books inspected · changes applied` : `${metadataJob.books_inspected.toLocaleString()} books inspected · ${unresolved.toLocaleString()} ${unresolved === 1 ? 'issue' : 'issues'} left to review${remembered}`
	renderBatchAction(batchable, locked)
	$('#metadata-apply').hidden = metadataJob.status !== 'resolved'
	$('#metadata-apply').innerHTML = metadataJob.status === 'resolved' ? '<button id="preview-metadata-apply">Review apply plan</button><p>Nothing changes until you confirm a supported apply plan.</p>' : ''
	$('#preview-metadata-apply')?.addEventListener('click', previewMetadataApply)
	const groupMarkup = [...allDuplicateGroups.entries()].filter(([key]) => matchingDuplicateKeys.has(key)).map(([, group]) => duplicateGroupMarkup(group, locked))
	const authorGroupMarkup = [...matchingAuthorGroups.values()].map(group => authorProposalGroupMarkup(group, locked))
	const individualMarkup = individual.slice(0, 100).map(issue => {
		const state = metadataDecision(issue)
		const evidence = issue.proposal ? `${escapeHTML(issue.proposal.confidence)}-confidence suggestion from ${escapeHTML(issue.proposal.source)}.` : 'No reliable correction could be inferred automatically.'
		const author = (issue.authors || []).filter(Boolean).join(' & ')
		const decisionLabel = state === 'pending' ? metadataIssueLabels[issue.code] || issue.code : state === 'accepted' ? 'Correction saved' : issue.code === 'duplicate_edition' ? 'Intentional edition' : 'Keeping current value'
		return `<article data-book="${escapeHTML(issue.book_id)}" data-code="${escapeHTML(issue.code)}"><div class="metadata-book"><strong>${escapeHTML(issue.title || 'Untitled')}</strong><small>${escapeHTML(author || issue.series || issue.path)}</small><small class="metadata-path">${escapeHTML(issue.path)}</small></div><span>${escapeHTML(decisionLabel)}</span><p>${escapeHTML(issue.reason)} ${evidence}</p>${metadataReviewMarkup(issue, state, locked)}</article>`
	})
	const rendered = [...authorGroupMarkup, ...groupMarkup, ...individualMarkup]
	$('#metadata-issues').innerHTML = rendered.length ? rendered.join('') : `<p class="empty-state">${metadataJob.issue_count ? 'No issues match this filter.' : 'No obvious metadata problems found.'}</p>`
	if (individual.length > 100) $('#metadata-issues').insertAdjacentHTML('beforeend', `<p class="metadata-more">Showing 100 of ${individual.length.toLocaleString()} matching individual issues.</p>`)
	$$('.metadata-decision').forEach(details => {
		const input = details.querySelector('input')
		const accept = details.querySelector('[data-decision="accepted"]')
		if (input && accept) input.oninput = () => { accept.disabled = !input.value.trim() }
		details.querySelectorAll('[data-decision]').forEach(button => button.onclick = event => { event.preventDefault(); saveMetadataDecision(details.closest('article'), button.dataset.decision, input?.value || '') })
		details.querySelector('[data-online-lookup]')?.addEventListener('click', event => { event.preventDefault(); publicationMatcher.search(details.closest('article'), details) })
	})
	$$('.metadata-duplicate-group').forEach(group => {
		group.querySelector('[data-group-action="save"]')?.addEventListener('click', () => saveDuplicateGroup(group, false))
		group.querySelector('[data-group-action="keep"]')?.addEventListener('click', () => saveDuplicateGroup(group, true))
		group.querySelectorAll('.metadata-group-row').forEach(row => row.querySelector('[data-online-lookup]')?.addEventListener('click', event => { event.preventDefault(); publicationMatcher.search(row, row) }))
	})
	$$('.metadata-author-group').forEach(group => {
		group.querySelector('[data-author-action="save"]')?.addEventListener('click', () => saveAuthorGroup(group, false))
		group.querySelector('[data-author-action="keep"]')?.addEventListener('click', () => saveAuthorGroup(group, true))
	})
}

function isHighConfidenceVolumeCorrection(issue) {
	return metadataDecision(issue) === 'pending' && issue.proposal?.field === 'series_index' && issue.proposal?.source === 'title volume marker' && issue.proposal?.confidence === 'high' && issue.proposal?.proposed
}

function renderBatchAction(issues, locked) {
	document.querySelector('#metadata-batch')?.remove()
	if (locked || !issues.length) return
	const action = document.createElement('div')
	action.id = 'metadata-batch'
	action.className = 'metadata-batch'
	action.innerHTML = `<div><strong>${issues.length.toLocaleString()} clear volume ${issues.length === 1 ? 'match' : 'matches'}</strong><small>Explicit volume or book numbers found in titles.</small></div><button>Accept ${issues.length === 1 ? 'correction' : 'corrections'}</button><p role="status"></p>`
	$('#metadata-summary').insertAdjacentElement('afterend', action)
	action.querySelector('button').onclick = () => acceptHighConfidenceVolumes(action, issues)
}

async function acceptHighConfidenceVolumes(action, issues) {
	const button = action.querySelector('button')
	const status = action.querySelector('[role="status"]')
	button.disabled = true
	button.textContent = 'Saving…'
	status.textContent = ''
	try {
		const decisions = issues.map(issue => ({ book_id: issue.book_id, code: issue.code, decision: 'accepted', value: issue.proposal.proposed }))
		metadataJob = (await requestJSON(`/api/metadata/jobs/${encodeURIComponent(metadataJob.id)}/items`, { method: 'PUT', headers: {'Content-Type':'application/json'}, body: JSON.stringify({ decisions }) })).data
		renderMetadataJob()
		await loadMetadataJobs()
	} catch (error) {
		status.textContent = `Could not save these corrections: ${error.message}`
		button.disabled = false
		button.textContent = 'Try again'
	}
}

async function saveAuthorGroup(group, keepAll) {
	const buttons = group.querySelectorAll('button')
	const status = group.querySelector('.metadata-group-status')
	const bookIDs = new Set([...group.querySelectorAll('.metadata-author-row')].map(row => row.dataset.book))
	const issues = metadataJob.items.filter(issue => issue.code === 'missing_author' && bookIDs.has(issue.book_id))
	buttons.forEach(button => { button.disabled = true })
	status.textContent = keepAll ? 'Keeping the current authors…' : 'Saving the author corrections…'
	try {
		const decisions = issues.map(issue => ({ book_id: issue.book_id, code: issue.code, decision: keepAll ? 'ignored' : 'accepted', value: keepAll ? '' : group.dataset.author }))
		metadataJob = (await requestJSON(`/api/metadata/jobs/${encodeURIComponent(metadataJob.id)}/items`, { method: 'PUT', headers: {'Content-Type':'application/json'}, body: JSON.stringify({ decisions }) })).data
		renderMetadataJob()
		await loadMetadataJobs()
	} catch (error) {
		status.textContent = `Could not save this group: ${error.message}`
		buttons.forEach(button => { button.disabled = false })
	}
}

async function saveDuplicateGroup(group, keepAll) {
	const buttons = group.querySelectorAll('button')
	const status = group.querySelector('.metadata-group-status')
	const rows = [...group.querySelectorAll('.metadata-group-row')]
	buttons.forEach(button => { button.disabled = true })
	status.textContent = keepAll ? 'Keeping the current numbers…' : 'Saving the volume numbers…'
	try {
		const decisions = rows.map(row => {
			const input = row.querySelector('input')
			const value = keepAll ? '' : input.value.trim()
			if (!keepAll && !value) throw new Error(`Enter a volume number for ${row.querySelector('strong').textContent}.`)
			const decision = keepAll || value === row.dataset.current ? 'ignored' : 'accepted'
			return { book_id: row.dataset.book, code: row.dataset.code, decision, value }
		})
		metadataJob = (await requestJSON(`/api/metadata/jobs/${encodeURIComponent(metadataJob.id)}/items`, { method: 'PUT', headers: {'Content-Type':'application/json'}, body: JSON.stringify({ decisions }) })).data
		renderMetadataJob()
		await loadMetadataJobs()
	} catch (error) {
		status.textContent = `Could not save this group: ${error.message}`
		buttons.forEach(button => { button.disabled = false })
	}
}

async function previewMetadataApply() {
	const button = $('#preview-metadata-apply')
	button.disabled = true
	$('#metadata-status').textContent = 'Preparing apply plan…'
	try {
		const preview = (await requestJSON(`/api/metadata/jobs/${encodeURIComponent(metadataJob.id)}/preview`)).data
		metadataApplyPreview = preview
		const applicable = preview.changes.length
		$('#metadata-status').textContent = `${applicable.toLocaleString()} supported ${applicable === 1 ? 'change' : 'changes'} · ${preview.unsupported.toLocaleString()} unavailable`
		const canApply = preview.mode !== 'unavailable'
		const impact = metadataApplyImpact(preview)
		$('#metadata-apply').innerHTML = `<strong>${escapeHTML(preview.source_type)} library</strong><p>${escapeHTML(preview.message)}</p>${impact ? `<p><b>File impact:</b> ${escapeHTML(impact)}</p>` : ''}${applicable ? `<details><summary>See ${applicable.toLocaleString()} proposed ${applicable === 1 ? 'change' : 'changes'}</summary><div>${preview.changes.slice(0, 100).map(change => `<span><b>${escapeHTML(change.title)}</b><small>${escapeHTML(change.field.replace('_', ' '))} → ${escapeHTML(change.value)} · ${escapeHTML(change.mode)}</small></span>`).join('')}</div></details>` : ''}${canApply && applicable && !preview.unsupported ? '<button id="start-metadata-apply" class="danger">Apply these changes…</button>' : ''}`
		$('#start-metadata-apply')?.addEventListener('click', openMetadataApplyConfirmation)
	} catch (error) {
		$('#metadata-status').textContent = `Could not prepare apply plan: ${error.message}`
		button.disabled = false
	}
}

function openMetadataApplyConfirmation() {
	const expected = `APPLY ${metadataJob.id}`
	$('#metadata-confirmation-label').textContent = expected
	$('#metadata-confirmation').value = ''
	$('#metadata-apply-status').textContent = ''
	$('#confirm-metadata-apply').disabled = true
	$('#metadata-apply-dialog').showModal()
	$('#metadata-confirmation').focus()
}

async function applyMetadataJob() {
	const expected = `APPLY ${metadataJob.id}`
	if ($('#metadata-confirmation').value !== expected) return
	const button = $('#confirm-metadata-apply')
	button.disabled = true
	$('#metadata-apply-status').textContent = 'Applying, verifying, and preparing rollback copies…'
	try {
		const result = (await requestJSON(`/api/metadata/jobs/${encodeURIComponent(metadataJob.id)}/apply`, { method: 'POST', headers: {'Content-Type':'application/json'}, body: JSON.stringify({ confirmation: expected }) })).data
		metadataJob = result.job
		$('#metadata-apply-dialog').close()
		renderMetadataJob()
		$('#metadata-status').textContent = metadataApplyReceipt(result, metadataApplyPreview)
		await loadMetadataJobs()
	} catch (error) {
		$('#metadata-apply-status').textContent = `Apply stopped: ${error.message}`
		button.disabled = false
	}
}

async function saveMetadataDecision(article, decision, value) {
	const buttons = article.querySelectorAll('button')
	buttons.forEach(button => { button.disabled = true })
	$('#metadata-status').textContent = 'Saving review decision…'
	try {
		const path = `/api/metadata/jobs/${encodeURIComponent(metadataJob.id)}/items/${encodeURIComponent(article.dataset.book)}/${encodeURIComponent(article.dataset.code)}`
		metadataJob = (await requestJSON(path, { method: 'PUT', headers: {'Content-Type':'application/json'}, body: JSON.stringify({ decision, value }) })).data
		renderMetadataJob()
		await loadMetadataJobs()
	} catch (error) {
		$('#metadata-status').textContent = `Could not save decision: ${error.message}`
		buttons.forEach(button => { button.disabled = false })
	}
}

async function openMetadataJob(id) {
	metadataApplyPreview = undefined
	$('#metadata-status').textContent = 'Opening saved inspection…'
	try {
		metadataJob = (await requestJSON(`/api/metadata/jobs/${encodeURIComponent(id)}`)).data
		$('#metadata-filter').value = 'pending'
		renderMetadataJob()
	} catch (error) {
		$('#metadata-status').textContent = `Could not open inspection: ${error.message}`
	}
}

async function loadMetadataJobs() {
	const libraryID = $('#metadata-library').value
	if (!libraryID) { $('#metadata-jobs').innerHTML = '<p>No publication shelf selected.</p>'; return }
	if (selectedMetadataLibrary()?.kind === 'music') { $('#metadata-jobs').innerHTML = ''; return }
	try {
		const jobs = (await requestJSON(`/api/metadata/jobs?library=${encodeURIComponent(libraryID)}`)).data
		$('#metadata-jobs').innerHTML = jobs.length ? jobs.slice(0, 8).map(job => `<button data-job="${escapeHTML(job.id)}"><span><strong>${new Intl.DateTimeFormat(undefined, { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(job.created_at))}</strong><small>${job.books_inspected.toLocaleString()} books inspected</small></span><b>${job.status === 'applied' ? 'Applied' : job.status === 'resolved' ? 'Resolved' : `${job.issue_count.toLocaleString()} issues`}</b></button>`).join('') : '<p>No saved inspections yet.</p>'
		$$('#metadata-jobs [data-job]').forEach(button => button.onclick = () => openMetadataJob(button.dataset.job))
	} catch (error) {
		$('#metadata-jobs').innerHTML = `<p>Could not load inspection history: ${escapeHTML(error.message)}</p>`
	}
}

function setLibraries(libraries) {
	metadataLibraries = libraries
	const metadataLibrary = $('#metadata-library')
	const current = metadataLibrary.value
	metadataLibrary.innerHTML = libraries
		.filter(library => ['books', 'lightnovels', 'comics', 'manga', 'mixed', 'music'].includes(library.kind || 'books'))
		.map(library => `<option value="${escapeHTML(library.id)}">${escapeHTML(library.name)}</option>`).join('')
	if (current && [...metadataLibrary.options].some(option => option.value === current)) metadataLibrary.value = current
	updateMetadataMode()
}

function updateMetadataMode() {
	const music = selectedMetadataLibrary()?.kind === 'music'
	document.querySelector('#metadata-batch')?.remove()
	$('.metadata-ingest-preview').hidden = music
	$('.metadata-history').hidden = music
	$('#metadata-filter-wrap').hidden = true
	$('#music-quality-filter-wrap').hidden = true
	$('#metadata-apply').hidden = true
	$('#inspect-metadata').textContent = music ? 'Check music quality' : 'New inspection'
	$('#metadata-intro').textContent = music ? 'Review a read-only health report for one music shelf. Tags and media files are never changed.' : 'Inspect one library at a time. Nothing changes until you review and confirm it.'
	$('#metadata-status').textContent = music ? 'Check tags and known artwork for this music shelf.' : ''
	musicQuality.reset()
	$('#metadata-summary').hidden = true
	$('#metadata-issues').innerHTML = ''
	if (!music) loadMetadataJobs()
}

function bind() {
	$('#inspect-metadata').onclick = inspectMetadata
	$('#metadata-library').onchange = () => { ingestPreview.reset(); updateMetadataMode() }
	$('#preview-ingest').onclick = () => ingestPreview.load(true)
	$('#ingest-preview-more').onclick = () => ingestPreview.load()
	$('#metadata-filter').onchange = renderMetadataJob
	$('#music-quality-filter').onchange = musicQuality.render
	$('#metadata-confirmation').oninput = () => { $('#confirm-metadata-apply').disabled = $('#metadata-confirmation').value !== `APPLY ${metadataJob?.id}` }
	$('#confirm-metadata-apply').onclick = applyMetadataJob
}

return { bind, setLibraries }
}
