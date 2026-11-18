package net.kdhmedia.mochil.opds

data class OpdsLink(val relation: String, val href: String, val mediaType: String = "")

data class OpdsEntry(
    val id: String,
    val title: String,
    val authors: List<String>,
    val links: List<OpdsLink>,
) {
    val artworkLink: OpdsLink? get() = links.firstOrNull { it.relationTokens().any(COVER_RELATIONS::contains) }
    val acquisitionLink: OpdsLink? get() = links.firstOrNull { it.relationTokens().any { relation -> relation.startsWith(ACQUISITION_RELATION) } }
    val navigationLink: OpdsLink? get() = links.firstOrNull { link ->
        link.relationTokens().any { it in NAVIGATION_RELATIONS } || link.mediaType.startsWith("application/atom+xml")
    }
}

data class OpdsFeed(
    val id: String,
    val title: String,
    val entries: List<OpdsEntry>,
    val navigation: List<OpdsLink>,
) {
    val nextLink get() = navigation.firstOrNull { "next" in it.relationTokens() }
    val searchLink get() = navigation.firstOrNull { "search" in it.relationTokens() }
}

internal fun OpdsLink.relationTokens(): Set<String> = relation.lowercase().split(Regex("\\s+")).filter(String::isNotBlank).toSet()

internal const val ACQUISITION_RELATION = "http://opds-spec.org/acquisition"
private val COVER_RELATIONS = setOf(
    "http://opds-spec.org/image",
    "http://opds-spec.org/image/thumbnail",
    "x-stanza-cover-image",
    "x-stanza-cover-image-thumbnail",
)
private val NAVIGATION_RELATIONS = setOf("subsection", "collection", "http://opds-spec.org/shelf")
