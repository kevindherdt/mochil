package net.kdhmedia.mochil.opds

import android.content.Context
import java.io.File
import java.net.URI
import java.security.MessageDigest
import java.util.concurrent.TimeUnit
import kotlinx.coroutines.sync.Semaphore
import kotlinx.coroutines.sync.withPermit
import okhttp3.OkHttpClient
import okhttp3.Request

class OpdsArtworkRepository(context: Context) {
    private val directory = File(context.cacheDir, "opds-artwork").apply { mkdirs() }
    private val concurrency = Semaphore(4)
    // Trimming used to re-scan and sum the whole directory on every single write. This instance is
    // held at OpdsViewModel scope (one per OPDS screen session), so track the running total in memory
    // instead: one real directory scan the first time anything is written, then O(1) bookkeeping.
    private val cacheSize = ArtworkCacheSize(directory)

    suspend fun load(connection: OpdsConnection, artworkUrl: String): File = concurrency.withPermit {
        val origin = validatedHttps(connection.catalogueUrl)
        var current = validatedHttps(artworkUrl)
        val target = File(directory, artworkCacheKey(connection, artworkUrl))
        if (target.isFile && target.length() in 1..MAX_ARTWORK_BYTES) {
            target.setLastModified(System.currentTimeMillis())
            return@withPermit target
        }
        repeat(MAX_REDIRECTS + 1) { redirect ->
            val request = Request.Builder().url(current.toString()).get().header("Accept", "image/*").apply {
                if (sameArtworkOrigin(origin, current) && connection.username.isNotBlank()) header("Authorization", basicOpdsAuthorization(connection))
            }.build()
            HTTP.newCall(request).execute().use { response ->
                if (response.code in 300..399) {
                    require(redirect < MAX_REDIRECTS) { "The OPDS artwork returned too many redirects." }
                    current = validatedHttps(current.resolve(response.header("Location") ?: error("The OPDS artwork returned an invalid redirect.")).toString())
                    return@repeat
                }
                require(response.isSuccessful) { "The OPDS artwork server returned ${response.code}." }
                require(response.header("Content-Type").orEmpty().substringBefore(';').trim().startsWith("image/")) { "The OPDS artwork response was not an image." }
                val announced = response.body.contentLength()
                require(announced <= MAX_ARTWORK_BYTES) { "The OPDS artwork is too large." }
                val temporary = File(directory, ".${target.name}.part")
                try {
                    val bytes = response.body.byteStream().use { readBounded(it, MAX_ARTWORK_BYTES.toInt()) }
                    require(bytes.isNotEmpty()) { "The OPDS artwork response was empty." }
                    temporary.writeBytes(bytes)
                    require(temporary.renameTo(target) || runCatching { temporary.copyTo(target, overwrite = true); true }.getOrDefault(false)) { "Could not cache OPDS artwork." }
                    target.setLastModified(System.currentTimeMillis())
                    trimArtworkCache(directory, target, cacheSize, bytes.size.toLong())
                } finally { temporary.delete() }
                return@withPermit target
            }
        }
        error("The OPDS artwork request could not be completed.")
    }

    private companion object {
        const val MAX_REDIRECTS = 3
        const val MAX_ARTWORK_BYTES = 12L * 1024 * 1024
        val HTTP = OkHttpClient.Builder().followRedirects(false).followSslRedirects(false).connectTimeout(12, TimeUnit.SECONDS).readTimeout(20, TimeUnit.SECONDS).build()
    }
}

/** Running total of the artwork cache directory's size, scanned from disk only once (lazily) and
 *  then updated incrementally by callers, instead of a full listFiles()+sumOf() on every write. */
internal class ArtworkCacheSize(private val directory: File) {
    private var total: Long? = null

    /** Adds [delta] (negative to subtract) to the tracked total, scanning the directory first if this
     *  is the first call. Returns the new total. */
    @Synchronized fun add(delta: Long): Long {
        val current = (total ?: scan()) + delta
        total = current
        return current
    }

    private fun scan(): Long = directory.listFiles()?.filter { it.isFile && !it.name.endsWith(".part") }?.sumOf(File::length) ?: 0L
}

private fun trimArtworkCache(directory: File, protected: File, cacheSize: ArtworkCacheSize, writtenBytes: Long) {
    val total = cacheSize.add(writtenBytes)
    if (total <= MAX_CACHE_BYTES) return
    var removed = 0L
    directory.listFiles()?.filter { it.isFile && !it.name.endsWith(".part") && it != protected }?.sortedBy(File::lastModified)?.forEach { file ->
        if (total - removed <= MAX_CACHE_BYTES) return@forEach
        val size = file.length()
        if (file.delete()) removed += size
    }
    if (removed > 0) cacheSize.add(-removed)
}

private const val MAX_CACHE_BYTES = 256L * 1024 * 1024

/** Authenticated artwork must not survive as the cache hit for another source credential. */
internal fun artworkCacheKey(connection: OpdsConnection, url: String): String = MessageDigest.getInstance("SHA-256")
    .digest("${connection.id}\u0000${connection.cacheGeneration}\u0000$url".encodeToByteArray())
    .take(16).joinToString("") { "%02x".format(it) }
internal fun basicOpdsAuthorization(connection: OpdsConnection) = basicAuthorization(connection.username, connection.password)
internal fun sameArtworkOrigin(left: URI, right: URI) = left.scheme.equals(right.scheme, true) && left.host.equals(right.host, true) && (if (left.port >= 0) left.port else 443) == (if (right.port >= 0) right.port else 443)

private fun basicAuthorization(username: String, password: String): String = "Basic " + java.util.Base64.getEncoder()
    .encodeToString("$username:$password".toByteArray(Charsets.UTF_8))
