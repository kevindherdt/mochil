package net.kdhmedia.mochil.wiki

import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.async
import kotlinx.coroutines.awaitAll
import kotlinx.coroutines.coroutineScope
import kotlinx.coroutines.currentCoroutineContext
import kotlinx.coroutines.ensureActive
import kotlinx.coroutines.withContext
import net.kdhmedia.mochil.auth.DeviceCredential

internal data class WikiOfflineMirrorResult(
    val saved: Int,
    val available: Int,
    val limited: Boolean,
    val assetsSaved: Int,
    val assetsAvailable: Int,
    val assetsLimited: Boolean,
    val assetUrls: Map<String, String>,
)

/** Downloads a bounded Markdown-only vault snapshot without moving network orchestration into Compose. */
internal class WikiOfflineMirror(
    private val client: WikiClient,
    private val store: WikiOfflineStore,
    private val assets: WikiOfflineAssetStore,
) {
    suspend fun save(
        credential: DeviceCredential,
        libraryId: String,
        entries: List<WikiEntry>,
        selectedPath: String,
        progress: (completed: Int, total: Int) -> Unit,
    ): WikiOfflineMirrorResult {
        val available = entries.filter { !it.isFolder && it.format.equals("md", ignoreCase = true) }
        val targets = wikiOfflineMirrorTargets(entries, selectedPath)
        val savedAt = System.currentTimeMillis()
        val pages = loadBoundedWikiPages(
            WikiOfflineSnapshot(libraryId, entries, selectedPath, emptyList(), savedAt),
            targets, MIRROR_CONCURRENCY,
            loadBatch = { batch -> coroutineScope {
                batch.map { entry -> async(Dispatchers.IO) {
                    WikiOfflinePage(entry.path, client.page(credential, libraryId, entry.id), System.currentTimeMillis())
                } }.awaitAll()
            } },
            progress = progress,
        )
        withContext(Dispatchers.IO) {
            store.save(credential.deviceId, WikiOfflineSnapshot(libraryId, entries, selectedPath, pages, savedAt))
        }
        val persisted = withContext(Dispatchers.IO) { store.load(credential.deviceId, libraryId) }
            ?: throw java.io.IOException("Could not verify the saved offline Wiki copy")
        val requestedIds = pages.mapTo(hashSetOf()) { it.page.id }
        val saved = persisted.pages.count { it.page.id in requestedIds }
        val allAssetTargets = withContext(Dispatchers.Default) {
            val context = currentCoroutineContext()
            wikiOfflineAssetTargets(persisted.entries, persisted.pages, Int.MAX_VALUE) { context.ensureActive() }
        }
        val assetTargets = allAssetTargets.take(MAX_OFFLINE_ASSETS)
        val downloadTargets = withContext(Dispatchers.IO) {
            assets.downloadTargets(credential.deviceId, libraryId, assetTargets)
        }
        val loadedAssets = loadBoundedWikiAssets(downloadTargets, MIRROR_CONCURRENCY) { batch ->
            coroutineScope {
                batch.map { entry -> async(Dispatchers.IO) { entry to loadOptionalWikiAsset { client.asset(credential, libraryId, entry) } } }.awaitAll()
            }
        }
        val assetsSaved = withContext(Dispatchers.IO) { assets.mergeAndPrune(credential.deviceId, libraryId, assetTargets, loadedAssets) }
        val assetUrls = withContext(Dispatchers.IO) { assets.cachedUrls(credential.deviceId, libraryId, entries) }
        return WikiOfflineMirrorResult(
            saved,
            available.size,
            (available.size > targets.size || saved < targets.size),
            assetsSaved,
            allAssetTargets.size,
            (allAssetTargets.size > assetTargets.size || downloadTargets.size > loadedAssets.size || assetsSaved < assetTargets.size),
            assetUrls,
        )
    }

    private companion object { const val MIRROR_CONCURRENCY = 4 }
}

internal fun buildOfflineWikiStatus(result: WikiOfflineMirrorResult): String {
    val pages = if (result.limited) "Saved ${result.saved} of ${result.available} pages offline." else "${result.saved} pages are available offline."
    if (result.assetsAvailable == 0) return pages
    val images = if (result.assetsLimited) " ${result.assetsSaved} of ${result.assetsAvailable} embedded images were saved."
    else " ${result.assetsSaved} embedded images are available offline."
    return pages + images
}

internal fun wikiOfflineAssetTargets(
    entries: List<WikiEntry>,
    pages: List<WikiOfflinePage>,
    limit: Int = MAX_OFFLINE_ASSETS,
    checkpoint: () -> Unit = {},
): List<WikiEntry> {
    checkpoint()
    if (limit <= 0) return emptyList()
    val index = WikiAssetIndex(entries)
    val targets = LinkedHashMap<String, WikiEntry>()
    pages.forEach { cached ->
        checkpoint()
        parseWikiMarkdown(cached.page.content, checkpoint).asSequence().filterIsInstance<WikiMarkdownBlock.Image>().forEach { image ->
            checkpoint()
            index.resolve(cached.path, image.target)?.let { entry -> targets.putIfAbsent(entry.id, entry) }
        }
    }
    return targets.values.take(limit.coerceAtLeast(0))
}

internal fun wikiOfflineMirrorTargets(entries: List<WikiEntry>, selectedPath: String): List<WikiEntry> {
    val pages = entries.asSequence().filter { !it.isFolder && it.format.equals("md", ignoreCase = true) }
    val selected = pages.firstOrNull { it.path == selectedPath }
    return buildList {
        if (selected != null) add(selected)
        pages.filterNot { it.id == selected?.id }.take(MAX_OFFLINE_PAGES - size).forEach(::add)
    }
}
