import test from 'node:test'
import assert from 'node:assert/strict'
import { musicQualityView } from '../static/music-quality-review.js'

const escapeHTML = value => String(value).replaceAll('&', '&amp;').replaceAll('<', '&lt;').replaceAll('>', '&gt;')

test('music quality view stays compact and escapes report values', () => {
	const view = musicQualityView({
		tracks: 12,
		albums: 2,
		counts: { unknown_artist: 1 },
		issues: [{ code: 'unknown_artist', title: '<Unknown>', detail: 'Missing & unsafe', paths: ['Music/<track>.flac'] }],
	}, escapeHTML)
	assert.match(view.summary, />12<\/strong>Tracks/)
	assert.match(view.summary, />2<\/strong>Album folders/)
	assert.match(view.summary, /Unknown artists/)
	assert.match(view.issues, /&lt;Unknown&gt;/)
	assert.match(view.issues, /Missing &amp; unsafe/)
	assert.doesNotMatch(view.issues, /<Unknown>/)
	assert.match(view.issues, /Music\/&lt;track&gt;\.flac/)
})

test('music quality view filters a bounded category without losing totals', () => {
	const report = {
		tracks: 2, albums: 1, counts: { unknown_artist: 1, duplicate_track: 1 },
		issues: [
			{ code: 'unknown_artist', title: 'Unknown', detail: 'Missing' },
			{ code: 'duplicate_track', title: 'Duplicate', detail: 'Repeated' },
		],
	}
	const view = musicQualityView(report, escapeHTML, 'duplicate_track')
	assert.match(view.summary, /Unknown artists/)
	assert.match(view.issues, /Duplicate/)
	assert.doesNotMatch(view.issues, />Unknown</)
})

test('music quality separates likely copies from tag identity conflicts', () => {
	const view = musicQualityView({
		tracks: 2, albums: 1,
		counts: { duplicate_file_candidate: 1, track_identity_conflict: 1 },
		issues: [],
	}, escapeHTML)
	assert.match(view.summary, /Likely file copies/)
	assert.match(view.summary, /Track identity conflicts/)
})

test('music quality view has a useful empty state', () => {
	assert.match(musicQualityView({ tracks: 0, albums: 0 }, escapeHTML).issues, /No obvious music tag problems/)
})

test('artwork findings describe the combined embedded and sidecar check', () => {
	const view = musicQualityView({
		tracks: 1, albums: 1, counts: { missing_artwork: 1 },
		issues: [{ code: 'missing_artwork', title: 'Album', detail: 'No known embedded artwork or sidecar was found.' }],
	}, escapeHTML)
	assert.match(view.summary, /No known artwork/)
	assert.match(view.issues, /No known embedded artwork or sidecar/)
})
