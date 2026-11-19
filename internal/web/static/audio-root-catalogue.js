const pageSize = kind => kind === 'music' ? 72 : 120

function totalFrom(response, itemCount) {
  const raw = response.headers.get('X-Total-Count')
  if (raw === null || raw === '') return itemCount
  const total = Number(raw)
  return Number.isFinite(total) && total >= 0 ? total : itemCount
}

export function createAudioRootCatalogue({ requestJSON }) {
  async function page({ library, kind, search = '', sort = '', played = '', cursor = '', offset = 0, limit = pageSize(kind), signal }) {
    const query = new URLSearchParams({ library, limit, sort })
    if (played) query.set('played', played)
    if (cursor) query.set('cursor', cursor)
    else if (offset > 0) query.set('offset', offset)
    if (search) query.set('q', search)

    const result = await requestJSON(`/api/audio?${query}`, { signal })
    const items = Array.isArray(result.data) ? result.data : []
    return {
      items,
      total: totalFrom(result.response, items.length),
      nextCursor: result.response.headers.get('X-Next-Cursor') || '',
    }
  }

  return { page }
}

