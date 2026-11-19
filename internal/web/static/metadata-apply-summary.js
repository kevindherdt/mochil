function plural(count, singular, multiple = `${singular}s`) {
  return `${Number(count).toLocaleString()} ${Number(count) === 1 ? singular : multiple}`
}

export function metadataApplyReceipt(result, preview) {
  const parts = []
  const applyMode = String(result.apply_mode || '').toLowerCase()
  const sourceType = String(preview?.source_type || '').toLowerCase()
  const calibre = applyMode === 'calibre-database' || sourceType === 'calibre'
  if (result.source_books) parts.push(calibre
    ? `${plural(result.source_books, 'Calibre record')} updated`
    : `${plural(result.source_books, 'source file')} updated`)
  if (result.override_books) parts.push(`${plural(result.override_books, 'catalogue override')} saved`)

  const overrideFormats = new Set((preview?.changes || [])
    .filter(change => change.mode === 'catalogue override')
    .map(change => String(change.format || '').toLowerCase()))

  if (calibre && result.source_books) {
    parts.push('Calibre database updated; publication archives unchanged')
  } else if (result.override_books) {
    if (overrideFormats.has('cbz') || overrideFormats.has('cbr')) {
      parts.push('Comic archives unchanged; ComicInfo.xml was not rewritten')
    } else {
      parts.push('Original publication files unchanged')
    }
  } else if (result.source_books) {
    parts.push('Verified backup copies retained')
  }

  if (result.scan_warning) parts.push(`Catalogue refresh needs attention: ${result.scan_warning}`)
  else if (result.scan_books) parts.push(`${plural(result.scan_books, 'catalogue item')} verified after refresh`)

  return parts.join(' · ') || `${plural(result.applied_books || 0, 'publication')} updated`
}

export function metadataApplyImpact(preview) {
  const changes = preview?.changes || []
  const overrideFormats = new Set(changes
    .filter(change => change.mode === 'catalogue override')
    .map(change => String(change.format || '').toLowerCase()))
  if (overrideFormats.has('cbz') || overrideFormats.has('cbr')) {
    return 'Comic changes are reversible catalogue overrides. CBZ/CBR archives and ComicInfo.xml remain unchanged.'
  }
  if (String(preview?.source_type || '').toLowerCase() === 'calibre') {
    return 'Changes update the Calibre database after verification; publication archives remain unchanged.'
  }
  if (changes.some(change => change.mode === 'catalogue override')) {
    return 'Catalogue overrides change how Mochil and OPDS display these publications; original files remain unchanged.'
  }
  if (changes.some(change => change.mode === 'source file')) {
    return 'Validated EPUB source files are updated with rollback copies retained.'
  }
  return ''
}
