import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'
import { catalogTitleBucket } from '../static/catalog-title-bucket.js'
import { musicTitleBucket } from '../static/music-navigation-policy.js'

test('web catalogue and music buckets follow the shared server/TV contract', () => {
  const cases = JSON.parse(readFileSync(new URL('../../../contracts/navigation_buckets.json', import.meta.url), 'utf8'))
  assert(cases.length > 0)
  for (const { title, bucket } of cases) {
    assert.equal(catalogTitleBucket(title), bucket, title)
    assert.equal(musicTitleBucket(title), bucket, title)
  }
})
