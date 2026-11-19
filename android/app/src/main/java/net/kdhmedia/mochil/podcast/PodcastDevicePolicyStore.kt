package net.kdhmedia.mochil.podcast

import android.content.Context
import org.json.JSONObject

data class PodcastDevicePolicy(val autoDownload: Boolean = false, val retainDownloads: Int = 3, val unmeteredOnly: Boolean = true)
data class PodcastDeviceDownloadPlan(val download: Set<String>, val remove: Set<String>)

internal fun podcastDeviceDownloadPlan(newest: List<String>, offline: Set<String>, retain: Int): PodcastDeviceDownloadPlan {
    val wanted = newest.distinct().take(retain.coerceIn(1, 10)).toSet()
    return PodcastDeviceDownloadPlan(download = wanted - offline, remove = offline - wanted)
}

class PodcastDevicePolicyStore(context: Context) {
    private val preferences = context.getSharedPreferences("podcast_device_policies", Context.MODE_PRIVATE)

    fun get(feedId: String): PodcastDevicePolicy = runCatching {
        JSONObject(preferences.getString(feedId, "{}") ?: "{}").run {
            PodcastDevicePolicy(optBoolean("auto_download"), optInt("retain_downloads", 3).coerceIn(1, 10), optBoolean("unmetered_only", true))
        }
    }.getOrDefault(PodcastDevicePolicy())

    fun put(feedId: String, policy: PodcastDevicePolicy) {
        require(feedId.isNotBlank() && policy.retainDownloads in 1..10)
        val value = JSONObject().put("auto_download", policy.autoDownload).put("retain_downloads", policy.retainDownloads).put("unmetered_only", policy.unmeteredOnly)
        preferences.edit().putString(feedId, value.toString()).apply()
    }
}
