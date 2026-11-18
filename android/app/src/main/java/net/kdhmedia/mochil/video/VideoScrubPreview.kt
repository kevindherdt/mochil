package net.kdhmedia.mochil.video

import android.content.Context
import android.graphics.Bitmap
import android.graphics.BitmapFactory
import android.graphics.Color
import android.graphics.drawable.GradientDrawable
import android.view.Gravity
import android.view.View
import android.widget.FrameLayout
import android.widget.ImageView
import android.widget.TextView
import java.io.ByteArrayOutputStream
import java.net.HttpURLConnection
import java.net.URI
import java.net.URLEncoder
import java.nio.charset.StandardCharsets
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

internal fun quantizedVideoPreviewSeconds(positionMs: Long, durationSeconds: Double, intervalSeconds: Int = 10): Int? {
    if (!durationSeconds.isFinite() || durationSeconds <= 0.0 || intervalSeconds <= 0) return null
    val maximum = (durationSeconds.toLong() - 1L).coerceAtLeast(0L)
    val position = (positionMs.coerceAtLeast(0L) / 1_000L).coerceAtMost(maximum)
    return (position / intervalSeconds * intervalSeconds).toInt()
}

internal fun videoPreviewPath(entry: VideoPlaybackEntry, seconds: Int): String =
    "/api/android/v1/video-previews/${encodePreviewSegment(entry.libraryId)}/${encodePreviewSegment(entry.itemId)}/${seconds.coerceAtLeast(0)}.jpg" +
        entry.revision.takeIf(String::isNotBlank)?.let { "?v=${encodePreviewSegment(it)}" }.orEmpty()

internal fun loadVideoPreview(serverUrl: String, token: String, entry: VideoPlaybackEntry, seconds: Int): Bitmap? {
    if (serverUrl.isBlank() || token.isBlank() || entry.libraryId == "local" || entry.libraryId.startsWith("webdav:")) return null
    val connection = (URI(serverUrl.trimEnd('/') + videoPreviewPath(entry, seconds)).toURL().openConnection() as HttpURLConnection).apply {
        requestMethod = "GET"
        connectTimeout = 7_000
        readTimeout = 10_000
        instanceFollowRedirects = false
        setRequestProperty("Authorization", "Bearer $token")
        setRequestProperty("Accept", "image/jpeg")
    }
    return try {
        if (connection.responseCode !in 200..299 || !connection.contentType.orEmpty().lowercase().startsWith("image/")) return null
        val bytes = connection.inputStream.use { input ->
            val output = ByteArrayOutputStream()
            val buffer = ByteArray(16 * 1024)
            var total = 0
            while (true) {
                val count = input.read(buffer)
                if (count < 0) break
                total += count
                if (total > MAX_PREVIEW_BYTES) return null
                output.write(buffer, 0, count)
            }
            output.toByteArray()
        }
        val bounds = BitmapFactory.Options().apply { inJustDecodeBounds = true }
        BitmapFactory.decodeByteArray(bytes, 0, bytes.size, bounds)
        if (bounds.outWidth <= 0 || bounds.outHeight <= 0) return null
        var sample = 1
        while (bounds.outWidth / sample > 720 || bounds.outHeight / sample > 480) sample *= 2
        BitmapFactory.decodeByteArray(bytes, 0, bytes.size, BitmapFactory.Options().apply { inSampleSize = sample })
    } finally {
        connection.disconnect()
    }
}

/** Scrubbing drags repeatedly land back on the same quantized timeline bucket (see
 *  [quantizedVideoPreviewSeconds]); without a cache every one of those re-fetches and re-decodes
 *  the same frame from the server. Bounded LRU, same `LinkedHashMap(..., accessOrder = true)` +
 *  `removeEldestEntry` shape used elsewhere in this codebase (TvGridPositionStore, VideoProgressClient),
 *  generalized over the value type so the eviction logic itself can be unit tested without a real
 *  [Bitmap] (which the JVM unit test SDK stub cannot construct or mock). */
internal class BoundedLruCache<K, V : Any>(private val maxEntries: Int = 24) {
    init { require(maxEntries > 0) }
    private val entries = object : LinkedHashMap<K, V>(maxEntries, 0.75f, true) {
        override fun removeEldestEntry(eldest: MutableMap.MutableEntry<K, V>?): Boolean = size > maxEntries
    }

    @Synchronized fun get(key: K): V? = entries[key]
    @Synchronized fun put(key: K, value: V) { entries[key] = value }
    @Synchronized fun size(): Int = entries.size
    @Synchronized fun clear() = entries.clear()
}

/** The cache key for one timeline bucket: scoped to the item (and its revision, so a re-encoded
 *  video does not serve a stale frame) plus the quantized second. */
internal fun videoScrubPreviewCacheKey(entry: VideoPlaybackEntry, seconds: Int): String =
    "${entry.libraryId}\u0000${entry.itemId}\u0000${entry.revision}\u0000$seconds"

/** Coordinates one scrub-preview request at a time: quantizes the position, serves from
 *  [cache] on a hit, or fetches on a miss and posts the frame back only if the caller confirms
 *  (via [isCurrent]) the request is still relevant by the time it lands. Extracted out of
 *  VideoPlayerActivity to keep that file within the project's module-size budget. */
internal class VideoScrubController(
    private val scope: CoroutineScope,
    private val cache: BoundedLruCache<String, Bitmap>,
    private val onFrame: (Int, Bitmap) -> Unit,
) {
    private var job: Job? = null
    var currentSecond: Int? = null
        private set

    fun request(
        entry: VideoPlaybackEntry,
        positionMs: Long,
        durationSeconds: Double,
        credential: net.kdhmedia.mochil.auth.DeviceCredential?,
        isCurrent: () -> Boolean,
    ): Int? {
        val seconds = quantizedVideoPreviewSeconds(positionMs, durationSeconds) ?: return null
        if (currentSecond == seconds) return seconds
        currentSecond = seconds
        job?.cancel()
        val cacheKey = videoScrubPreviewCacheKey(entry, seconds)
        cache.get(cacheKey)?.let { onFrame(seconds, it); return seconds }
        val cred = credential ?: return seconds
        job = scope.launch(Dispatchers.IO) {
            val bitmap = runCatching { loadVideoPreview(cred.serverUrl, cred.token, entry, seconds) }.getOrNull() ?: return@launch
            cache.put(cacheKey, bitmap)
            withContext(Dispatchers.Main) {
                if (isCurrent() && currentSecond == seconds) onFrame(seconds, bitmap)
            }
        }
        return seconds
    }

    fun cancel() {
        job?.cancel()
        job = null
        currentSecond = null
    }
}

internal class VideoScrubPreviewView(context: Context) : FrameLayout(context) {
    private val image = ImageView(context).apply { scaleType = ImageView.ScaleType.CENTER_CROP }
    private val time = TextView(context).apply {
        setTextColor(Color.WHITE)
        textSize = 14f
        gravity = Gravity.CENTER
        setPadding(dp(8), dp(3), dp(8), dp(3))
        background = GradientDrawable().apply { setColor(0xCC000000.toInt()); cornerRadius = dp(6).toFloat() }
    }

    init {
        visibility = View.GONE
        isClickable = false
        isFocusable = false
        clipToOutline = true
        background = GradientDrawable().apply { setColor(0xE6151515.toInt()); cornerRadius = dp(10).toFloat() }
        addView(image, LayoutParams(LayoutParams.MATCH_PARENT, LayoutParams.MATCH_PARENT))
        addView(time, LayoutParams(LayoutParams.WRAP_CONTENT, LayoutParams.WRAP_CONTENT, Gravity.BOTTOM or Gravity.CENTER_HORIZONTAL).apply { bottomMargin = dp(7) })
    }

    fun showPosition(seconds: Int) {
        time.text = formatPreviewTime(seconds)
        visibility = View.VISIBLE
    }

    fun showFrame(seconds: Int, bitmap: Bitmap) {
        image.setImageBitmap(bitmap)
        showPosition(seconds)
    }

    fun hidePreview() {
        visibility = View.GONE
        image.setImageDrawable(null)
    }

    private fun dp(value: Int) = (value * resources.displayMetrics.density).toInt()
}

private fun encodePreviewSegment(value: String) = URLEncoder.encode(value, StandardCharsets.UTF_8.toString()).replace("+", "%20")

private fun formatPreviewTime(seconds: Int): String {
    val total = seconds.coerceAtLeast(0)
    val hours = total / 3600
    val minutes = total % 3600 / 60
    val remaining = total % 60
    return if (hours > 0) "%d:%02d:%02d".format(hours, minutes, remaining) else "%d:%02d".format(minutes, remaining)
}

private const val MAX_PREVIEW_BYTES = 2 * 1024 * 1024
