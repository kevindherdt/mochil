import test from 'node:test'
import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'

// The server only accepts POST for this action; a plain GET answered "Method not allowed" on the shelf.
test('bulk TMDB match always asks the server with POST', async () => {
	const source = await readFile(new URL('../static/video-metadata-match.js', import.meta.url), 'utf8')
	const requests = source.match(/requestJSON\([^)]*\)[^)]*\)/g) || []
	assert.ok(requests.length > 0)
	for (const request of requests) assert.match(request, /method: ['"]POST['"]/)
})
