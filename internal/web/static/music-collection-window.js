/** Owns incremental rendering state for finite, in-memory music collections. */
export function createMusicCollectionWindow({ pageSize = 120, onMore }) {
  let limit = pageSize
  let hasMore = false

  const reset = () => { limit = pageSize; hasMore = false }
  const apply = view => { hasMore = Boolean(view?.hasMore); return view }
  const more = () => {
    if (!hasMore) return false
    limit += pageSize
    onMore()
    return true
  }

  return {
    reset,
    apply,
    more,
    get limit() { return limit },
    get hasMore() { return hasMore },
  }
}
