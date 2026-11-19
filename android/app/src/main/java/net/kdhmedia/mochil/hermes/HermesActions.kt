package net.kdhmedia.mochil.hermes

import net.kdhmedia.mochil.catalogue.ReadingStatus
import org.json.JSONArray
import org.json.JSONObject

// Quick actions, the same as on the website (hermes-actions.js): one tap sends a question together with what Mochil
// knows (new books, what you are reading, one book's details). The context is plain text added to the message inside a
// block the chat folds away, so you can always see exactly what she was given. Nothing else leaves the server.

data class HermesAction(val id: String, val label: String, val hint: String, val prompt: String = "", val draft: String = "", val pick: Boolean = false)

val HERMES_ACTIONS = listOf(
    HermesAction("recent", "What's new in my library", "Recently added books", prompt = "What's new in my library? Tell me what stands out among the recently added books."),
    HermesAction("next", "What should I read next", "Uses what you are reading and finished", prompt = "What should I read next? Suggest a few books from my library and say why each suits me."),
    HermesAction("book", "Tell me about a book", "Pick a book to summarise", prompt = "Tell me about this book and what it is about, without spoiling more than the description does.", pick = true),
    HermesAction("polish", "Polish my text", "Paste your text after the request", draft = "Polish this text. Keep my meaning and my voice, fix mistakes, and make it read better:\n\n"),
)

private val ENTITIES = mapOf("&amp;" to "&", "&lt;" to "<", "&gt;" to ">", "&quot;" to "\"", "&#39;" to "'", "&nbsp;" to " ", "&rsquo;" to "’", "&lsquo;" to "‘")

/** A description without markup, cut to a length worth reading. */
fun plainText(html: String, limit: Int = 700): String {
    val text = html.replace(Regex("<[^>]*>"), " ").replace(Regex("&[#a-zA-Z0-9]+;")) { ENTITIES[it.value] ?: " " }.replace(Regex("\\s+"), " ").trim()
    return if (text.length > limit) text.take(limit).replace(Regex("\\s+\\S*$"), "") + "…" else text
}

private fun JSONObject.strings(name: String): List<String> = optJSONArray(name)?.let { a -> (0 until a.length()).map { a.optString(it) } }.orEmpty().filter { it.isNotBlank() }

/** One book as one line: title, author, series, and how far you are. */
fun bookLine(book: JSONObject): String {
    val authors = book.strings("authors").joinToString(", ")
    val index = book.optDouble("series_index", Double.NaN)
    val series = book.optString("series").takeIf { it.isNotBlank() }?.let { name ->
        val number = if (!index.isNaN() && index > 0) " #" + (if (index % 1.0 == 0.0) index.toInt().toString() else index.toString()) else ""
        " ($name$number)"
    }.orEmpty()
    val progress = book.optJSONObject("reading_progress")
    val fraction = progress?.optDouble("fraction", 0.0) ?: book.optDouble("fraction", 0.0)
    val state = when {
        progress?.optBoolean("finished") == true -> ", finished"
        fraction > 0 -> ", ${Math.round(fraction * 100)}% read"
        else -> ""
    }
    return "- ${book.optString("title", "Untitled")}${if (authors.isNotEmpty()) " by $authors" else ""}$series$state"
}

private fun section(title: String, books: List<JSONObject>, max: Int) =
    if (books.isEmpty()) "" else "$title:\n" + books.take(max).joinToString("\n", transform = ::bookLine)

/** The text added to a message, in a block the chat knows how to fold away again. */
fun wrapContext(prompt: String, context: String) = if (context.isEmpty()) prompt else "$prompt\n\n<library-context>\n$context\n</library-context>"

data class SplitContext(val prompt: String, val context: String)

/** Splits a message back into what you asked and the context that came with it. */
fun splitContext(text: String): SplitContext {
    val match = Regex("^([\\s\\S]*?)\\s*<library-context>\\n([\\s\\S]*?)\\n</library-context>\\s*$").find(text)
    return if (match != null) SplitContext(match.groupValues[1], match.groupValues[2]) else SplitContext(text, "")
}

/** Gathers the context for an action from the library. [api] fetches a JSON list from a path such as "books?limit=15". */
suspend fun gatherContext(action: HermesAction, api: suspend (String) -> List<JSONObject>, book: JSONObject? = null): String {
    suspend fun safe(path: String) = runCatching { api(path) }.getOrDefault(emptyList())
    return when {
        action.id == "recent" -> section("Recently added books", api("books?sort=recent&limit=15"), 15)
        action.id == "next" -> listOf(
            section("Reading now", safe(ReadingStatus.Reading.wire), 8),
            section("Recently finished", safe("reading?finished=true&limit=12"), 12),
            section("Recently added", safe("books?sort=recent&limit=12"), 12),
        ).filter { it.isNotEmpty() }.joinToString("\n\n")
        action.id == "book" && book != null -> {
            val detail = plainText(book.optString("description"))
            val language = book.optString("language").takeIf { it.isNotBlank() }?.let { ", language: $it" }.orEmpty()
            "${bookLine(book)}\nLibrary: ${book.optString("library_name")}$language" + if (detail.isNotEmpty()) "\nDescription: $detail" else "\n(No description is saved for this book.)"
        }
        else -> ""
    }
}

internal fun jsonObjects(array: JSONArray): List<JSONObject> = (0 until array.length()).mapNotNull { array.optJSONObject(it) }
