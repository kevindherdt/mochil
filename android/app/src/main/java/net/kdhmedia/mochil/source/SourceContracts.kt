package net.kdhmedia.mochil.source

data class SourceLibrary(val id: String, val name: String, val kind: String)
data class SourceItem(
    val id: String,
    val libraryId: String,
    val title: String,
    val artworkUrl: String?,
    val authors: List<String> = emptyList(),
    val mediaType: String = "",
    val acquisitionUrl: String? = null,
    val navigationUrl: String? = null,
)

interface CatalogueSource {
    val id: String
    suspend fun libraries(): List<SourceLibrary>
    suspend fun latest(libraryId: String, cursor: String?): List<SourceItem>
    suspend fun search(libraryId: String, query: String, cursor: String?): List<SourceItem>
}
