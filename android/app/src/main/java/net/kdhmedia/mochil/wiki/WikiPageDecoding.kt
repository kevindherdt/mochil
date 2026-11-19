package net.kdhmedia.mochil.wiki

import org.json.JSONObject

/** A malformed page must never become an editable empty document. */
internal fun decodeWikiPage(value: JSONObject, expectedId: String): WikiPage {
    val id = value.get("id")
    val content = value.get("content")
    check(id is String && id.isNotBlank() && id == expectedId && content is String) { "Invalid Wiki page response." }
    if (content.length > MAX_WIKI_DRAFT_CHARS) throw WikiException(0, "This Wiki page is too large to open safely.")
    val writable = value.opt("writable")
    check(writable == null || writable is Boolean) { "Invalid Wiki page permissions." }
    val revision = value.opt("edit_revision")
    check(revision == null || revision is String) { "Invalid Wiki page revision." }
    check(writable != true || (revision is String && revision.isNotBlank())) { "Missing Wiki edit revision." }
    return WikiPage(id, value.optString("title", "Untitled"), content, revision ?: "", writable == true)
}
