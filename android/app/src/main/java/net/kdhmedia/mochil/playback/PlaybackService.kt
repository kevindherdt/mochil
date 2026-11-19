package net.kdhmedia.mochil.playback

import android.content.Context
import android.content.SharedPreferences
import androidx.media3.common.AudioAttributes
import androidx.media3.common.C
import androidx.media3.common.MediaItem
import androidx.media3.common.util.UnstableApi
import androidx.media3.datasource.DefaultHttpDataSource
import androidx.media3.exoplayer.ExoPlayer
import androidx.media3.exoplayer.DefaultRenderersFactory
import androidx.media3.exoplayer.audio.AudioSink
import androidx.media3.exoplayer.audio.DefaultAudioSink
import androidx.media3.exoplayer.source.DefaultMediaSourceFactory
import androidx.media3.session.MediaSession
import androidx.media3.session.MediaSessionService
import androidx.media3.common.Player
import androidx.media3.common.PlaybackException
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.currentCoroutineContext
import kotlinx.coroutines.ensureActive
import kotlinx.coroutines.delay
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch
import net.kdhmedia.mochil.auth.DeviceCredentialStore
import net.kdhmedia.mochil.auth.DeviceCredential
import net.kdhmedia.mochil.diagnostics.PerformanceDiagnostics
import net.kdhmedia.mochil.webdav.BoboMediaDataSourceFactory

@androidx.annotation.OptIn(markerClass = [UnstableApi::class])
class PlaybackService : MediaSessionService() {
    private var session: MediaSession? = null
    // Media3 sessions may only be inspected from their application thread.
    // Network persistence is dispatched separately after taking a small snapshot.
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Main.immediate)
    private val progressClient = AudioProgressClient()
    private val progressSubmissions = AudioProgressSubmissions()
    private lateinit var progressJournal: AudioProgressJournal
    private lateinit var progressDelivery: PlaybackProgressDeliveryQueue
    private lateinit var gainProcessor: ReplayGainAudioProcessor
    private lateinit var gainPreferences: ReplayGainPreferences
    private lateinit var sleepTimer: SleepTimerStore
    private lateinit var playbackSnapshot: PlaybackSnapshotStore
    private val seekOrigin = AudioSeekOrigin()
    private var gainPreferenceListener: SharedPreferences.OnSharedPreferenceChangeListener? = null

    override fun onCreate() {
        super.onCreate()
        PerformanceDiagnostics.initialize(applicationContext)
        val credential = DeviceCredentialStore(this).load()
        val token = credential?.token
        val httpFactory = DefaultHttpDataSource.Factory().apply {
            if (token != null) setDefaultRequestProperties(mapOf("Authorization" to "Bearer $token"))
        }
        gainProcessor = ReplayGainAudioProcessor()
        gainPreferences = ReplayGainPreferences(this)
        sleepTimer = SleepTimerStore(this)
        playbackSnapshot = PlaybackSnapshotStore(this)
        progressJournal = AudioProgressJournal(this)
        progressDelivery = PlaybackProgressDeliveryQueue(scope, onFailure = {
            android.util.Log.w("PlaybackService", "Could not deliver audio progress")
        })
        if (credential != null) scope.launch(Dispatchers.IO) {
            progressJournal.entries(credential.deviceId).forEach { pending ->
                currentCoroutineContext().ensureActive()
                progressSubmissions.submit(pending, replay = true, committed = pending)?.let { ticket ->
                    enqueueProgress(credential, ticket)
                }
            }
        }
        val player = ExoPlayer.Builder(this, ReplayGainRenderersFactory(this, gainProcessor))
            .setMediaSourceFactory(DefaultMediaSourceFactory(BoboMediaDataSourceFactory(this, httpFactory)))
            .setAudioAttributes(
                AudioAttributes.Builder().setUsage(C.USAGE_MEDIA).setContentType(C.AUDIO_CONTENT_TYPE_MUSIC).build(),
                true,
            )
            .setHandleAudioBecomingNoisy(true)
            .build()
        playbackSnapshot.restore(credential)?.let { restored ->
            player.setMediaItems(restored.items, restored.startIndex, restored.positionMs)
            player.repeatMode = restored.repeatMode
            player.shuffleModeEnabled = restored.shuffle
            player.setPlaybackSpeed(restored.speed)
            player.prepare()
        }
        session = MediaSession.Builder(this, player).build()
        player.addListener(object : Player.Listener {
            override fun onPlayerError(error: PlaybackException) {
                PerformanceDiagnostics.record("playback", playbackDiagnosticName(error.errorCodeName), outcome = "error")
                playbackDetailNames(error).forEach { PerformanceDiagnostics.record("playback", it, outcome = "error") }
                net.kdhmedia.mochil.network.httpStatusFromPlaybackError(error)?.let { net.kdhmedia.mochil.network.reportIfExpiredDeviceSession(it) }
                // A converted stream is a convenience for slow connections, never a requirement: if it fails, the
                // original file takes over at the same position instead of playback simply stopping.
                val item = player.currentMediaItem ?: return
                val original = item.localConfiguration?.uri?.toString()?.let(::originalStreamUrl) ?: return
                val index = player.currentMediaItemIndex
                val position = player.currentPosition
                player.replaceMediaItem(index, item.buildUpon().setUri(original).build())
                player.seekTo(index, position)
                player.prepare()
                player.play()
            }
            override fun onIsPlayingChanged(isPlaying: Boolean) {
                if (isPlaying) savePlaybackSnapshot() else saveProgress(durable = true)
            }
            override fun onMediaItemTransition(mediaItem: MediaItem?, reason: Int) {
                applyReplayGain(mediaItem)
                if (shouldContinueAfterAutomaticTransition(reason, player.playWhenReady)) player.play()
                savePlaybackSnapshot()
            }
            override fun onPositionDiscontinuity(oldPosition: Player.PositionInfo, newPosition: Player.PositionInfo, reason: Int) {
                if (oldPosition.mediaItemIndex != newPosition.mediaItemIndex) {
                    val oldItem = oldPosition.mediaItem
                    val durationMs = resolvedPlaybackDurationMs(
                        oldItem?.mediaMetadata?.extras?.optionalDouble("duration_seconds"),
                        0L,
                    )
                    // The next item's shutdown checkpoint cannot recover this departing item.
                    if (oldItem != null) saveProgress(oldItem, oldPosition.positionMs, durationMs, durable = true)
                }
                if (reason == Player.DISCONTINUITY_REASON_SEEK) {
                    val current = newPosition.mediaItem
                    val durationMs = resolvedPlaybackDurationMs(
                        current?.mediaMetadata?.extras?.optionalDouble("duration_seconds"),
                        player.duration,
                    )
                    if (current != null) saveProgress(
                        current,
                        newPosition.positionMs,
                        durationMs,
                        durable = true,
                        explicitSeek = seekOrigin.isExplicit(
                            current.mediaMetadata.extras?.getLong("automatic_start_position_ms", -1L) ?: -1L,
                            current.mediaMetadata.extras?.getString("automatic_seek_token").orEmpty(),
                            newPosition.positionMs,
                        ),
                    )
                }
                if (reason == Player.DISCONTINUITY_REASON_SEEK || oldPosition.mediaItemIndex != newPosition.mediaItemIndex) savePlaybackSnapshot()
            }
        })
        gainPreferenceListener = SharedPreferences.OnSharedPreferenceChangeListener { _, key ->
            if (key == ReplayGainPreferences.MODE) scope.launch { applyReplayGain(player.currentMediaItem) }
        }.also { gainPreferences.sharedPreferences().registerOnSharedPreferenceChangeListener(it) }
        applyReplayGain(player.currentMediaItem)
        scope.launch {
            while (isActive) {
                delay(15_000)
                if (shouldPersistPeriodicProgress(player.isPlaying, player.currentMediaItem != null)) saveProgress()
            }
        }
        scope.launch {
            while (isActive) {
                delay(1_000)
                pauseForExpiredSleepTimer(sleepTimer::consumeExpired, player::pause)
            }
        }
    }

    override fun onGetSession(controllerInfo: MediaSession.ControllerInfo): MediaSession? = session

    override fun onTaskRemoved(rootIntent: android.content.Intent?) {
        session?.player?.let { player ->
            saveProgress(durable = true)
            player.stop()
            player.clearMediaItems()
        }
        playbackSnapshot.clear(durable = true)
        stopSelf()
        super.onTaskRemoved(rootIntent)
    }

    override fun onDestroy() {
        // Persist before cancelling delivery; Android may destroy the service without task removal.
        saveProgress(durable = true)
        scope.cancel()
        gainPreferenceListener?.let { gainPreferences.sharedPreferences().unregisterOnSharedPreferenceChangeListener(it) }
        gainPreferenceListener = null
        session?.run { player.release(); release() }
        session = null
        super.onDestroy()
    }

    private fun applyReplayGain(item: MediaItem?) {
        val extras = item?.mediaMetadata?.extras
        val metadata = ReplayGainMetadata(
            trackGainDb = extras?.optionalDouble("replay_track_db"),
            albumGainDb = extras?.optionalDouble("replay_album_db"),
            trackPeak = extras?.optionalDouble("replay_track_peak"),
            albumPeak = extras?.optionalDouble("replay_album_peak"),
        )
        gainProcessor.setMultiplier(ReplayGainCalculator.multiplier(gainPreferences.mode(), metadata))
    }

    private fun savePlaybackSnapshot(durable: Boolean = false) {
        session?.player?.let { playbackSnapshot.save(it, durable) }
    }

    private fun saveProgress(durable: Boolean = false) {
        val player = session?.player ?: return
        playbackSnapshot.save(player, durable)
        val item = player.currentMediaItem ?: return
        saveProgress(item, player.currentPosition, player.duration, durable)
    }

    private fun saveProgress(item: MediaItem, positionMs: Long, durationMs: Long, durable: Boolean = false, explicitSeek: Boolean = false) {
        val libraryId = item.mediaMetadata.extras?.getString("library_id").orEmpty()
        if (libraryId == "local" || libraryId.startsWith("webdav:")) return
        val credential = DeviceCredentialStore(this).load() ?: return
        val position = positionMs.coerceAtLeast(0) / 1_000.0
        val duration = durationMs.takeIf { it > 0 }?.div(1_000.0) ?: return
        val itemId = item.mediaId
        val pending = PendingAudioProgress(
            libraryId,
            itemId,
            position,
            duration,
            explicitSeek,
            explicitSeekAtEpochMs = if (explicitSeek) System.currentTimeMillis() else 0,
            owner = audioProgressOwnerIdentity(credential.deviceId),
        )
        val committed = if (durable) progressJournal.put(pending) else null
        if (durable && committed == null) PerformanceDiagnostics.record("storage", "audio_progress_not_saved")
        val submitted = committed ?: pending
        val ticket = progressSubmissions.submit(submitted, committed = committed) ?: return
        enqueueProgress(credential, ticket)
    }

    private fun enqueueProgress(credential: DeviceCredential, ticket: AudioProgressSubmissions.Ticket) {
        progressDelivery.submit(ticket.key, mayQueue = { progressSubmissions.isCurrent(ticket) }) {
            val context = currentCoroutineContext()
            replayPendingAudioProgress(
                listOf(ticket.value),
                send = { value -> progressClient.save(credential, value.libraryId, value.itemId, value.positionSeconds, value.durationSeconds, value.explicitSeekForReplay(System.currentTimeMillis())) {
                    context.ensureActive()
                    progressSubmissions.isCurrent(ticket)
                } },
                acknowledge = {
                    progressSubmissions.acknowledge(ticket)
                    ticket.committed?.let(progressJournal::removeIfCurrent)
                },
            )
        }
    }
}

/** The original-file URL for a converted (HLS) track URL, or null when the URL is not a converted stream. */
internal fun originalStreamUrl(uri: String): String? {
    val marker = "/audio-hls/"
    val start = uri.indexOf(marker)
    if (start < 0) return null
    val parts = uri.substring(start + marker.length).substringBefore('?').split('/')
    if (parts.size != 4 || parts[3] != "index.m3u8") return null
    return uri.substring(0, start) + "/audio-stream/" + parts[0] + "/" + parts[1] + "/audio"
}

/** The error code, the exception type behind it and, for a refused request, the HTTP status; all bounded, none carrying addresses or titles. */
internal fun playbackDetailNames(error: PlaybackException): List<String> = buildList {
    add("audio_code_${error.errorCode.coerceIn(0, 99_999)}")
    val cause = generateSequence<Throwable>(error.cause) { it.cause }.take(4).toList()
    cause.firstNotNullOfOrNull { it as? androidx.media3.datasource.HttpDataSource.InvalidResponseCodeException }?.let { add("audio_http_${it.responseCode.coerceIn(100, 599)}") }
    cause.firstOrNull()?.javaClass?.simpleName?.lowercase()?.replace(Regex("[^a-z0-9]"), "_")?.take(40)?.takeIf { it.isNotBlank() }?.let { add("audio_cause_$it") }
}

internal fun playbackDiagnosticName(errorCodeName: String): String = when {
    "DECOD" in errorCodeName || "FORMAT" in errorCodeName -> "audio_decoder_error"
    "NETWORK" in errorCodeName || "TIMEOUT" in errorCodeName -> "audio_network_error"
    "IO_" in errorCodeName -> "audio_source_error"
    "PARSING" in errorCodeName -> "audio_parsing_error"
    "AUDIO_TRACK" in errorCodeName -> "audio_track_error"
    "UNSPECIFIED" in errorCodeName || "REMOTE" in errorCodeName || "RUNTIME" in errorCodeName -> "audio_unspecified_error"
    else -> "audio_playback_error"
}

private fun android.os.Bundle.optionalDouble(key: String): Double? = if (containsKey(key)) getDouble(key).takeIf(Double::isFinite) else null

internal fun resolvedPlaybackDurationMs(metadataSeconds: Double?, playerDurationMs: Long): Long =
    metadataSeconds?.takeIf { it.isFinite() && it > 0.0 }?.times(1_000.0)?.toLong()
        ?: playerDurationMs.coerceAtLeast(0)

/** Converted sources may briefly clear play intent while the next manifest is prepared. */
internal fun shouldContinueAfterAutomaticTransition(reason: Int, playWhenReady: Boolean): Boolean =
    reason == Player.MEDIA_ITEM_TRANSITION_REASON_AUTO && !playWhenReady

@UnstableApi
private class ReplayGainRenderersFactory(context: Context, private val processor: ReplayGainAudioProcessor) : DefaultRenderersFactory(context) {
    override fun buildAudioSink(context: Context, enableFloatOutput: Boolean, enableAudioOutputPlaybackParameters: Boolean): AudioSink =
        DefaultAudioSink.Builder(context)
            .setAudioProcessors(arrayOf(processor))
            .setEnableFloatOutput(false)
            .setEnableAudioOutputPlaybackParameters(enableAudioOutputPlaybackParameters)
            .build()
}
