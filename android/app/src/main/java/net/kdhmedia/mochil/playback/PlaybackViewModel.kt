package net.kdhmedia.mochil.playback

import android.app.Application
import android.content.ComponentName
import androidx.core.content.ContextCompat
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import androidx.media3.common.MediaItem
import androidx.media3.common.Player
import androidx.media3.common.PlaybackException
import androidx.media3.session.MediaController
import androidx.media3.session.SessionToken
import java.util.UUID
import kotlinx.coroutines.delay
import kotlinx.coroutines.Job
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.distinctUntilChanged
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.flow.stateIn
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch
import net.kdhmedia.mochil.audio.AudioTrack
import net.kdhmedia.mochil.audio.audioTrackIdentityKey
import net.kdhmedia.mochil.audio.identityKey

data class PlaybackState(
    val connected: Boolean = false,
    val current: AudioTrack? = null,
    val isPlaying: Boolean = false,
    val positionMs: Long = 0,
    val durationMs: Long = 0,
    val repeatMode: Int = Player.REPEAT_MODE_OFF,
    val shuffleEnabled: Boolean = false,
    val replayGainMode: ReplayGainMode = ReplayGainMode.Off,
    val speed: Float = 1f,
    val sleepRemainingMs: Long = 0,
    val audiobookBookmarks: List<AudiobookBookmark> = emptyList(),
)

/** Stable TV dock chrome; intentionally excludes the once-per-second clock. */
data class PlaybackDockState(
    val current: AudioTrack? = null,
    val isPlaying: Boolean = false,
)

data class PlaybackTimelineState(
    val positionMs: Long = 0,
    val durationMs: Long = 0,
)

data class PlaybackQueueState(
    val tracks: List<AudioTrack> = emptyList(),
    val currentIndex: Int = -1,
)

data class PlaybackTransportState(
    val current: AudioTrack? = null,
    val isPlaying: Boolean = false,
    val repeatMode: Int = Player.REPEAT_MODE_OFF,
    val shuffleEnabled: Boolean = false,
    val speed: Float = 1f,
)

data class PlaybackPlayerState(
    val current: AudioTrack? = null,
    val isPlaying: Boolean = false,
    val repeatMode: Int = Player.REPEAT_MODE_OFF,
    val shuffleEnabled: Boolean = false,
    val speed: Float = 1f,
    val sleepRemainingMs: Long = 0,
    val audiobookBookmarks: List<AudiobookBookmark> = emptyList(),
)

internal fun PlaybackState.toDockState() = PlaybackDockState(current, isPlaying)
internal fun PlaybackState.toTimelineState() = PlaybackTimelineState(positionMs, durationMs)
internal fun PlaybackState.toTransportState() = PlaybackTransportState(current, isPlaying, repeatMode, shuffleEnabled, speed)
internal fun PlaybackState.toPlayerState() = PlaybackPlayerState(current, isPlaying, repeatMode, shuffleEnabled, speed, sleepRemainingMs, audiobookBookmarks)

class PlaybackViewModel(application: Application) : AndroidViewModel(application), Player.Listener {
    private val future = MediaController.Builder(application, SessionToken(application, ComponentName(application, PlaybackService::class.java))).buildAsync()
    private var controller: MediaController? = null
    private var queueByKey: Map<String, AudioTrack> = emptyMap()
    private val replayGainPreferences = ReplayGainPreferences(application)
    private val sleepTimer = SleepTimerStore(application)
    private val audiobookBookmarks = AudiobookBookmarkStore(application)
    private val playbackSnapshot = PlaybackSnapshotStore(application)
    private var bookmarkTrackKey = ""
    private var bookmarkSnapshot = emptyList<AudiobookBookmark>()
    private val mutableState = MutableStateFlow(PlaybackState(replayGainMode = replayGainPreferences.mode()))
    private val mutableQueueState = MutableStateFlow(PlaybackQueueState())
    val state: StateFlow<PlaybackState> = mutableState.asStateFlow()
    val queueState: StateFlow<PlaybackQueueState> = mutableQueueState.asStateFlow()
    val replayGainMode: StateFlow<ReplayGainMode> = state
        .map { playback -> playback.replayGainMode }
        .distinctUntilChanged()
        .stateIn(viewModelScope, PLAYBACK_SHARING, replayGainPreferences.mode())
    val dockState: StateFlow<PlaybackDockState> = state
        .map(PlaybackState::toDockState)
        .distinctUntilChanged()
        .stateIn(viewModelScope, PLAYBACK_SHARING, mutableState.value.toDockState())
    val timelineState: StateFlow<PlaybackTimelineState> = state
        .map(PlaybackState::toTimelineState)
        .distinctUntilChanged()
        .stateIn(viewModelScope, PLAYBACK_SHARING, mutableState.value.toTimelineState())
    val transportState: StateFlow<PlaybackTransportState> = state
        .map(PlaybackState::toTransportState)
        .distinctUntilChanged()
        .stateIn(viewModelScope, PLAYBACK_SHARING, mutableState.value.toTransportState())
    val currentTrack: StateFlow<AudioTrack?> = state
        .map { playback -> playback.current }
        .distinctUntilChanged()
        .stateIn(viewModelScope, PLAYBACK_SHARING, mutableState.value.current)
    val playerState: StateFlow<PlaybackPlayerState> = state
        .map(PlaybackState::toPlayerState)
        .distinctUntilChanged()
        .stateIn(viewModelScope, PLAYBACK_SHARING, mutableState.value.toPlayerState())

    private var positionTicker: Job? = null
    // Each queue-mutating action below resolves its stream URL(s) on a coroutine before touching
    // controller/queueByKey; this mutex keeps a rapid double-tap (play/playNext/addToQueue fired in
    // quick succession) applying in the order the user triggered it, instead of two independently
    // suspended coroutines racing to mutate the same player/queue state out of order.
    private val queueMutationMutex = Mutex()

    init {
        future.addListener({
            runCatching { future.get() }.onSuccess {
                controller = it
                it.addListener(this)
                publishQueue(it)
                publish(it)
                updatePositionTicker(it)
            }
        }, ContextCompat.getMainExecutor(application))
    }

    fun play(tracks: List<AudioTrack>, selected: AudioTrack, streamUrl: suspend (AudioTrack) -> String, requestedStartPositionMs: Long = 0) {
        val player = controller ?: return
        val startPositionMs = playbackStartPositionMs(selected, requestedStartPositionMs)
        val boundedTracks = playbackQueueForSelection(tracks, selected)
        val selectedIndex = boundedTracks.indexOfFirst { it.playbackKey() == selected.playbackKey() }.coerceAtLeast(0)
        val automaticToken = if (startPositionMs > 0) UUID.randomUUID().toString() else ""
        viewModelScope.launch {
            queueMutationMutex.withLock {
                val items = boundedTracks.mapIndexed { index, track ->
                    track.toPlaybackMediaItem(
                        streamUrl(track),
                        automaticStartPositionMs = startPositionMs.takeIf { index == selectedIndex && it > 0 },
                        automaticSeekToken = automaticToken,
                    )
                }
                queueByKey = boundedTracks.associateBy(AudioTrack::playbackKey)
                player.setMediaItems(items, selectedIndex, startPositionMs.coerceAtLeast(0))
                publishQueue(player)
                player.prepare()
                player.play()
            }
        }
    }

    fun toggle() {
        controller?.let { if (it.isPlaying) it.pause() else it.play() }
    }

    fun pause() = controller?.pause()

    fun closePlayback() {
        controller?.let { player ->
            player.stop()
            player.clearMediaItems()
        }
        playbackSnapshot.clear(durable = true)
        queueByKey = emptyMap()
        mutableQueueState.value = PlaybackQueueState()
        mutableState.value = PlaybackState(connected = controller != null)
    }

    fun playNext(track: AudioTrack, streamUrl: suspend (AudioTrack) -> String) {
        val player = controller ?: return
        viewModelScope.launch {
            queueMutationMutex.withLock {
                val item = track.toPlaybackMediaItem(streamUrl(track))
                removeQueuedCopy(player, track)
                queueByKey = queueByKey + (track.playbackKey() to track)
                val target = if (player.currentMediaItemIndex >= 0) player.currentMediaItemIndex + 1 else 0
                player.addMediaItem(target.coerceAtMost(player.mediaItemCount), item)
                publishQueue(player)
            }
        }
    }

    fun addToQueue(track: AudioTrack, streamUrl: suspend (AudioTrack) -> String) {
        val player = controller ?: return
        viewModelScope.launch {
            queueMutationMutex.withLock {
                val item = track.toPlaybackMediaItem(streamUrl(track))
                removeQueuedCopy(player, track)
                queueByKey = queueByKey + (track.playbackKey() to track)
                player.addMediaItem(item)
                publishQueue(player)
            }
        }
    }

    fun playNext(tracks: List<AudioTrack>, streamUrl: suspend (AudioTrack) -> String) = insertTracks(tracks, true, streamUrl)
    fun addToQueue(tracks: List<AudioTrack>, streamUrl: suspend (AudioTrack) -> String) = insertTracks(tracks, false, streamUrl)

    private fun insertTracks(tracks: List<AudioTrack>, next: Boolean, streamUrl: suspend (AudioTrack) -> String) {
        val player = controller ?: return
        val distinct = tracks.distinctBy(AudioTrack::playbackKey).take(PlaybackSnapshotStore.MAX_ITEMS)
        if (distinct.isEmpty()) return
        viewModelScope.launch {
            queueMutationMutex.withLock {
                // Resolve every item before touching the current queue; preparation may fail.
                val items = distinct.map { it.toPlaybackMediaItem(streamUrl(it)) }
                removeQueuedCopies(player, distinct.mapTo(mutableSetOf(), AudioTrack::playbackKey))
                queueByKey = queueByKey + distinct.associateBy(AudioTrack::playbackKey)
                val target = if (next && player.currentMediaItemIndex >= 0) player.currentMediaItemIndex + 1 else player.mediaItemCount
                player.addMediaItems(target.coerceAtMost(player.mediaItemCount), items)
                publishQueue(player)
            }
        }
    }

    fun playQueueItem(index: Int) {
        val player = controller ?: return
        if (index !in 0 until player.mediaItemCount) return
        player.seekToDefaultPosition(index)
        player.play()
    }

    fun playQueueItem(track: AudioTrack) = playQueueItem(currentQueueIndex(track))
    fun removeQueueItem(track: AudioTrack): List<AudioTrack> = removeQueueItem(currentQueueIndex(track))
    fun moveQueueItem(track: AudioTrack, direction: Int): List<AudioTrack> = moveQueueItem(currentQueueIndex(track), direction)

    private fun currentQueueIndex(track: AudioTrack): Int {
        val player = controller ?: return -1
        return playbackQueueIndex(player.mediaItemCount, track.identityKey) { index ->
            val item = player.getMediaItemAt(index)
            audioTrackIdentityKey(item.mediaMetadata.extras?.getString("library_id").orEmpty(), item.mediaId)
        }
    }

    fun removeQueueItem(index: Int): List<AudioTrack> {
        val player = controller ?: return mutableQueueState.value.tracks
        if (index !in 0 until player.mediaItemCount || player.mediaItemCount <= 1) return mutableQueueState.value.tracks
        player.removeMediaItem(index)
        return publishQueue(player).tracks
    }

    fun moveQueueItem(index: Int, direction: Int): List<AudioTrack> {
        val player = controller ?: return mutableQueueState.value.tracks
        if (index !in 0 until player.mediaItemCount) return mutableQueueState.value.tracks
        val target = (index + direction).coerceIn(0, player.mediaItemCount - 1)
        if (target != index) player.moveMediaItem(index, target)
        return publishQueue(player).tracks
    }

    fun next() = controller?.seekToNextMediaItem()
    fun previous() = controller?.seekToPreviousMediaItem()
    fun seekTo(positionMs: Long) = controller?.let { it.seekTo(positionMs.coerceAtLeast(0)); publish(it) }
    fun toggleRepeat() = controller?.let { player ->
        player.repeatMode = when (player.repeatMode) {
            Player.REPEAT_MODE_OFF -> Player.REPEAT_MODE_ALL
            Player.REPEAT_MODE_ALL -> Player.REPEAT_MODE_ONE
            else -> Player.REPEAT_MODE_OFF
        }
    }
    fun toggleShuffle() = controller?.let { it.shuffleModeEnabled = !it.shuffleModeEnabled }
    fun cycleSpeed() = controller?.let { it.setPlaybackSpeed(nextSpokenPlaybackSpeed(it.playbackParameters.speed)) }
    fun setSleepTimer(minutes: Int?) {
        if (minutes == null) sleepTimer.clear() else sleepTimer.set(minutes)
        controller?.let(::publish)
    }
    fun setReplayGainMode(mode: ReplayGainMode) {
        replayGainPreferences.setMode(mode)
        mutableState.value = mutableState.value.copy(replayGainMode = mode)
    }

    fun addAudiobookBookmark() {
        val player = controller ?: return
        val track = mutableState.value.current?.takeIf { it.libraryKind == "audiobooks" } ?: return
        bookmarkSnapshot = audiobookBookmarks.add(track, player.currentPosition.coerceAtLeast(0))
        bookmarkTrackKey = track.playbackKey()
        publish(player)
    }

    fun removeAudiobookBookmark(bookmarkId: String) {
        val player = controller ?: return
        val track = mutableState.value.current?.takeIf { it.libraryKind == "audiobooks" } ?: return
        bookmarkSnapshot = audiobookBookmarks.remove(track, bookmarkId)
        bookmarkTrackKey = track.playbackKey()
        publish(player)
    }

    override fun onIsPlayingChanged(isPlaying: Boolean) { controller?.let { publish(it); updatePositionTicker(it) } }
    override fun onMediaItemTransition(mediaItem: MediaItem?, reason: Int) { controller?.let { publishQueue(it); publish(it) } }
    override fun onTimelineChanged(timeline: androidx.media3.common.Timeline, reason: Int) { controller?.let(::publishQueue) }
    override fun onPlaybackStateChanged(playbackState: Int) { controller?.let(::publish) }
    override fun onRepeatModeChanged(repeatMode: Int) { controller?.let(::publish) }
    override fun onShuffleModeEnabledChanged(shuffleModeEnabled: Boolean) { controller?.let(::publish) }
    override fun onPlaybackParametersChanged(playbackParameters: androidx.media3.common.PlaybackParameters) { controller?.let(::publish) }
    override fun onPlayerError(error: PlaybackException) {
        controller?.let(::publish)
    }

    private fun publish(player: Player) {
        val duration = player.duration.takeIf { it > 0 } ?: 0
        val item = player.currentMediaItem
        val current = item?.let { mediaItem ->
            queueByKey[audioTrackIdentityKey(mediaItem.mediaMetadata.extras?.getString("library_id").orEmpty(), mediaItem.mediaId)]
                ?: mediaItem.toPlaybackTrack(duration)
        }
        if (current != null && current.libraryKind == "audiobooks") {
            val key = current.playbackKey()
            if (bookmarkTrackKey != key) {
                bookmarkTrackKey = key
                bookmarkSnapshot = audiobookBookmarks.list(current)
            }
        } else {
            bookmarkTrackKey = ""
            bookmarkSnapshot = emptyList()
        }
        mutableState.value = PlaybackState(
            connected = true,
            current = current,
            isPlaying = player.isPlaying,
            positionMs = player.currentPosition.coerceAtLeast(0),
            durationMs = duration,
            repeatMode = player.repeatMode,
            shuffleEnabled = player.shuffleModeEnabled,
            replayGainMode = replayGainPreferences.mode(),
            speed = player.playbackParameters.speed,
            sleepRemainingMs = sleepTimer.remainingMillis(),
            audiobookBookmarks = bookmarkSnapshot,
        )
    }

    private fun updatePositionTicker(player: Player) {
        positionTicker?.cancel()
        positionTicker = if (player.isPlaying) viewModelScope.launch {
            while (isActive) {
                delay(1_000)
                publish(player)
            }
        } else null
    }

    private fun publishQueue(player: Player): PlaybackQueueState {
        val tracks = (0 until player.mediaItemCount).map { index ->
            val mediaItem = player.getMediaItemAt(index)
            queueByKey[audioTrackIdentityKey(mediaItem.mediaMetadata.extras?.getString("library_id").orEmpty(), mediaItem.mediaId)]
                ?: mediaItem.toPlaybackTrack()
        }
        // Removed media must not retain its artwork/metadata for the ViewModel lifetime.
        queueByKey = tracks.associateBy(AudioTrack::playbackKey)
        return PlaybackQueueState(tracks, player.currentMediaItemIndex.takeIf { it in tracks.indices } ?: -1)
            .also { mutableQueueState.value = it }
    }

    override fun onCleared() {
        positionTicker?.cancel()
        controller?.removeListener(this)
        MediaController.releaseFuture(future)
    }

    private fun removeQueuedCopy(player: Player, track: AudioTrack) {
        removeQueuedCopies(player, setOf(track.playbackKey()))
    }

    private fun removeQueuedCopies(player: Player, additions: Set<String>) {
        val ranges = queuedCopyRemovalRanges(player.mediaItemCount, player.currentMediaItemIndex, additions) { index ->
            val item = player.getMediaItemAt(index)
            audioTrackIdentityKey(item.mediaMetadata.extras?.getString("library_id").orEmpty(), item.mediaId)
        }
        ranges.forEach { player.removeMediaItems(it.start, it.endExclusive) }
    }
}

private val PLAYBACK_SHARING = SharingStarted.WhileSubscribed(stopTimeoutMillis = 5_000)

private fun AudioTrack.playbackKey() = identityKey

internal fun boundedPlaybackQueue(tracks: List<AudioTrack>, selectedIndex: Int): List<AudioTrack> {
    if (tracks.size <= PlaybackSnapshotStore.MAX_ITEMS) return tracks
    val current = selectedIndex.coerceIn(tracks.indices)
    val first = (current - PlaybackSnapshotStore.MAX_ITEMS / 2).coerceAtLeast(0)
        .coerceAtMost(tracks.size - PlaybackSnapshotStore.MAX_ITEMS)
    return tracks.subList(first, first + PlaybackSnapshotStore.MAX_ITEMS)
}

/** Queue selection follows durable media identity, not mutable catalogue metadata. */
/** Audiobooks and podcast episodes continue where you stopped; a song always starts from the beginning. */
internal fun playbackStartPositionMs(track: AudioTrack, requestedMs: Long): Long =
    if (track.libraryKind == "music") 0 else requestedMs.coerceAtLeast(0)

internal fun playbackQueueForSelection(tracks: List<AudioTrack>, selected: AudioTrack): List<AudioTrack> {
    val selectedIndex = tracks.indexOfFirst { it.playbackKey() == selected.playbackKey() }
    return if (selectedIndex < 0) listOf(selected) else boundedPlaybackQueue(tracks, selectedIndex)
}
