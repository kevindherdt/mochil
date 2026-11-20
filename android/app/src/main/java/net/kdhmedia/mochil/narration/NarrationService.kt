package net.kdhmedia.mochil.narration

import android.os.Bundle
import androidx.media3.common.AudioAttributes
import androidx.media3.common.C
import androidx.media3.common.MediaItem
import androidx.media3.common.MediaMetadata
import androidx.media3.common.PlaybackException
import androidx.media3.common.Player
import androidx.media3.common.util.UnstableApi
import androidx.media3.datasource.DefaultHttpDataSource
import androidx.media3.exoplayer.ExoPlayer
import androidx.media3.exoplayer.source.DefaultMediaSourceFactory
import androidx.media3.session.MediaSession
import androidx.media3.session.MediaSessionService
import androidx.media3.session.SessionCommand
import androidx.media3.session.SessionError
import androidx.media3.session.SessionResult
import com.google.common.util.concurrent.Futures
import com.google.common.util.concurrent.ListenableFuture
import com.google.common.util.concurrent.SettableFuture
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.delay
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch
import net.kdhmedia.mochil.auth.DeviceCredential
import net.kdhmedia.mochil.auth.DeviceCredentialStore
import net.kdhmedia.mochil.reader.epub.EpubLocatorStore
import org.json.JSONObject

/**
 * Reads a book aloud in the background, with its own player so it never touches music, audiobooks
 * or TV playback. It queues one media item per sentence (the server makes the audio), adds the next
 * chapter as the queue runs low, and pauses by itself when a sleep timer says so. The reader
 * follows along through the media session and only draws the highlight.
 */
@androidx.annotation.OptIn(markerClass = [UnstableApi::class])
class NarrationService : MediaSessionService() {
    private var session: MediaSession? = null
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Main.immediate)
    private val client = NarrationClient()
    private var book: BookQueue? = null
    private var queueing: Job? = null
    private var sleep: NarrationSleep = NarrationSleep.Off
    private var sleepDeadline = 0L
    private var sleepChapter = -1
    private var failures = 0
    private lateinit var locatorStore: EpubLocatorStore

    private class BookQueue(
        val credential: DeviceCredential, val libraryId: String, val bookId: String, val title: String,
        val voice: String, val progressKey: String, val chapters: List<NarrationChapter>,
        var lastChapter: Int, var finished: Boolean = false,
    )

    override fun onCreate() {
        super.onCreate()
        locatorStore = EpubLocatorStore(this)
        val token = DeviceCredentialStore(this).load()?.token
        val http = DefaultHttpDataSource.Factory().apply {
            if (token != null) setDefaultRequestProperties(mapOf("Authorization" to "Bearer $token"))
        }
        val player = ExoPlayer.Builder(this)
            .setMediaSourceFactory(DefaultMediaSourceFactory(http))
            .setAudioAttributes(AudioAttributes.Builder().setUsage(C.USAGE_MEDIA).setContentType(C.AUDIO_CONTENT_TYPE_SPEECH).build(), true)
            .setHandleAudioBecomingNoisy(true)
            .build()
        // The music player has a session in the same process, so this one needs an id of its own.
        // Failing to make it must never take the app down: narration just does not start.
        session = runCatching { MediaSession.Builder(this, player).setId(SESSION_ID).setCallback(SessionCallback()).build() }.getOrNull()
        if (session == null) {
            player.release()
            stopSelf()
            return
        }
        player.addListener(object : Player.Listener {
            override fun onMediaItemTransition(mediaItem: MediaItem?, reason: Int) {
                val extras = mediaItem?.mediaMetadata?.extras ?: return
                rememberPosition(extras)
                if (sleep == NarrationSleep.EndOfChapter && sleepChapter >= 0 && extras.getInt(CHAPTER) != sleepChapter) {
                    sleep = NarrationSleep.Off
                    player.pause()
                }
                queueMoreIfLow()
            }
            override fun onIsPlayingChanged(isPlaying: Boolean) { if (isPlaying) failures = 0 }
            override fun onPlaybackStateChanged(playbackState: Int) {
                if (playbackState == Player.STATE_IDLE || playbackState == Player.STATE_ENDED) NarrationPreferences(this@NarrationService).saveActiveBook("")
            }
            override fun onPlayerError(error: PlaybackException) {
                // One sentence that cannot be made is skipped; a server that is gone ends narration.
                net.kdhmedia.mochil.network.httpStatusFromPlaybackError(error)?.let { net.kdhmedia.mochil.network.reportIfExpiredDeviceSession(it) }
                if (++failures > MAX_FAILURES) { player.stop(); return }
                player.seekToNextMediaItem()
                player.prepare()
                player.play()
            }
        })
        scope.launch {
            while (isActive) {
                delay(500)
                tickSleep(player)
            }
        }
    }

    override fun onGetSession(controllerInfo: MediaSession.ControllerInfo): MediaSession? = session

    override fun onTaskRemoved(rootIntent: android.content.Intent?) {
        session?.player?.run { stop(); clearMediaItems() }
        stopSelf()
        super.onTaskRemoved(rootIntent)
    }

    override fun onDestroy() {
        NarrationPreferences(this).saveActiveBook("")
        scope.cancel()
        session?.run { player.release(); release() }
        session = null
        super.onDestroy()
    }

    private inner class SessionCallback : MediaSession.Callback {
        override fun onConnect(session: MediaSession, controller: MediaSession.ControllerInfo): MediaSession.ConnectionResult {
            return MediaSession.ConnectionResult.accept(
                MediaSession.ConnectionResult.DEFAULT_SESSION_COMMANDS.buildUpon().add(START).add(START_AUDIO).add(SLEEP).build(),
                MediaSession.ConnectionResult.DEFAULT_PLAYER_COMMANDS,
            )
        }

        override fun onCustomCommand(session: MediaSession, controller: MediaSession.ControllerInfo, customCommand: SessionCommand, args: Bundle): ListenableFuture<SessionResult> =
            when (customCommand.customAction) {
                ACTION_START -> startReading(args)
                ACTION_START_AUDIO -> startRecording(args)
                ACTION_SLEEP -> { setSleep(args); Futures.immediateFuture(SessionResult(SessionResult.RESULT_SUCCESS)) }
                else -> Futures.immediateFuture(SessionResult(SessionError.ERROR_NOT_SUPPORTED))
            }
    }

    private fun startReading(args: Bundle): ListenableFuture<SessionResult> {
        val done = SettableFuture.create<SessionResult>()
        val credential = DeviceCredentialStore(this).load()
        if (credential == null) {
            done.set(SessionResult(SessionError.ERROR_PERMISSION_DENIED))
            return done
        }
        queueing?.cancel()
        scope.launch {
            val code = runCatching { begin(credential, args) }.getOrElse { SessionError.ERROR_UNKNOWN }
            done.set(SessionResult(code))
        }
        return done
    }

    /**
     * Plays a book's own recorded narration (a read-along EPUB): one item per audio file, which the
     * server streams out of the book. The reader works out which sentence is speaking from the
     * position, so this only has to play, keep going with the screen off and honour the sleep timer.
     */
    private fun startRecording(args: Bundle): ListenableFuture<SessionResult> {
        val player = session?.player ?: return Futures.immediateFuture(SessionResult(SessionError.ERROR_UNKNOWN))
        val urls = args.getStringArrayList(URLS).orEmpty()
        if (urls.isEmpty()) return Futures.immediateFuture(SessionResult(SessionError.ERROR_BAD_VALUE))
        queueing?.cancel()
        book = null
        val libraryId = args.getString(LIBRARY).orEmpty()
        val bookId = args.getString(BOOK).orEmpty()
        val items = urls.mapIndexed { i, url ->
            MediaItem.Builder()
                .setMediaId("recording:$i")
                .setUri(url)
                .setMediaMetadata(
                    MediaMetadata.Builder()
                        .setTitle(args.getString(TITLE).orEmpty())
                        .setExtras(Bundle().apply { putString(BOOK, bookId); putString(LIBRARY, libraryId) })
                        .build(),
                )
                .build()
        }
        NarrationPreferences(this).saveActiveBook("$libraryId/$bookId")
        sleep = NarrationSleep.Off
        player.volume = 1f
        player.setMediaItems(items, args.getInt(ITEM, 0).coerceIn(0, items.lastIndex), args.getLong(POSITION_MS, 0L))
        player.setPlaybackSpeed(args.getFloat(SPEED, 1f))
        setSleep(args)
        player.prepare()
        player.play()
        return Futures.immediateFuture(SessionResult(SessionResult.RESULT_SUCCESS))
    }

    private suspend fun begin(credential: DeviceCredential, args: Bundle): Int {
        val player = session?.player ?: return SessionError.ERROR_UNKNOWN
        val libraryId = args.getString(LIBRARY).orEmpty()
        val bookId = args.getString(BOOK).orEmpty()
        val chapters = client.chapters(credential, libraryId, bookId)
        val first = NarrationPlan.chapterFor(args.getString(HREF).orEmpty(), chapters)
            ?: chapters.firstOrNull { !it.matter } ?: return SessionError.ERROR_BAD_VALUE
        val queue = BookQueue(
            credential, libraryId, bookId, args.getString(TITLE).orEmpty(), args.getString(VOICE).orEmpty(),
            args.getString(PROGRESS_KEY).orEmpty(), chapters, lastChapter = first.index - 1,
        )
        var items = emptyList<MediaItem>()
        var chapter: NarrationChapter? = first
        var progression = args.getDouble(PROGRESSION, 0.0)
        while (items.isEmpty() && chapter != null) {
            val sentences = client.sentences(credential, libraryId, bookId, chapter.index)
            queue.lastChapter = chapter.index
            val from = NarrationPlan.startSentence(sentences, progression)
            items = sentences.drop(from).mapIndexed { i, s -> item(queue, chapter, from + i, s) }
            // A page with no text (a picture) moves on to the next chapter.
            chapter = if (items.isEmpty()) NarrationPlan.nextChapter(chapters, chapter.index) else null
            progression = 0.0
        }
        if (items.isEmpty()) return SessionError.ERROR_BAD_VALUE
        book = queue
        NarrationPreferences(this).saveActiveBook("$libraryId/$bookId")
        sleep = NarrationSleep.Off
        player.volume = 1f
        player.setMediaItems(items, 0, 0)
        player.setPlaybackSpeed(args.getFloat(SPEED, 1f))
        setSleep(args) // a timer chosen before pressing play counts from here
        player.prepare()
        player.play()
        queueMoreIfLow()
        return SessionResult.RESULT_SUCCESS
    }

    private fun item(queue: BookQueue, chapter: NarrationChapter, index: Int, sentence: NarrationSentence): MediaItem {
        val extras = Bundle().apply {
            putString(BOOK, queue.bookId); putString(LIBRARY, queue.libraryId); putString(PROGRESS_KEY, queue.progressKey)
            putString(HREF, chapter.href); putInt(CHAPTER, chapter.index); putInt(INDEX, index)
            putDouble(PROGRESSION, sentence.progression)
            putString(BEFORE, sentence.before); putString(TEXT, sentence.text); putString(AFTER, sentence.after)
        }
        return MediaItem.Builder()
            .setMediaId("${chapter.index}:$index")
            .setUri(NarrationClient.audioUrl(queue.credential, queue.libraryId, queue.bookId, chapter.index, index, queue.voice))
            .setMediaMetadata(
                MediaMetadata.Builder()
                    .setTitle(chapter.title.ifBlank { "Chapter ${chapter.index + 1}" })
                    .setArtist(queue.title)
                    .setExtras(extras)
                    .build(),
            )
            .build()
    }

    /** Adds the next chapter once few sentences are left, so there is never a wait at a chapter's end. */
    private fun queueMoreIfLow() {
        val player = session?.player ?: return
        val queue = book ?: return
        if (queue.finished || queueing?.isActive == true) return
        if (player.mediaItemCount - player.currentMediaItemIndex > NarrationPlan.QUEUE_AHEAD) return
        queueing = scope.launch {
            var chapter = NarrationPlan.nextChapter(queue.chapters, queue.lastChapter)
            while (chapter != null) {
                val sentences = runCatching { client.sentences(queue.credential, queue.libraryId, queue.bookId, chapter.index) }.getOrNull() ?: return@launch
                queue.lastChapter = chapter.index
                if (sentences.isNotEmpty()) {
                    player.addMediaItems(sentences.mapIndexed { i, s -> item(queue, chapter, i, s) })
                    return@launch
                }
                chapter = NarrationPlan.nextChapter(queue.chapters, chapter.index)
            }
            queue.finished = true
        }
    }

    private fun setSleep(args: Bundle) {
        val player = session?.player ?: return
        player.volume = 1f
        sleep = when (args.getString(SLEEP_KIND)) {
            "after" -> NarrationSleep.After(args.getInt(SLEEP_MINUTES).coerceAtLeast(1))
            "chapter" -> NarrationSleep.EndOfChapter
            else -> NarrationSleep.Off
        }
        sleepChapter = player.currentMediaItem?.mediaMetadata?.extras?.getInt(CHAPTER, -1) ?: -1
        sleepDeadline = (sleep as? NarrationSleep.After)?.let { System.currentTimeMillis() + it.minutes * 60_000L } ?: 0L
    }

    private fun tickSleep(player: Player) {
        if (sleep !is NarrationSleep.After) return
        val left = sleepDeadline - System.currentTimeMillis()
        if (left <= 0) {
            sleep = NarrationSleep.Off
            player.pause()
            player.volume = 1f
        } else {
            player.volume = sleepFadeVolume(left)
        }
    }

    /** Keeps the place in the book on this device, so opening the reader later resumes where the voice got to. */
    private fun rememberPosition(extras: Bundle) {
        val key = extras.getString(PROGRESS_KEY).orEmpty()
        val href = extras.getString(HREF).orEmpty()
        if (key.isEmpty() || href.isEmpty()) return
        runCatching {
            val locator = JSONObject()
                .put("href", href).put("type", "application/xhtml+xml")
                .put("locations", JSONObject().put("progression", extras.getDouble(PROGRESSION)))
                .put("text", JSONObject().put("before", extras.getString(BEFORE).orEmpty()).put("highlight", extras.getString(TEXT).orEmpty()).put("after", extras.getString(AFTER).orEmpty()))
            locatorStore.save(key, org.readium.r2.shared.publication.Locator.fromJSON(locator) ?: return@runCatching)
        }
    }

    companion object {
        private const val SESSION_ID = "mochil-narration"
        const val ACTION_START = "net.kdhmedia.mochil.narration.START"
        const val ACTION_START_AUDIO = "net.kdhmedia.mochil.narration.START_AUDIO"
        const val ACTION_SLEEP = "net.kdhmedia.mochil.narration.SLEEP"
        val START = SessionCommand(ACTION_START, Bundle.EMPTY)
        val START_AUDIO = SessionCommand(ACTION_START_AUDIO, Bundle.EMPTY)
        val SLEEP = SessionCommand(ACTION_SLEEP, Bundle.EMPTY)
        const val URLS = "urls"
        const val ITEM = "item"
        const val POSITION_MS = "position_ms"

        const val LIBRARY = "library"
        const val BOOK = "book"
        const val TITLE = "title"
        const val HREF = "href"
        const val CHAPTER = "chapter"
        const val INDEX = "index"
        const val PROGRESSION = "progression"
        const val PROGRESS_KEY = "progress_key"
        const val VOICE = "voice"
        const val SPEED = "speed"
        const val BEFORE = "before"
        const val TEXT = "text"
        const val AFTER = "after"
        const val SLEEP_KIND = "sleep_kind"
        const val SLEEP_MINUTES = "sleep_minutes"
        private const val MAX_FAILURES = 3
    }
}
