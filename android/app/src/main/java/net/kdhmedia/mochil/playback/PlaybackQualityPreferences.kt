package net.kdhmedia.mochil.playback

import android.content.Context
import android.net.ConnectivityManager
import android.net.NetworkCapabilities

enum class PlaybackQuality(val label: String) {
    Maximum("Maximum"), High("High"), Medium("Medium"), Low("Low");

    val wireValue: String get() = name.lowercase()
}

data class PlaybackQualityPolicy(
    val wifi: PlaybackQuality = PlaybackQuality.Maximum,
    val mobile: PlaybackQuality = PlaybackQuality.High,
)

class PlaybackQualityPreferences(context: Context) {
    private val appContext = context.applicationContext
    private val values = appContext.getSharedPreferences("playback_quality", Context.MODE_PRIVATE)

    fun load() = PlaybackQualityPolicy(read("wifi", PlaybackQuality.Maximum), read("mobile", PlaybackQuality.High))

    fun save(policy: PlaybackQualityPolicy) {
        values.edit().putString("wifi", policy.wifi.name).putString("mobile", policy.mobile.name).apply()
    }

    fun effective(): PlaybackQuality {
        val policy = load()
        val connectivity = appContext.getSystemService(ConnectivityManager::class.java) ?: return policy.wifi
        val network = connectivity.activeNetwork ?: return policy.wifi
        val capabilities = connectivity.getNetworkCapabilities(network) ?: return policy.wifi
        return if (capabilities.hasCapability(NetworkCapabilities.NET_CAPABILITY_NOT_METERED)) policy.wifi else policy.mobile
    }

    private fun read(key: String, fallback: PlaybackQuality) =
        runCatching { PlaybackQuality.valueOf(values.getString(key, fallback.name) ?: fallback.name) }.getOrDefault(fallback)
}
