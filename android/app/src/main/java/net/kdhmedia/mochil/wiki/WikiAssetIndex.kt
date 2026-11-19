package net.kdhmedia.mochil.wiki

/** Image embeds retain their narrow format policy independently of attachment opening. */
internal class WikiAssetIndex(entries: List<WikiEntry>) {
    private val files = WikiFileIndex(entries, WIKI_IMAGE_FORMATS)
    fun resolve(pagePath: String, target: String): WikiEntry? = files.resolve(pagePath, target)
}
