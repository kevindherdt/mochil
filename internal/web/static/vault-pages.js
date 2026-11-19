/** Page navigation is distinct from the complete attachment catalogue. */
export function isVaultPage(entry) {
  if (entry.kind === 'folder') return false
  const format = String(entry.format || String(entry.path || '').split('.').at(-1)).toLowerCase()
  return format === 'md' || format === 'markdown'
}

export function vaultPageEntries(entries) {
  return entries.filter(entry => entry.kind === 'folder' || isVaultPage(entry))
}
