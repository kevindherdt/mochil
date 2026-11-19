// A summary is deliberately not a complete track list.
export function decodePlaylistOverview(data) {
  if (Array.isArray(data)) return data.map(value => ({...value, items: Array.isArray(value?.items) ? value.items : []}))
  if (data?.version !== 1 || !Array.isArray(data.playlists)) throw new Error('Unsupported playlist overview.')
  return data.playlists.map(value => {
    if (typeof value?.name !== 'string' || !Number.isSafeInteger(value.item_count) || value.item_count < 0 || !Array.isArray(value.artwork_urls)) throw new Error('Invalid playlist overview.')
    return {name:value.name, item_count:value.item_count, artwork_urls:value.artwork_urls.filter(url=>typeof url==='string').slice(0,4), items:[], detailLoaded:false}
  })
}

export function decodePlaylistDetail(data, name) {
  if (data?.name !== name || !Array.isArray(data.items)) throw new Error('Invalid playlist detail.')
  return {...data, detailLoaded:true}
}
