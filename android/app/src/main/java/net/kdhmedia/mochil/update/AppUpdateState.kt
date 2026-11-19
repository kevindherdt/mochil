package net.kdhmedia.mochil.update

import org.json.JSONObject

data class AppUpdateState(
    val checking: Boolean = false,
    val downloading: Boolean = false,
    val availableVersion: String? = null,
    val downloadPath: String? = null,
    val message: String? = null,
    val releaseNotes: ReleaseNotes? = null,
)

data class ReleaseNotes(
    val newFeatures: List<String> = emptyList(),
    val fixes: List<String> = emptyList(),
    val performance: List<String> = emptyList(),
    val changes: List<String> = emptyList(),
)

private fun org.json.JSONArray?.strings(): List<String> =
    if (this == null) emptyList() else List(length()) { optString(it).take(240) }.filter(String::isNotBlank)

internal fun JSONObject?.releaseFixes(): List<String> =
    (this?.optJSONArray("fixes") ?: this?.optJSONArray("fixed")).strings()

internal fun JSONObject?.releaseNotes(): ReleaseNotes = ReleaseNotes(
    newFeatures = this?.optJSONArray("new").strings(),
    fixes = releaseFixes(),
    performance = this?.optJSONArray("performance").strings(),
    changes = this?.optJSONArray("changes").strings(),
)
