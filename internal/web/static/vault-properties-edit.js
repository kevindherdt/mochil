// Editing a note's properties as a form. Only the leading "---" block is rewritten: the body is passed through untouched,
// and a block this form cannot read back safely (anything but simple key/value and lists) is never offered for editing.
import { splitVaultFrontmatter, parseProperties } from './vault-frontmatter.js?v=2'

const needsQuotes = value => value === '' || /^[\s\-?:,\[\]{}#&*!|>'"%@`]|: | #|\s$/.test(value)
const scalar = value => (needsQuotes(value) ? `"${value.replace(/\\/g, '\\\\').replace(/"/g, '\\"')}"` : value)

export function propertiesToYAML(rows) {
  return rows.filter(row => row.key.trim()).map(row => {
    const values = row.values.map(value => value.trim()).filter(Boolean)
    if (!row.list) return `${row.key.trim()}: ${scalar(values[0] ?? '')}\n`
    return values.length ? `${row.key.trim()}:\n${values.map(value => `  - ${scalar(value)}\n`).join('')}` : `${row.key.trim()}: []\n`
  }).join('')
}

/** The same note with its properties replaced; no rows removes the block. The body is never touched. */
export function withProperties(content, rows) {
  const { properties, body } = splitVaultFrontmatter(content)
  const yaml = propertiesToYAML(rows)
  const rest = properties === null ? content : body
  return yaml ? `---\n${yaml}---\n${rest}` : rest
}

export const editableRows = content => {
  const { properties } = splitVaultFrontmatter(content)
  return properties === null ? null : parseProperties(properties)
}

/** Rows from the form: a list is typed as comma-separated values. */
export function rowsFromForm(form) {
  return [...form.querySelectorAll('[data-prop-row]')].map(row => {
    const list = row.dataset.list === 'true'
    const value = row.querySelector('[data-prop-value]').value
    return { key: row.querySelector('[data-prop-key]').value, list, values: list ? value.split(',') : [value] }
  })
}

const rowHTML = (row, escapeHTML) => `<div class="prop-row" data-prop-row data-list="${row.list}"><input data-prop-key value="${escapeHTML(row.key)}" aria-label="Property name"><input data-prop-value value="${escapeHTML(row.values.join(', '))}" aria-label="Value${row.list ? 's, separated by commas' : ''}" placeholder="${row.list ? 'a, b, c' : ''}"><button type="button" data-prop-remove aria-label="Remove ${escapeHTML(row.key)}">×</button></div>`

/** Opens the form. `save(newContent)` does the writing and returns when it is done; the dialog shows its error. */
export function openPropertiesDialog({ document, rows, escapeHTML, save }) {
  const dialog = document.createElement('dialog')
  dialog.className = 'vault-prop-dialog'
  dialog.innerHTML = `<form method="dialog"><h2>Properties</h2><div class="prop-rows">${rows.map(row => rowHTML(row, escapeHTML)).join('')}</div><p class="prop-actions"><button type="button" data-prop-add>+ Add property</button><label><input type="checkbox" data-prop-list> as list</label></p><p class="prop-error" role="alert"></p><footer><button type="button" data-prop-cancel>Cancel</button><button type="submit" class="primary">Save</button></footer></form>`
  const form = dialog.querySelector('form')
  dialog.addEventListener('click', event => {
    if (event.target.closest('[data-prop-remove]')) event.target.closest('[data-prop-row]').remove()
    else if (event.target.closest('[data-prop-cancel]')) dialog.close()
    else if (event.target.closest('[data-prop-add]')) {
      const list = form.querySelector('[data-prop-list]').checked
      form.querySelector('.prop-rows').insertAdjacentHTML('beforeend', rowHTML({ key: '', list, values: [] }, escapeHTML))
      form.querySelector('.prop-rows [data-prop-row]:last-child [data-prop-key]').focus()
    }
  })
  form.addEventListener('submit', async event => {
    event.preventDefault()
    try { await save(rowsFromForm(form)); dialog.close() } catch (error) { form.querySelector('.prop-error').textContent = error.message || 'Could not save' }
  })
  dialog.addEventListener('close', () => dialog.remove())
  document.body.append(dialog)
  dialog.showModal()
  return dialog
}
