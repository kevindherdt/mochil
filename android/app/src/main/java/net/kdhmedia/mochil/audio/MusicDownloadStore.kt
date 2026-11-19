@file:android.annotation.SuppressLint("ApplySharedPref") // A completed media file is not exposed until its durable index is committed.
package net.kdhmedia.mochil.audio

import android.content.Context
import android.net.Uri
import java.io.File
import java.net.HttpURLConnection
import java.net.URI
import java.nio.file.StandardCopyOption
import java.security.MessageDigest
import net.kdhmedia.mochil.auth.DeviceCredential
import net.kdhmedia.mochil.catalogue.DownloadLocationStore
import net.kdhmedia.mochil.offline.DownloadSpacePolicy
import net.kdhmedia.mochil.offline.ResumableHttpDownload
import org.json.JSONArray
import org.json.JSONObject

class OfflineAudioStore(private val context: Context) : AudioDownloadStore {
    private val preferences = context.getSharedPreferences("music_downloads", Context.MODE_PRIVATE)
    private val directory = File(context.filesDir, "music").apply { mkdirs() }
    private var cachedEntries: List<DownloadEntry>? = null
    private val mutations = OfflineAudioMutationGate(this)

    init {
        val existing = decode()
        if (existing.any { !it.externalUri.isNullOrBlank() }) writeExternalIndex(existing)
    }

    @Synchronized
    override fun ids(): Set<String> = validEntries().mapTo(mutableSetOf()) { it.track?.identityKey ?: it.key }

    @Synchronized
    override fun tracks(libraryId: String?): List<AudioTrack> = validEntries().mapNotNull { it.track }
        .filter { libraryId.isNullOrBlank() || it.libraryId == libraryId }
        .sortedWith(compareBy<AudioTrack> { it.artist.lowercase() }.thenBy { it.album.lowercase() }.thenBy { it.title.lowercase() })

    @Synchronized
    override fun snapshot(libraryId: String?): AudioDownloadSnapshot {
        val entries = validEntries()
        return AudioDownloadSnapshot(
            ids = entries.mapTo(mutableSetOf()) { it.track?.identityKey ?: it.key },
            tracks = entries.mapNotNull { it.track }
                .filter { libraryId.isNullOrBlank() || it.libraryId == libraryId }
                .sortedWith(compareBy<AudioTrack> { it.artist.lowercase() }.thenBy { it.album.lowercase() }.thenBy { it.title.lowercase() }),
        )
    }

    @Synchronized
    override fun contains(track: AudioTrack): Boolean = playbackUrl(track) != null

    @Synchronized
    override fun playbackUrl(track: AudioTrack): String? = resolveOfflineAudioLocation(
        entries = decode(), requested = track, trackOf = { it.track },
        resolve = { entry ->
            validLocalFile(entry.path)?.let { Uri.fromFile(it).toString() } ?: run {
                val location = DownloadLocationStore(context)
                entry.externalUri?.takeIf { location.owns(it) && location.canRead(it) }
            }
        },
    )

    /** Rebuild the app-private index after reinstall when the user selects the same folder again. */
    @Synchronized
    fun recoverExternalDownloads(): Int {
        val location = DownloadLocationStore(context)
        val current = decode()
        val currentKeys = current.mapTo(mutableSetOf()) { offlineAudioRecoveryKey(it.key, it.track) }
        val indexedAdditions = recoverableExternalDownloads(
            indexed = decode(location.readAudioIndex()),
            currentKeys = currentKeys,
            keyOf = { offlineAudioRecoveryKey(it.key, it.track) },
            isReadable = { entry ->
                entry.track != null && !entry.externalUri.isNullOrBlank() && location.canRead(entry.externalUri)
            },
        ).map { it.copy(path = "") }
        val knownUris = (current + indexedAdditions).mapNotNullTo(mutableSetOf()) { it.externalUri }
        val orphanAdditions = location.audioFiles().filter { it.uri !in knownUris }.map { file ->
            val title = file.displayName.substringBeforeLast('.').trim().ifBlank { "Recovered audio" }
            val id = stableExternalAudioId(file.uri)
            DownloadEntry(
                key = "recovered:$id",
                path = "",
                track = AudioTrack(
                    id = id,
                    libraryId = "",
                    title = title,
                    artist = "",
                    album = "",
                    durationSeconds = 0.0,
                    mediaType = file.mediaType,
                    sizeBytes = file.sizeBytes,
                ),
                externalUri = file.uri,
            )
        }
        val additions = indexedAdditions + orphanAdditions
        if (additions.isNotEmpty()) save(current + additions)
        return additions.size
    }

    /** Reconnect recovered podcast files after an RSS subscription is recreated. */
    @Synchronized
    internal fun reconcilePodcastFeeds(libraryId: String, feeds: List<OfflinePodcastFeed>): Int {
        if (libraryId.isBlank() || feeds.isEmpty()) return 0
        var changed = 0
        val entries = decode().map { entry ->
            val track = entry.track ?: return@map entry
            val reconciled = reconcileOfflinePodcastTrack(track, libraryId, feeds)
            if (reconciled == track) entry else {
                changed++
                entry.copy(key = reconciled.key, track = reconciled)
            }
        }.distinctBy { offlineAudioRecoveryKey(it.key, it.track) }
        if (changed > 0) save(entries)
        return changed
    }

    /** Rebinds recovered files using current episode metadata from one RSS feed. */
    @Synchronized
    internal fun reconcilePodcastEpisodes(libraryId: String, feedId: String, episodes: List<AudioTrack>): Int {
        if (libraryId.isBlank() || feedId.isBlank() || episodes.isEmpty()) return 0
        var changed = 0
        val entries = decode().map { entry ->
            val track = entry.track ?: return@map entry
            val reconciled = reconcileOfflinePodcastEpisode(track, libraryId, feedId, episodes)
            if (reconciled == track) entry else {
                changed++
                entry.copy(key = reconciled.key, track = reconciled)
            }
        }.distinctBy { offlineAudioRecoveryKey(it.key, it.track) }
        if (changed > 0) save(entries)
        return changed
    }

    override fun download(credential: DeviceCredential, track: AudioTrack, url: String) {
        mutations.transfer(
            fetch = { fetchDownload(credential, track, url) },
            publish = { target ->
                val previous = decode().firstOrNull { it.track?.let { saved -> sameOfflineAudioIdentity(saved, track) } == true }
                val locationStore = DownloadLocationStore(context)
                val displayName = track.title.ifBlank { "Track" } + MusicDownloadNaming.extensionFor(track.mediaType)
                val externalUri = locationStore.mirror(target, displayName, track.mediaType) ?: previous?.externalUri
                if (externalUri != null && externalUri != previous?.externalUri) locationStore.delete(previous?.externalUri)
                val entries = decode().filterNot { it.track?.let { saved -> sameOfflineAudioIdentity(saved, track) } == true } + DownloadEntry(track.key, target.absolutePath, track, externalUri)
                save(entries)
            },
        )
    }

    private fun fetchDownload(credential: DeviceCredential, track: AudioTrack, url: String): File {
        val extension = MusicDownloadNaming.extensionFor(track.mediaType)
        val target = File(directory, MusicDownloadNaming.safeName(track.key) + extension)
        val partial = File(directory, target.name + ".part")
        val connection = URI(url + if (url.contains('?')) "&download=1" else "?download=1").toURL().openConnection() as HttpURLConnection
        connection.connectTimeout = 15_000
        connection.readTimeout = 30_000
        connection.instanceFollowRedirects = false
        connection.setRequestProperty("Authorization", "Bearer ${credential.token}")
        val resume = ResumableHttpDownload.configure(connection, partial)
        try {
            if (connection.responseCode == ResumableHttpDownload.HTTP_RANGE_NOT_SATISFIABLE && resume.offset > 0) {
                ResumableHttpDownload.discard(partial)
                error("The saved partial download no longer matches the server file. Try again to restart it.")
            }
            if (connection.responseCode !in 200..299) error("The server returned ${connection.responseCode}.")
            val length = connection.contentLengthLong
            if (resume.offset == 0L) DownloadSpacePolicy.requireCapacity(directory, length, MAX_DOWNLOAD_BYTES)
            ResumableHttpDownload.copyResponse(connection, partial, directory, MAX_DOWNLOAD_BYTES, resume)
            runCatching { java.nio.file.Files.move(partial.toPath(), target.toPath(), StandardCopyOption.ATOMIC_MOVE, StandardCopyOption.REPLACE_EXISTING) }
                .recoverCatching { java.nio.file.Files.move(partial.toPath(), target.toPath(), StandardCopyOption.REPLACE_EXISTING) }
                .getOrElse { error("Could not finish the offline download.") }
            ResumableHttpDownload.finish(partial)
            return target
        } finally {
            connection.disconnect()
        }
    }

    override fun remove(track: AudioTrack): Boolean = mutations.mutate {
        val entry = validEntries(includeMissing = true).firstOrNull { candidate ->
            candidate.track?.let { sameOfflineAudioIdentity(it, track) } == true
        } ?: return@mutate true
        val local = validLocalFile(entry.path, includeMissing = true)
        if (entry.path.isNotBlank() && local == null) return@mutate false
        val localRemoved = local == null || !local.exists() || local.delete()
        val externalRemoved = DownloadLocationStore(context).delete(entry.externalUri)
        val removed = downloadRemovalComplete(localRemoved, externalRemoved)
        if (removed) {
            save(decode().filterNot { candidate -> candidate.track?.let { track -> entry.track?.let { sameOfflineAudioIdentity(track, it) } } == true })
        }
        removed
    }

    private fun decode(raw: String? = null): List<DownloadEntry> {
        if (raw == null) cachedEntries?.let { return it }
        val decoded = parseEntries(raw ?: preferences.getString("items", "[]"))
        if (raw == null) cachedEntries = decoded
        return decoded
    }

    private fun parseEntries(raw: String?): List<DownloadEntry> = runCatching {
        val values = JSONArray(raw ?: "[]")
        List(values.length()) { index -> values.getJSONObject(index).run {
            DownloadEntry(
                getString("key"), getString("path"),
                optJSONObject("track")?.let(AudioDownloadMetadata::decode),
                optString("external_uri").takeIf(String::isNotBlank),
            )
        } }
    }.getOrDefault(emptyList())

    private fun save(entries: List<DownloadEntry>) {
        val values = JSONArray()
        entries.forEach { entry -> values.put(JSONObject().put("key", entry.key).put("path", entry.path).put("external_uri", entry.externalUri).apply {
            entry.track?.let { track -> put("track", AudioDownloadMetadata.encode(track)) }
        }) }
        check(preferences.edit().putString("items", values.toString()).commit()) { "Could not save the offline audio index." }
        cachedEntries = entries.toList()
        writeExternalIndex(entries)
    }

    private fun writeExternalIndex(entries: List<DownloadEntry>) {
        val external = entries.filter { !it.externalUri.isNullOrBlank() && it.track != null }
        val values = JSONArray()
        external.forEach { entry -> values.put(JSONObject().put("key", entry.key).put("path", "").put("external_uri", entry.externalUri).apply {
            entry.track?.let { track -> put("track", AudioDownloadMetadata.encode(track)) }
        }) }
        DownloadLocationStore(context).writeAudioIndex(values.toString())
    }

    private fun validEntries(includeMissing: Boolean = false): List<DownloadEntry> = decode().filter { entry ->
        val localValid = validLocalFile(entry.path, includeMissing) != null
        val location = DownloadLocationStore(context)
        localValid || (location.owns(entry.externalUri) && (includeMissing || location.canRead(entry.externalUri)))
    }

    private fun validLocalFile(path: String, includeMissing: Boolean = false): File? =
        offlineAudioLocalFile(directory, path, includeMissing)

    private data class DownloadEntry(val key: String, val path: String, val track: AudioTrack? = null, val externalUri: String? = null)
    private val AudioTrack.key get() = identityKey

    companion object { private const val MAX_DOWNLOAD_BYTES = 2L * 1024 * 1024 * 1024 }
}

internal fun downloadRemovalComplete(localRemoved: Boolean, externalRemoved: Boolean): Boolean =
    localRemoved && externalRemoved

/** Library recreation must not restore a second copy of the same feed episode. */
internal fun offlineAudioRecoveryKey(storedKey: String, track: AudioTrack?): String =
    if (track?.libraryKind == "podcasts" && track.feedId.isNotBlank())
        "podcast\u0000${track.feedId}\u0000${track.id}"
    else track?.identityKey ?: storedKey

internal fun stableExternalAudioId(uri: String): String = MessageDigest.getInstance("SHA-256")
    .digest(uri.toByteArray(Charsets.UTF_8))
    .take(12)
    .joinToString("") { "%02x".format(it) }

internal object MusicDownloadNaming {
    // Hash the full identity: sanitizing punctuation or truncating keys aliases distinct tracks.
    // Existing downloads remain readable through their persisted index paths.
    fun safeName(value: String): String = MessageDigest.getInstance("SHA-256")
        .digest(value.toByteArray(Charsets.UTF_8))
        .joinToString("") { "%02x".format(it) }
    fun extensionFor(mediaType: String) = when (mediaType.lowercase()) {
        "audio/flac" -> ".flac"
        "audio/mp4", "audio/x-m4a", "audio/aac" -> ".m4a"
        "audio/ogg", "audio/opus" -> ".ogg"
        "audio/wav", "audio/x-wav" -> ".wav"
        "audio/webm" -> ".webm"
        else -> ".mp3"
    }
}
