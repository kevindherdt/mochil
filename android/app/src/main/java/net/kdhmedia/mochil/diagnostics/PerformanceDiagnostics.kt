package net.kdhmedia.mochil.diagnostics

import android.content.Context
import android.os.Build
import java.net.SocketTimeoutException
import java.net.UnknownHostException
import net.kdhmedia.mochil.BuildConfig
import org.json.JSONArray
import org.json.JSONObject

data class DiagnosticBatch(val body: String, val eventCount: Int)

class PerformanceDiagnosticsStore(context: Context) {
    private val preferences = context.getSharedPreferences(PREFERENCES, Context.MODE_PRIVATE)

    fun enabled() = preferences.getBoolean(ENABLED, false)
    fun setEnabled(enabled: Boolean) = preferences.edit().putBoolean(ENABLED, enabled).commit()
    fun pendingCount(): Int = synchronized(lock) { events().length() }

    fun record(kind: String, name: String, durationMs: Long, outcome: String, detail: String? = null) {
        if (!enabled() || kind !in allowedKinds || !(name in allowedNames || playbackDetailName.matches(name)) || outcome !in allowedOutcomes) return
        synchronized(lock) {
            val values = events()
            values.put(JSONObject().apply {
                put("at", System.currentTimeMillis())
                put("kind", kind)
                put("name", name)
                if (durationMs > 0) put("duration_ms", durationMs.coerceAtMost(MAX_DURATION_MS))
                put("outcome", outcome)
            })
            while (values.length() > MAX_EVENTS) values.remove(0)
            while (values.toString().toByteArray().size > MAX_EVENT_BYTES && values.length() > 1) values.remove(0)
            persist(values)
        }
    }

    /** The oldest [maxEvents] saved events as one upload, so a large backlog goes up in several small requests. */
    fun batch(maxEvents: Int = MAX_BATCH_EVENTS): DiagnosticBatch? = synchronized(lock) {
        val values = events()
        if (values.length() == 0) return null
        val ordered = JSONArray().apply {
            (0 until values.length()).mapNotNull(values::optJSONObject).sortedBy { it.optLong("at") }.take(maxEvents).forEach(::put)
        }
        DiagnosticBatch(JSONObject().apply {
            put("app_version", BuildConfig.VERSION_NAME)
            put("android", Build.VERSION.RELEASE.orEmpty().take(32))
            put("device", "${Build.MANUFACTURER} ${Build.MODEL}".trim().take(120))
            put("events", ordered)
        }.toString(), ordered.length())
    }

    fun acknowledge(count: Int) = synchronized(lock) {
        val values = events()
        repeat(count.coerceIn(0, values.length())) { values.remove(0) }
        persist(values)
    }

    fun clear() = synchronized(lock) {
        cachedEvents = JSONArray()
        preferences.edit().remove(EVENTS).apply()
    }

    private fun events(): JSONArray {
        cachedEvents?.let { return it }
        return runCatching {
            val stored = JSONArray(preferences.getString(EVENTS, "[]"))
            JSONArray().apply {
                (0 until minOf(stored.length(), MAX_EVENTS)).mapNotNull(stored::optJSONObject).filter(::validEvent).forEach(::put)
            }
        }.getOrDefault(JSONArray()).also { cachedEvents = it }
    }

    private fun persist(values: JSONArray) {
        cachedEvents = values
        preferences.edit().putString(EVENTS, values.toString()).apply()
    }

    private fun validEvent(value: JSONObject): Boolean =
        value.optLong("at") > 0 && value.optString("kind") in allowedKinds && value.optString("name") in allowedNames &&
            value.optString("outcome") in allowedOutcomes && value.optLong("duration_ms") in 0..MAX_DURATION_MS &&
            value.optString("detail").isEmpty()

    private companion object {
        const val PREFERENCES = "performance_diagnostics"
        const val ENABLED = "enabled"
        const val EVENTS = "events"
        const val MAX_EVENTS = 200
        const val MAX_BATCH_EVENTS = 50
        const val MAX_EVENT_BYTES = 56 * 1024
        const val MAX_DURATION_MS = 3_600_000L
        val lock = Any()
        @Volatile var cachedEvents: JSONArray? = null
        val allowedKinds = setOf("lifecycle", "network", "playback", "reader", "storage", "tv_ui")
        val allowedNames = setOf(
            "app_foreground", "foreground_session", "mobile_first_frame", "tv_app_first_frame", "load_libraries", "load_books", "load_series", "load_authors", "load_author_books", "load_continue_reading", "load_dashboard_home", "load_dashboard_reading",
            "load_dashboard_listening", "load_dashboard_watching", "load_audio_groups", "load_audio_tracks", "load_audio_favorites",
            "load_discovery_albums", "load_discovery_artists", "load_discovery_recent", "load_discovery_played", "load_recently_played", "update_audio_favorite",
            "load_podcast_shows", "load_podcast_episodes", "reconcile_podcast_downloads",
            "podcast_downloads_device_match", "podcast_downloads_server_match",
            "podcast_downloads_unmatched", "podcast_downloads_empty",
            "load_webdav_folder", "download_webdav_media", "stream_webdav_media", "index_webdav_library",
            "video_codec_error", "video_media_error", "video_network_error", "video_server_error", "video_storage_error",
            "webdav_auth_error", "webdav_not_found_error", "webdav_endpoint_error", "webdav_redirect_error", "webdav_server_error", "webdav_range_error", "webdav_reference_error", "webdav_response_error", "webdav_transport_error",
            "playback_error", "audio_decoder_error", "audio_network_error", "audio_source_error", "audio_parsing_error", "audio_track_error", "audio_unspecified_error", "audio_playback_error", "open_epub", "open_comic",
            "video_ready", "video_fallback", "video_buffering_timeout", "video_reconnect_retry", "video_buffering_reconnect",
            "highlight_action_selected", "highlight_locator_resolved", "highlight_state_saved",
            "highlight_decorations_applied", "highlight_selection_cleared", "highlight_action_mode_finished",
            "load_wiki_tree", "load_wiki_page", "save_wiki_page", "create_wiki_entry", "move_wiki_page", "trash_wiki_page",
            "format_wiki_page", "format_wiki_tree",
            "load_wiki_cache", "save_wiki_cache",
            "home_first_frame", "library_first_frame", "nowplaying_first_frame", "settings_first_frame",
        )
        // Playback failures name their Media3 error code, exception type and HTTP status so a stuck stream can be told apart.
        val playbackDetailName = Regex("audio_(?:code_[0-9]{1,5}|http_[0-9]{3}|cause_[a-z0-9_]{1,40})")
        val allowedOutcomes = setOf("ok", "error", "cancelled", "offline", "timeout")
    }
}

object PerformanceDiagnostics {
    @Volatile private var store: PerformanceDiagnosticsStore? = null

    fun initialize(context: Context) {
        if (store == null) synchronized(this) {
            if (store == null) store = PerformanceDiagnosticsStore(context.applicationContext)
        }
    }

    fun record(kind: String, name: String, durationMs: Long = 0, outcome: String = "ok", detail: String? = null) {
        store?.record(kind, name, durationMs, outcome, detail)
    }

    fun <T> measure(kind: String, name: String, block: () -> T): T {
        val started = System.nanoTime()
        return try {
            block().also { record(kind, name, elapsedMilliseconds(started), "ok") }
        } catch (error: Throwable) {
            val outcome = when (error) {
                is SocketTimeoutException -> "timeout"
                is UnknownHostException -> "offline"
                is kotlinx.coroutines.CancellationException -> "cancelled"
                else -> "error"
            }
            record(kind, name, elapsedMilliseconds(started), outcome)
            throw error
        }
    }

    private fun elapsedMilliseconds(started: Long) = ((System.nanoTime() - started) / 1_000_000).coerceAtLeast(0)
}
