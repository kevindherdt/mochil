package net.kdhmedia.mochil.audio

import net.kdhmedia.mochil.network.requireAtMost
import org.json.JSONObject

/** Metadata only: deliberately cannot be passed to APIs expecting playlist tracks. */
data class MusicPlaylistOverview(
    val name: String,
    val trackCount: Int,
    val artworkUrls: List<String>,
)

internal fun parsePlaylistSummaries(value: JSONObject): List<MusicPlaylistOverview> {
    check(value.overviewInteger("version", 1) == 1) { "Unsupported playlist overview." }
    val lists = value.getJSONArray("playlists").requireAtMost(1_000, "Music playlists")
    return List(lists.length()) { index ->
        val list = lists.getJSONObject(index)
        val name = list.getString("name")
        val count = list.overviewInteger("item_count", 10_000)
        check(name.isNotBlank() && count in 0..10_000) { "Invalid playlist overview." }
        val artwork = list.getJSONArray("artwork_urls").requireAtMost(4, "Playlist artwork")
        MusicPlaylistOverview(name, count, List(artwork.length()) { artwork.getString(it) })
    }
}

/** JSONObject.getInt coerces strings, truncates fractions, and can wrap large numbers. */
private fun JSONObject.overviewInteger(key: String, maximum: Int): Int {
    val value = get(key)
    check(value is Number) { "Invalid playlist overview." }
    val number = value.toDouble()
    check(number.isFinite() && number >= 0 && number <= maximum && number == number.toInt().toDouble()) {
        "Invalid playlist overview."
    }
    return number.toInt()
}
