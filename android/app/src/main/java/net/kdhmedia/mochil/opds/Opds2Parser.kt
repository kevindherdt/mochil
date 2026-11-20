package net.kdhmedia.mochil.opds

import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.jsonObject

object Opds2Parser {
    fun parse(bytes: ByteArray): OpdsFeed {
        require(bytes.size in 1..MAX_FEED_BYTES) { "The OPDS 2 response is empty or too large." }
        val root = Json.parseToJsonElement(bytes.decodeToString()).jsonObject
        val metadata = root.objectValue("metadata")
        val entries = buildList {
            addPublications(root.arrayValue("publications"))
            addNavigation(root.arrayValue("navigation"))
            root.arrayValue("groups").take(MAX_GROUPS).forEach { groupValue ->
                val group = groupValue as? JsonObject ?: return@forEach
                addPublications(group.arrayValue("publications"))
                addNavigation(group.arrayValue("navigation"))
            }
        }.take(MAX_ENTRIES)
        return OpdsFeed(
            id = metadata.string("identifier").ifBlank { metadata.string("title") }.take(MAX_TEXT),
            title = metadata.string("title").ifBlank { "OPDS catalogue" }.take(MAX_TEXT),
            entries = entries,
            navigation = parseLinks(root.arrayValue("links")),
        )
    }

    private fun MutableList<OpdsEntry>.addPublications(values: JsonArray) {
        values.take(MAX_ENTRIES - size).forEach { value ->
            val publication = value as? JsonObject ?: return@forEach
            val metadata = publication.objectValue("metadata")
            val title = metadata.string("title").ifBlank { "Untitled" }.take(MAX_TEXT)
            val links = parseLinks(publication.arrayValue("links")).toMutableList()
            publication.arrayValue("images").take(MAX_IMAGES).mapNotNullTo(links) { image ->
                parseLink(image as? JsonObject ?: return@mapNotNullTo null, defaultRelation = "http://opds-spec.org/image")
            }
            add(OpdsEntry(metadata.string("identifier").ifBlank { title }.take(MAX_TEXT), title, parseAuthors(metadata["author"]), links.take(MAX_LINKS)))
        }
    }

    private fun MutableList<OpdsEntry>.addNavigation(values: JsonArray) {
        values.take(MAX_ENTRIES - size).forEach { value ->
            val navigation = value as? JsonObject ?: return@forEach
            val title = navigation.string("title").ifBlank { navigation.objectValue("metadata").string("title") }.ifBlank { "Untitled shelf" }.take(MAX_TEXT)
            val link = parseLinks(navigation.arrayValue("links")).firstOrNull() ?: parseLink(navigation, defaultRelation = "subsection") ?: return@forEach
            add(OpdsEntry(navigation.string("identifier").ifBlank { link.href }.take(MAX_TEXT), title, emptyList(), listOf(link.copy(relation = "subsection"))))
        }
    }

    private fun parseLinks(values: JsonArray): List<OpdsLink> = values.take(MAX_LINKS).mapNotNull { parseLink(it as? JsonObject ?: return@mapNotNull null) }

    private fun parseLink(value: JsonObject, defaultRelation: String = ""): OpdsLink? {
        val href = value.string("href").take(MAX_URL)
        if (href.isBlank()) return null
        val relation = when (val rel = value["rel"]) {
            is JsonArray -> rel.mapNotNull { (it as? JsonPrimitive)?.contentOrNull }.joinToString(" ")
            is JsonPrimitive -> rel.contentOrNull.orEmpty()
            else -> ""
        }.ifBlank { defaultRelation }.take(MAX_RELATION)
        val type = value.string("type").take(MAX_MEDIA_TYPE)
        val acquisitionRelation = if (relation.isBlank() && type.substringBefore(';') in ACQUISITION_TYPES) ACQUISITION_RELATION else relation
        return OpdsLink(acquisitionRelation, href, type)
    }

    private fun parseAuthors(value: JsonElement?): List<String> = when (value) {
        is JsonArray -> value.mapNotNull(::authorName)
        null -> emptyList()
        else -> listOfNotNull(authorName(value))
    }.filter(String::isNotBlank).take(MAX_AUTHORS)

    private fun authorName(value: JsonElement): String? = when (value) {
        is JsonPrimitive -> value.contentOrNull
        is JsonObject -> value.string("name")
        else -> null
    }?.trim()?.take(MAX_TEXT)

    private fun JsonObject.objectValue(name: String) = this[name] as? JsonObject ?: JsonObject(emptyMap())
    private fun JsonObject.arrayValue(name: String) = this[name] as? JsonArray ?: JsonArray(emptyList())
    private fun JsonObject.string(name: String) = (this[name] as? JsonPrimitive)?.contentOrNull.orEmpty().trim()

    const val MAX_FEED_BYTES = 4 * 1024 * 1024
    private const val MAX_ENTRIES = 500
    private const val MAX_GROUPS = 50
    private const val MAX_IMAGES = 10
    private const val MAX_LINKS = 100
    private const val MAX_AUTHORS = 20
    private const val MAX_TEXT = 500
    private const val MAX_URL = 4096
    private const val MAX_RELATION = 500
    private const val MAX_MEDIA_TYPE = 160
    private val ACQUISITION_TYPES = setOf("application/epub+zip", "application/pdf", "application/vnd.comicbook+zip", "application/x-cbz")
}
