import test from 'node:test'
import assert from 'node:assert/strict'
import { isVaultPage } from '../static/vault-pages.js'

// A note saved as .markdown is a page in the vault tree, the editor and the server; the browser used
// to ask for exactly "md" in two places, which left those notes read-only.
test('both Markdown extensions are vault pages', () => {
  assert.equal(isVaultPage({ format: 'md' }), true)
  assert.equal(isVaultPage({ format: 'markdown' }), true)
  assert.equal(isVaultPage({ format: 'Markdown' }), true)
})

test('a page with no format falls back to its path extension', () => {
  assert.equal(isVaultPage({ path: 'Notes/Longform.markdown' }), true)
  assert.equal(isVaultPage({ path: 'Notes/Longform.md' }), true)
  assert.equal(isVaultPage({ path: 'Notes/report.pdf' }), false)
})

test('folders and non-Markdown attachments are not pages', () => {
  assert.equal(isVaultPage({ kind: 'folder', format: 'md' }), false)
  assert.equal(isVaultPage({ format: 'png' }), false)
  assert.equal(isVaultPage({ format: 'txt' }), false)
})
