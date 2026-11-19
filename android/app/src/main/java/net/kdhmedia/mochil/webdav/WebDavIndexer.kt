package net.kdhmedia.mochil.webdav

internal class WebDavIndexer(private val pageCache: WebDavPageCache) {
    fun scan(connection: WebDavConnection, onProgress: (folders: Int, media: Int) -> Unit = { _, _ -> }): WebDavIndex {
        val pending = ArrayDeque<String>().apply { add(connection.rootUrl) }
        val visited = mutableSetOf<String>()
        val media = linkedMapOf<String, WebDavEntry>()
        val client = WebDavClient(connection)
        while (pending.isNotEmpty() && visited.size < MAX_INDEX_FOLDERS && media.size < MAX_INDEX_ENTRIES) {
            val url = pending.removeFirst()
            if (!visited.add(url)) continue
            val cached = pageCache.get(connection, url)
            val fetched = client.list(url, cached?.validator.orEmpty())
            val page = resolveWebDavPage(cached, fetched)
            if (!fetched.notModified) pageCache.put(connection, page)
            page.entries.forEach { entry ->
                if (entry.collection) {
                    if (entry.href !in visited && pending.size + visited.size < MAX_INDEX_FOLDERS) pending.addLast(entry.href)
                } else if (media.size < MAX_INDEX_ENTRIES) media[entry.href] = entry
            }
            if (visited.size == 1 || visited.size % PROGRESS_INTERVAL == 0) onProgress(visited.size, media.size)
        }
        val complete = pending.isEmpty()
        onProgress(visited.size, media.size)
        return WebDavIndex(connection.id, webDavCredentialKey(connection), media.values.sortedBy { it.name.lowercase() }, System.currentTimeMillis(), complete)
    }

    private companion object { const val MAX_INDEX_FOLDERS = 2_000; const val PROGRESS_INTERVAL = 10 }
}
