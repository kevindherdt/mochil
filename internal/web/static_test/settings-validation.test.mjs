import test from 'node:test'
import assert from 'node:assert/strict'
import { defaultPreferredFormats, firstInvalidGlobalSetting } from '../static/settings.js'

const input = (panel, valid, disabled = false) => ({
	disabled,
	checkValidity: () => valid,
	closest: () => ({ dataset: { settingsPanel: panel } }),
})

test('account action forms never block the global settings save', () => {
	const accountPassword = input('accounts', false)
	const sourceURL = input('sources', true)
	assert.equal(firstInvalidGlobalSetting([accountPassword, sourceURL]), undefined)
})

test('global settings still report the first invalid enabled field', () => {
	const disabledInvalid = input('storage', false, true)
	const invalidSource = input('sources', false)
	assert.equal(firstInvalidGlobalSetting([disabledInvalid, invalidSource]), invalidSource)
})

test('a field inside its own form, like making a new voice, never blocks Save changes', () => {
	const voiceName = { ...input('modules', false), form: { id: 'nv-form' } }
	const libraryPath = input('libraries', false)
	assert.equal(firstInvalidGlobalSetting([voiceName]), undefined)
	assert.equal(firstInvalidGlobalSetting([voiceName, libraryPath]), libraryPath, 'a real library problem is still reported')
})

test('manga libraries prefer CBZ while ordinary books prefer EPUB', () => {
	assert.deepEqual(defaultPreferredFormats('manga'), ['cbz', 'pdf', 'epub'])
	assert.deepEqual(defaultPreferredFormats('books'), ['epub', 'pdf', 'mobi', 'azw3'])
})
