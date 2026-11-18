package net.kdhmedia.mochil.video

import android.app.PictureInPictureParams
import android.content.pm.ActivityInfo
import android.content.res.Configuration
import android.os.Bundle
import android.util.Rational
import android.graphics.Rect
import android.widget.FrameLayout
import android.view.ViewGroup
import androidx.activity.ComponentActivity
import androidx.activity.OnBackPressedCallback
import androidx.activity.enableEdgeToEdge
import androidx.annotation.OptIn
import androidx.core.view.WindowCompat
import androidx.core.view.WindowInsetsCompat
import androidx.core.view.WindowInsetsControllerCompat
import androidx.lifecycle.lifecycleScope
import androidx.media3.common.C
import androidx.media3.common.MediaItem
import androidx.media3.common.PlaybackException
import androidx.media3.common.Player
import androidx.media3.common.util.UnstableApi
import androidx.media3.datasource.DefaultHttpDataSource
import androidx.media3.datasource.HttpDataSource
import androidx.media3.exoplayer.ExoPlayer
import androidx.media3.exoplayer.source.DefaultMediaSourceFactory
import androidx.media3.ui.PlayerView
import androidx.media3.ui.TimeBar
import androidx.media3.ui.TrackSelectionDialogBuilder
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch
import net.kdhmedia.mochil.auth.DeviceCredentialStore
import net.kdhmedia.mochil.diagnostics.PerformanceDiagnostics
import net.kdhmedia.mochil.isTelevisionDevice
import net.kdhmedia.mochil.playback.shouldPersistPeriodicProgress
import net.kdhmedia.mochil.webdav.BoboMediaDataSourceFactory

@OptIn(markerClass = [UnstableApi::class])
class VideoPlayerActivity : ComponentActivity() {
    private var player: ExoPlayer? = null
    private var progressJob: Job? = null
    private var bufferingJob: Job? = null
    private var reconnectJob: Job? = null
    private var libraryId = ""
    private var itemId = ""
    private var playbackEntries: List<VideoPlaybackEntry> = emptyList()
    private var activeIndex = 0
    private var deviceCredential: net.kdhmedia.mochil.auth.DeviceCredential? = null
    private lateinit var playerOverlay: VideoPlayerOverlay
    private lateinit var playerView: PlayerView
    private lateinit var scrubPreview: VideoScrubPreviewView
    private val scrubController by lazy {
        VideoScrubController(lifecycleScope, BoundedLruCache(maxEntries = 24)) { seconds, bitmap -> scrubPreview.showFrame(seconds, bitmap) }
    }
    private var closeRequested = false
    private val progressReporter by lazy {
        VideoProgressReporter(lifecycleScope, journal = deviceCredential?.let { videoProgressJournalOrNull(this, it) },
            onPersistenceFailure = { PerformanceDiagnostics.record("storage", "video_progress_save_failed") })
    }
    private val recovery = VideoPlaybackRecovery()
    private val seekOrigin = VideoSeekOrigin()

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        window.addFlags(android.view.WindowManager.LayoutParams.FLAG_KEEP_SCREEN_ON)
        PerformanceDiagnostics.initialize(applicationContext)
        enableEdgeToEdge()
        WindowCompat.getInsetsController(window, window.decorView).apply {
            systemBarsBehavior = WindowInsetsControllerCompat.BEHAVIOR_SHOW_TRANSIENT_BARS_BY_SWIPE
            hide(WindowInsetsCompat.Type.systemBars())
        }
        val localPlayback = intent.getBooleanExtra(EXTRA_LOCAL, false)
        val television = isTelevisionDevice(this)
        val credential = DeviceCredentialStore(this).load()
        deviceCredential = credential
        if (!localPlayback && credential == null) return finish()
        val url = intent.getStringExtra(EXTRA_URL).orEmpty()
        val fallbackUrl = intent.getStringExtra(EXTRA_FALLBACK_URL).orEmpty()
        val legacyEntry = VideoPlaybackEntry(
            libraryId = intent.getStringExtra(EXTRA_LIBRARY_ID).orEmpty(), itemId = intent.getStringExtra(EXTRA_ITEM_ID).orEmpty(),
            title = intent.getStringExtra(EXTRA_TITLE).orEmpty(), revision = intent.getStringExtra(EXTRA_REVISION).orEmpty(), url = url, fallbackUrl = fallbackUrl,
            positionMs = intent.getLongExtra(EXTRA_POSITION_MS, 0L).coerceAtLeast(0L),
            durationSeconds = intent.getDoubleExtra(EXTRA_DURATION_SECONDS, 0.0).takeIf(Double::isFinite)?.coerceAtLeast(0.0) ?: 0.0,
            subtitles = intent.getStringExtra(EXTRA_SUBTITLES).orEmpty(),
            progressRevision = intent.getLongExtra(EXTRA_PROGRESS_REVISION, 0L).coerceAtLeast(0L),
            finished = intent.getBooleanExtra(EXTRA_FINISHED, false),
        )
        val requestedQueue = decodeVideoPlaybackQueue(intent.getStringExtra(EXTRA_QUEUE).orEmpty())
            ?: VideoPlaybackQueue(listOf(legacyEntry), 0)
        val queue = if (!localPlayback && credential != null) progressReporter.restore(credential, requestedQueue) else requestedQueue
        if (!localPlayback && credential != null) progressReporter.replay(credential)
        playbackEntries = queue.entries
        activeIndex = queue.currentIndex
        libraryId = playbackEntries[activeIndex].libraryId
        itemId = playbackEntries[activeIndex].itemId
        recovery.transition(videoPlaybackIdentityKey(libraryId, itemId))
        if (libraryId.isBlank() || itemId.isBlank() || playbackEntries[activeIndex].url.isBlank()) return finish()

        val httpSource = DefaultHttpDataSource.Factory().apply { credential?.let { setDefaultRequestProperties(mapOf("Authorization" to "Bearer ${it.token}")) } }
        val dataSource = BoboMediaDataSourceFactory(this, httpSource)
        val exoPlayer = ExoPlayer.Builder(this).setMediaSourceFactory(DefaultMediaSourceFactory(this).setDataSourceFactory(dataSource)).build()
        player = exoPlayer
        playerOverlay = VideoPlayerOverlay(this, onClose = ::closePlayer, onRotate = ::toggleOrientation, onAudioTracks = ::showAudioTracks, onSubtitleTracks = ::showSubtitleTracks, onPictureInPicture = ::enterVideoPictureInPicture, title = playbackEntries[activeIndex].title, television = television)
        onBackPressedDispatcher.addCallback(this, object : OnBackPressedCallback(true) {
            override fun handleOnBackPressed() { if (!playerOverlay.dismissMenu()) closePlayer() }
        })
        exoPlayer.setMediaItems(playbackEntries.map { it.toVideoMediaItem(credential) }, activeIndex, queue.startPositionMs())
        exoPlayer.addListener(object : Player.Listener {
            override fun onPlayerError(error: PlaybackException) {
                val failure = classifyVideoPlaybackFailure(error.errorCode)
                PerformanceDiagnostics.record("playback", videoPlaybackDiagnosticName(failure), outcome = "error")
                val active = playbackEntries.getOrNull(exoPlayer.currentMediaItemIndex) ?: return
                val responseCode = generateSequence<Throwable>(error) { throwable -> throwable.cause }
                    .filterIsInstance<HttpDataSource.InvalidResponseCodeException>()
                    .firstOrNull()?.responseCode
                performRecovery(
                    recovery.onError(failure, responseCode, active.fallbackUrl.isNotBlank()),
                    exoPlayer,
                    active,
                    exoPlayer.currentMediaItemIndex,
                    exoPlayer.currentPosition.coerceAtLeast(0L),
                    "Connection interrupted. Reconnecting…",
                    "video_reconnect_retry",
                    videoPlaybackFailureMessage(failure),
                )
            }

            override fun onPlaybackStateChanged(playbackState: Int) {
                bufferingJob?.cancel()
                if (playbackState == Player.STATE_READY) {
                    reconnectJob?.cancel()
                    reconnectJob = null
                    recovery.ready()
                    playerOverlay.clearMessage()
                    PerformanceDiagnostics.record("playback", "video_ready")
                } else if (playbackState == Player.STATE_BUFFERING) {
                    bufferingJob = lifecycleScope.launch {
                        delay(12_000)
                        if (exoPlayer.playbackState == Player.STATE_BUFFERING) {
                            PerformanceDiagnostics.record("playback", "video_buffering_timeout", durationMs = 12_000, outcome = "timeout")
                            val index = exoPlayer.currentMediaItemIndex
                            val active = playbackEntries.getOrNull(index)
                            val position = exoPlayer.currentPosition.coerceAtLeast(0L)
                            performRecovery(
                                recovery.onStall(active?.fallbackUrl?.isNotBlank() == true),
                                exoPlayer,
                                active,
                                index,
                                position,
                                "Playback stalled. Reconnecting…",
                                "video_buffering_reconnect",
                                "This video is taking too long to load. Check the connection and try again.",
                            )
                        }
                    }
                } else if (playbackState == Player.STATE_ENDED) {
                    saveProgress()
                }
            }

            override fun onMediaItemTransition(mediaItem: MediaItem?, reason: Int) {
                hideScrubPreview()
                activeIndex = exoPlayer.currentMediaItemIndex.coerceIn(playbackEntries.indices)
                val active = playbackEntries[activeIndex]
                libraryId = active.libraryId
                itemId = active.itemId
                recovery.transition(mediaItem?.mediaId.orEmpty())
                reconnectJob?.cancel()
                reconnectJob = null
                if (this@VideoPlayerActivity::playerView.isInitialized) playerView.contentDescription = active.title.also(playerOverlay::setTitle)
                PerformanceDiagnostics.record("playback", "video_item_transition")
            }

            override fun onPositionDiscontinuity(oldPosition: Player.PositionInfo, newPosition: Player.PositionInfo, reason: Int) {
                videoProgressCheckpoint(oldPosition, newPosition, reason, playbackEntries, exoPlayer.duration, seekOrigin)
                    ?.let { saveProgress(it.entry, it.positionMs, it.durationMs, it.finished, it.explicitSeek) }
            }

            override fun onTracksChanged(tracks: androidx.media3.common.Tracks) {
                playerOverlay.setTrackAvailability(
                    audio = tracks.groups.any { it.type == C.TRACK_TYPE_AUDIO && it.isSupported },
                    subtitles = tracks.groups.any { it.type == C.TRACK_TYPE_TEXT && it.isSupported },
                )
            }

            override fun onIsPlayingChanged(isPlaying: Boolean) {
                if (android.os.Build.VERSION.SDK_INT >= 31 && this@VideoPlayerActivity::playerView.isInitialized) {
                    setPictureInPictureParams(pictureInPictureParams(autoEnter = isPlaying))
                }
            }
        })
        exoPlayer.prepare()
        exoPlayer.playWhenReady = true

        playerView = newPlayerView(this, television, exoPlayer, playerOverlay).apply {
            useController = true
            controllerAutoShow = !television
            controllerHideOnTouch = true
            controllerShowTimeoutMs = if (television) 3_000 else 4_000
            // Keep the primary transport surface uncluttered; caption preferences are applied separately.
            setShowSubtitleButton(false)
            contentDescription = playbackEntries[activeIndex].title
            layoutParams = ViewGroup.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.MATCH_PARENT)
        }
        applySubtitlePreferences(playerView, SubtitlePreferencesStore(this).load())
        playerView.setControllerVisibilityListener(PlayerView.ControllerVisibilityListener { visibility ->
            playerOverlay.setControlsVisible(visibility == android.view.View.VISIBLE)
        })
        scrubPreview = VideoScrubPreviewView(this)
        setContentView(FrameLayout(this).apply {
            addView(playerView, FrameLayout.LayoutParams(FrameLayout.LayoutParams.MATCH_PARENT, FrameLayout.LayoutParams.MATCH_PARENT))
            addView(scrubPreview, FrameLayout.LayoutParams(if (television) dp(260) else dp(190), if (television) dp(166) else dp(122), android.view.Gravity.BOTTOM or android.view.Gravity.CENTER_HORIZONTAL).apply {
                bottomMargin = if (television) dp(104) else dp(150)
            })
            addView(playerOverlay, FrameLayout.LayoutParams(FrameLayout.LayoutParams.MATCH_PARENT, FrameLayout.LayoutParams.MATCH_PARENT))
        })
        bindScrubPreview()
        playerOverlay.setControlsVisible(false)
        if (television) {
            playerView.hideController()
            // With the controls hidden the player view must own focus, or the remote's
            // select / left / right keys never reach it and nothing can be seeked.
            playerView.isFocusable = true
            playerView.isFocusableInTouchMode = true
            playerView.requestFocus()
        }
        progressJob = lifecycleScope.launch {
            while (isActive) {
                delay(15_000)
                val current = player
                if (shouldPersistPeriodicProgress(current?.isPlaying == true, current?.currentMediaItem != null)) saveProgress()
            }
        }
    }

    private fun applySubtitlePreferences(view: PlayerView, preferences: SubtitlePreferences) {
        view.subtitleView?.apply {
            setStyle(preferences.captionStyle())
            setFractionalTextSize(0.0533f * preferences.textScale)
            setApplyEmbeddedStyles(false)
            setApplyEmbeddedFontSizes(false)
        }
    }

    private fun toggleOrientation() {
        requestedOrientation = if (resources.configuration.orientation == Configuration.ORIENTATION_LANDSCAPE) {
            ActivityInfo.SCREEN_ORIENTATION_SENSOR_PORTRAIT
        } else {
            ActivityInfo.SCREEN_ORIENTATION_SENSOR_LANDSCAPE
        }
    }

    private fun showAudioTracks() {
        showTrackSelection("Audio track", C.TRACK_TYPE_AUDIO, allowDisable = false)
    }

    private fun performRecovery(
        action: VideoRecoveryAction,
        current: ExoPlayer,
        active: VideoPlaybackEntry?,
        index: Int,
        position: Long,
        retryMessage: String,
        retryDiagnostic: String,
        terminalMessage: String,
    ) {
        when (action) {
            is VideoRecoveryAction.Retry -> {
                reconnectJob?.cancel()
                playerOverlay.showMessage(retryMessage)
                PerformanceDiagnostics.record("playback", retryDiagnostic)
                reconnectJob = lifecycleScope.launch {
                    delay(videoReconnectDelayMillis(action.attempt))
                    if (closeRequested || player !== current || current.currentMediaItemIndex != index) return@launch
                    seekOrigin.expectAutomatic(index, position)
                    current.seekTo(index, position)
                    current.prepare()
                    current.playWhenReady = true
                }
            }
            VideoRecoveryAction.Fallback -> {
                if (active == null || active.fallbackUrl.isBlank()) {
                    playerOverlay.showMessage(terminalMessage, "Retry", ::retryPlayback)
                    return
                }
                PerformanceDiagnostics.record("playback", "video_fallback")
                playerOverlay.showMessage("Preparing a compatible stream…")
                current.replaceMediaItem(index, active.toVideoMediaItem(deviceCredential, active.fallbackUrl))
                seekOrigin.expectAutomatic(index, position)
                current.seekTo(index, position)
                current.prepare()
                current.playWhenReady = true
            }
            is VideoRecoveryAction.Terminal -> {
                val retryable = action.failure?.let(::offersManualVideoRetry) ?: true
                playerOverlay.showMessage(
                    terminalMessage,
                    actionLabel = "Retry".takeIf { retryable },
                    action = ::retryPlayback.takeIf { retryable },
                )
            }
        }
    }

    private fun retryPlayback() {
        val current = player ?: return
        val index = current.currentMediaItemIndex.coerceIn(playbackEntries.indices)
        val position = current.currentPosition.coerceAtLeast(0L)
        reconnectJob?.cancel()
        bufferingJob?.cancel()
        recovery.manualRetry()
        playerOverlay.showMessage("Reconnecting…")
        PerformanceDiagnostics.record("playback", "video_manual_retry")
        seekOrigin.expectAutomatic(index, position)
        current.seekTo(index, position)
        current.prepare()
        current.playWhenReady = true
    }

    private fun showSubtitleTracks() {
        showTrackSelection("Subtitle track", C.TRACK_TYPE_TEXT, allowDisable = true)
    }

    private fun showTrackSelection(title: String, type: Int, allowDisable: Boolean) {
        val current = player ?: return
        if (current.currentTracks.groups.none { it.type == type && it.isSupported }) {
            playerOverlay.showMessage("No selectable ${title.lowercase()}s were found.")
            return
        }
        TrackSelectionDialogBuilder(this, title, current, type)
            .setTheme(net.kdhmedia.mochil.R.style.Theme_Mochil_TrackDialog)
            .setAllowAdaptiveSelections(false)
            .setAllowMultipleOverrides(false)
            .setShowDisableOption(allowDisable)
            .build()
            .show()
    }

    private fun enterVideoPictureInPicture() {
        if (isInPictureInPictureMode) return
        runCatching { enterPictureInPictureMode(pictureInPictureParams(autoEnter = player?.isPlaying == true)) }
    }

    private fun pictureInPictureParams(autoEnter: Boolean): PictureInPictureParams {
        val size = player?.videoSize
        val ratio = videoPictureInPictureAspectRatio(size?.width ?: 0, size?.height ?: 0)
        return PictureInPictureParams.Builder().setAspectRatio(Rational((ratio * 1_000).toInt(), 1_000)).apply {
            if (this@VideoPlayerActivity::playerView.isInitialized) {
                val source = Rect()
                if (playerView.getGlobalVisibleRect(source) && !source.isEmpty) setSourceRectHint(source)
            }
            if (android.os.Build.VERSION.SDK_INT >= 31) {
                setAutoEnterEnabled(autoEnter)
                setSeamlessResizeEnabled(true)
            }
        }.build()
    }

    override fun onPause() {
        saveProgress()
        super.onPause()
    }

    override fun onUserLeaveHint() {
        super.onUserLeaveHint()
        if (shouldEnterVideoPictureInPicture(closeRequested, player?.isPlaying == true)) enterVideoPictureInPicture()
    }

    override fun onPictureInPictureModeChanged(isInPictureInPictureMode: Boolean, newConfig: Configuration) {
        super.onPictureInPictureModeChanged(isInPictureInPictureMode, newConfig)
        playerOverlay.visibility = if (isInPictureInPictureMode) android.view.View.GONE else android.view.View.VISIBLE
        if (isInPictureInPictureMode) playerView.hideController()
    }

    private fun saveProgress() {
        val current = player ?: return
        val entry = playbackEntries.getOrNull(current.currentMediaItemIndex) ?: return
        val position = current.currentPosition.coerceAtLeast(0L)
        val duration = current.duration.takeIf { it != C.TIME_UNSET && it > 0 } ?: return
        val finished = current.playbackState == Player.STATE_ENDED
        saveProgress(entry, position, duration, finished)
    }

    private fun saveProgress(entry: VideoPlaybackEntry, position: Long, duration: Long, finished: Boolean, explicitSeek: Boolean = false) {
        if (entry.libraryId == "local" || entry.libraryId.startsWith("webdav:")) return
        val credential = deviceCredential ?: return
        progressReporter.submit(credential, entry, position, duration, finished, explicitSeek)
    }

    private fun closePlayer() {
        if (closeRequested) return
        closeRequested = true
        PerformanceDiagnostics.record("playback", "video_close")
        saveProgress()
        if (android.os.Build.VERSION.SDK_INT >= 31 && this::playerView.isInitialized) {
            setPictureInPictureParams(pictureInPictureParams(autoEnter = false))
        }
        releasePlayer()
        finish()
    }

    private fun releasePlayer() {
        hideScrubPreview()
        progressJob?.cancel()
        progressJob = null
        bufferingJob?.cancel()
        bufferingJob = null
        reconnectJob?.cancel()
        reconnectJob = null
        seekOrigin.clear()
        if (this::playerView.isInitialized) playerView.player = null
        player?.run { pause(); stop(); release() }
        player = null
    }

    private fun bindScrubPreview() {
        val timeBar = playerView.findViewById<android.view.View>(androidx.media3.ui.R.id.exo_progress) as? TimeBar ?: return
        timeBar.addListener(object : TimeBar.OnScrubListener {
            override fun onScrubStart(timeBar: TimeBar, position: Long) = requestScrubPreview(position)
            override fun onScrubMove(timeBar: TimeBar, position: Long) = requestScrubPreview(position)
            override fun onScrubStop(timeBar: TimeBar, position: Long, canceled: Boolean) = hideScrubPreview()
        })
    }

    private fun requestScrubPreview(positionMs: Long) {
        val current = player ?: return
        val entry = playbackEntries.getOrNull(current.currentMediaItemIndex) ?: return
        val durationSeconds = current.duration.takeIf { it != C.TIME_UNSET && it > 0 }?.div(1_000.0) ?: entry.durationSeconds
        val requestedIndex = current.currentMediaItemIndex
        val seconds = scrubController.request(entry, positionMs, durationSeconds, deviceCredential) { player?.currentMediaItemIndex == requestedIndex }
            ?: return hideScrubPreview()
        scrubPreview.showPosition((positionMs / 1_000L).toInt())
    }

    private fun hideScrubPreview() {
        scrubController.cancel()
        if (this::scrubPreview.isInitialized) scrubPreview.hidePreview()
    }

    private fun dp(value: Int) = (value * resources.displayMetrics.density).toInt()

    override fun onDestroy() {
        releasePlayer()
        super.onDestroy()
    }

    companion object {
        const val EXTRA_LIBRARY_ID = "library_id"
        const val EXTRA_ITEM_ID = "item_id"
        const val EXTRA_TITLE = "title"
        const val EXTRA_URL = "url"
        const val EXTRA_FALLBACK_URL = "fallback_url"
        const val EXTRA_POSITION_MS = "position_ms"
        const val EXTRA_DURATION_SECONDS = "duration_seconds"
        const val EXTRA_SUBTITLES = "subtitles"
        const val EXTRA_REVISION = "revision"
        const val EXTRA_PROGRESS_REVISION = "progress_revision"
        const val EXTRA_FINISHED = "finished"
        const val EXTRA_QUEUE = "playback_queue"
        const val EXTRA_LOCAL = "local_playback"
    }
}

internal fun shouldEnterVideoPictureInPicture(closeRequested: Boolean, isPlaying: Boolean): Boolean = !closeRequested && isPlaying
