package net.kdhmedia.mochil.dashboard

import java.net.HttpURLConnection
import java.net.URI
import net.kdhmedia.mochil.audio.AudioTrack
import net.kdhmedia.mochil.audio.optionalFiniteDouble
import net.kdhmedia.mochil.auth.DeviceCredential
import net.kdhmedia.mochil.catalogue.BookSummary
import net.kdhmedia.mochil.diagnostics.PerformanceDiagnostics
import net.kdhmedia.mochil.network.readBoundedResponseText
import net.kdhmedia.mochil.network.requireAtMost
import net.kdhmedia.mochil.network.openAndroidApiGet
import net.kdhmedia.mochil.network.reportIfExpiredDeviceSession
import org.json.JSONArray
import org.json.JSONObject
import net.kdhmedia.mochil.catalogue.toBookSummary
import net.kdhmedia.mochil.video.VideoClient

class DashboardClient {
    fun home(credential: DeviceCredential): DashboardSnapshot? {
        val connection = openAndroidApiGet(credential, "/api/android/v1/home")
        return try {
            if (connection.responseCode == 404) {
                null
            } else {
                if (connection.responseCode !in 200..299) error("The server returned ${connection.responseCode}.")
                val value = JSONObject(connection.inputStream.use { it.readBoundedResponseText() })
                DashboardSnapshot(
                    reading = value.requiredHomeArray("reading").mapObjects(::continueReading),
                    listening = value.requiredHomeArray("listening").mapObjectsNotNull(::continueListening),
                    watching = value.requiredHomeArray("watching").mapObjects(VideoClient()::video),
                )
            }
        } finally { connection.disconnect() }
    }

    fun reading(credential: DeviceCredential): List<ContinueReading> = PerformanceDiagnostics.measure("network", "load_dashboard_reading") { array(credential, "/api/android/v1/reading?limit=$HOME_SECTION_LIMIT") }.map(::continueReading)

    fun held(credential: DeviceCredential): List<ContinueReading> = array(credential, "/api/android/v1/reading?held=true&limit=$HOME_SECTION_LIMIT").map(::continueReading)

    fun finished(credential: DeviceCredential): List<ContinueReading> = array(credential, "/api/android/v1/reading?finished=true&limit=$HOME_SECTION_LIMIT").map(::continueReading)

    fun listening(credential: DeviceCredential): List<ContinueListening> = PerformanceDiagnostics.measure("network", "load_dashboard_listening") { array(credential, "/api/android/v1/audio-progress?continue=1&limit=$HOME_SECTION_LIMIT") }.mapNotNull(::continueListening)

    fun dismissReading(credential: DeviceCredential, book: BookSummary) {
        val path = "/api/android/v1/reading/${encode(book.libraryId)}/${encode(book.id)}/dismiss"
        val connection = URI(credential.serverUrl + path).toURL().openConnection() as HttpURLConnection
        connection.requestMethod = "POST"
        connection.connectTimeout = 10_000
        connection.readTimeout = 20_000
        connection.instanceFollowRedirects = false
        connection.setRequestProperty("Authorization", "Bearer ${credential.token}")
        connection.setRequestProperty("Accept", "application/json")
        try {
            val status = connection.responseCode
            reportIfExpiredDeviceSession(status)
            if (status !in 200..299) error("The server returned $status.")
        } finally { connection.disconnect() }
    }

    fun setListeningDismissed(credential: DeviceCredential, item: ContinueListening, dismissed: Boolean) {
        val path = "/api/android/v1/audio-progress/${encode(item.track.libraryId)}/${encode(item.track.id)}"
        val connection = URI(credential.serverUrl + path).toURL().openConnection() as HttpURLConnection
        connection.requestMethod = "PATCH"
        connection.connectTimeout = 10_000
        connection.readTimeout = 20_000
        connection.doOutput = true
        connection.instanceFollowRedirects = false
        connection.setRequestProperty("Authorization", "Bearer ${credential.token}")
        connection.setRequestProperty("Content-Type", "application/json")
        try {
            connection.outputStream.use { output ->
                output.write(listeningDismissalBody(dismissed).toByteArray())
            }
            val status = connection.responseCode
            reportIfExpiredDeviceSession(status)
            if (status !in 200..299) error("The server returned $status.")
        } finally { connection.disconnect() }
    }

    fun latestListening(credential: DeviceCredential, item: ContinueListening): ContinueListening {
        val path = "/api/android/v1/audio-progress/${encode(item.track.libraryId)}/${encode(item.track.id)}"
        val connection = openAndroidApiGet(credential, path, connectTimeoutMillis = 5_000, readTimeoutMillis = 8_000)
        return try {
            if (connection.responseCode !in 200..299) error("The server returned ${connection.responseCode}.")
            val progress = JSONObject(connection.inputStream.use { it.readBoundedResponseText() })
            listeningAt(item, progress.optDouble("position_seconds").finiteOrZero(), progress.optDouble("duration_seconds", item.durationSeconds).finiteOrZero())
        } finally { connection.disconnect() }
    }

    private fun array(credential: DeviceCredential, path: String): List<org.json.JSONObject> {
        val connection = openAndroidApiGet(credential, path)
        return try {
            if (connection.responseCode !in 200..299) error("The server returned ${connection.responseCode}.")
            val values = JSONArray(connection.inputStream.use { it.readBoundedResponseText() }).requireAtMost(HOME_SECTION_LIMIT, "Dashboard section")
            List(values.length()) { values.getJSONObject(it) }
        } finally { connection.disconnect() }
    }

    private fun encode(value: String) = java.net.URLEncoder.encode(value, Charsets.UTF_8.name()).replace("+", "%20")
}

private fun JSONObject.requiredHomeArray(name: String): JSONArray =
    getJSONArray(name).requireAtMost(HOME_SECTION_LIMIT, "Dashboard $name")

private fun <T> JSONArray.mapObjects(transform: (JSONObject) -> T) =
    List(length()) { transform(getJSONObject(it)) }

private fun JSONArray.mapObjectsNotNull(transform: (JSONObject) -> ContinueListening?) =
    List(length()) { getJSONObject(it) }.mapNotNull(transform)

internal fun continueReading(value: JSONObject) = ContinueReading(
    value.toBookSummary(),
    value.optDouble("fraction").finiteOrZero().coerceIn(0.0, 1.0), value.optLong("updated_at"),
)

internal fun listeningDismissalBody(dismissed: Boolean): String =
    JSONObject().put("dismissed", dismissed).toString()

internal fun continueListening(value: JSONObject): ContinueListening? {
    val progress = value.optJSONObject("progress") ?: return null
    if (progress.optBoolean("finished") || progress.optBoolean("dismissed")) return null
    val duration = progress.optDouble("duration_seconds", value.optDouble("duration_seconds")).finiteOrZero().coerceAtLeast(0.0)
    val position = progress.optDouble("position_seconds").finiteOrZero().coerceIn(0.0, duration)
    return ContinueListening(
        AudioTrack(
            id = value.getString("id"), libraryId = value.getString("library_id"), title = value.optString("title", "Untitled track"),
            artist = value.optString("artist", "Unknown artist"), album = value.optString("album"), durationSeconds = duration,
            replayTrackDb = value.optionalFiniteDouble("replaygain_track_db"), replayAlbumDb = value.optionalFiniteDouble("replaygain_album_db"),
            replayTrackPeak = value.optionalFiniteDouble("replaygain_track_peak"), replayAlbumPeak = value.optionalFiniteDouble("replaygain_album_peak"),
            libraryKind = value.optString("library_kind"), resumePositionSeconds = position,
        ),
        position, duration,
    )
}

internal fun listeningAt(item: ContinueListening, positionSeconds: Double, durationSeconds: Double): ContinueListening {
    val duration = durationSeconds.takeIf { it.isFinite() && it > 0.0 } ?: item.durationSeconds.coerceAtLeast(0.0)
    val position = positionSeconds.takeIf(Double::isFinite)?.coerceIn(0.0, duration) ?: item.positionSeconds
    return item.copy(track = item.track.copy(durationSeconds = duration, resumePositionSeconds = position), positionSeconds = position, durationSeconds = duration)
}

private fun Double.finiteOrZero() = takeIf(Double::isFinite) ?: 0.0
