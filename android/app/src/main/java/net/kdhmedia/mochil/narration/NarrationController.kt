package net.kdhmedia.mochil.narration

import android.content.ComponentName
import android.content.Context
import android.os.Bundle
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.core.content.ContextCompat
import androidx.media3.common.MediaItem
import androidx.media3.common.Player
import androidx.media3.session.MediaController
import androidx.media3.session.SessionCommand
import androidx.media3.session.SessionToken
import net.kdhmedia.mochil.auth.DeviceCredential

/** What the reader shows of narration. */
data class NarrationUi(
    /** Whether reading aloud is on offer for this book; otherwise the reader shows nothing. */
    val available: Boolean = false,
    val speeds: List<Float> = narrationSpeeds,
    /** Whether "end of chapter" makes sense: a recording has no chapters of its own to stop at. */
    val chapterSleep: Boolean = true,
    val voices: List<NarrationVoice> = emptyList(),
    val voice: String = "",
    val speed: Float = 1f,
    val active: Boolean = false,
    val playing: Boolean = false,
    val sleepText: String = "",
)

/** The sentence being read, as the reader needs it to draw and follow it. */
data class NarrationPlace(val href: String, val progression: Double, val before: String, val text: String, val after: String)

/** What the reader needs from whatever is reading the book aloud: the server's voice, or the book's own recording. */
interface ReadAloud {
    val ui: NarrationUi
    fun toggle(href: String, progression: Double)
    fun stop()
    fun setVoice(id: String)
    fun setSpeed(speed: Float)
    fun setSleep(sleep: NarrationSleep)
    /** Starts, or moves on to, reading from where the reader selected text. */
    fun readFromSelection(href: String, progression: Double)
    /** Catches the reader up with what is being read after it was away. */
    fun currentPlace()
    fun release()
}

/**
 * The reader's end of narration: it asks the service to read this book from the current page and
 * reports the sentence being read. The service keeps going when the reader is not on screen.
 */
class NarrationController(
    private val context: Context,
    private val preferences: NarrationPreferences,
    private val bookId: String,
    private val libraryId: String,
    private val progressKey: String,
    private val bookTitle: String,
    private val onPlace: (NarrationPlace?) -> Unit,
) : ReadAloud {
    override var ui by mutableStateOf(NarrationUi())
        private set
    private var controller: MediaController? = null
    private var defaultVoice = ""
    private var lastPlace: NarrationPlace? = null

    private var sleepChoice: NarrationSleep = NarrationSleep.Off
    private var connecting = false
    private var connectingFuture: com.google.common.util.concurrent.ListenableFuture<MediaController>? = null
    private var released = false
    private val waiting = mutableListOf<(MediaController) -> Unit>()

    /**
     * Shows the controls when narration is on. [status] is null when it is off or not set up, and the
     * reader then shows nothing. The player service is only started when reading begins, or at once
     * when this book is already being read, so opening a book never depends on it.
     */
    fun connect(status: NarrationStatus?) {
        if (status == null) return
        defaultVoice = status.defaultVoice
        val saved = preferences.voice()
        val voice = if (status.voices.any { it.id == saved }) saved else status.defaultVoice
        ui = ui.copy(available = true, voices = status.voices, voice = voice, speed = preferences.speed())
        if (preferences.activeBook() == "$libraryId/$bookId") withController { }
    }

    /** Runs [action] once the player is reachable, reaching it first if need be. If it cannot be reached, nothing happens. */
    private fun withController(action: (MediaController) -> Unit) {
        if (released) return
        controller?.let { action(it); return }
        waiting += action
        if (connecting) return
        connecting = true
        // Reading aloud is an extra: if the player cannot be reached the reader simply goes without it.
        val future = runCatching {
            MediaController.Builder(context, SessionToken(context, ComponentName(context, NarrationService::class.java))).buildAsync()
        }.getOrNull()
        if (future == null) { connecting = false; waiting.clear(); return }
        connectingFuture = future
        future.addListener({
            connecting = false
            connectingFuture = null
            val connected = runCatching { future.get() }.getOrNull()
            if (connected == null) { waiting.clear(); return@addListener }
            // The controller this call is waiting for may finish connecting after release() already ran
            // (release() cannot cancel a buildAsync() future already past its binder handshake). Accepting
            // it here would revive a live MediaController/service connection with nothing left to release it.
            if (released) { connected.release(); waiting.clear(); return@addListener }
            controller = connected
            connected.addListener(object : Player.Listener {
                override fun onMediaItemTransition(mediaItem: MediaItem?, reason: Int) { follow(mediaItem) }
                override fun onIsPlayingChanged(isPlaying: Boolean) { refresh() }
                override fun onPlaybackStateChanged(playbackState: Int) { refresh() }
            })
            // The service may already be reading this book: the reader came back to it.
            if (belongsHere(connected.currentMediaItem)) follow(connected.currentMediaItem)
            refresh()
            val pending = waiting.toList()
            waiting.clear()
            pending.forEach { it(connected) }
        }, ContextCompat.getMainExecutor(context))
    }

    private fun belongsHere(item: MediaItem?): Boolean {
        val extras = item?.mediaMetadata?.extras ?: return false
        return extras.getString(NarrationService.BOOK) == bookId && extras.getString(NarrationService.LIBRARY) == libraryId
    }

    private fun refresh() {
        val c = controller ?: return
        val here = belongsHere(c.currentMediaItem)
        ui = ui.copy(active = here && c.playbackState != Player.STATE_IDLE && c.playbackState != Player.STATE_ENDED, playing = here && c.isPlaying)
    }

    private fun follow(item: MediaItem?) {
        if (!belongsHere(item)) { lastPlace = null; onPlace(null); return }
        val e = item?.mediaMetadata?.extras ?: return
        val place = NarrationPlace(e.getString(NarrationService.HREF).orEmpty(), e.getDouble(NarrationService.PROGRESSION), e.getString(NarrationService.BEFORE).orEmpty(), e.getString(NarrationService.TEXT).orEmpty(), e.getString(NarrationService.AFTER).orEmpty())
        lastPlace = place
        onPlace(place)
        refresh()
    }

    /** Starts reading at this page, or pauses and resumes what is already being read. */
    override fun toggle(href: String, progression: Double) {
        withController { c ->
            if (belongsHere(c.currentMediaItem) && c.playbackState != Player.STATE_IDLE && c.playbackState != Player.STATE_ENDED) {
                if (c.isPlaying) c.pause() else c.play()
            } else start(c, href, progression)
        }
    }

    private fun start(c: MediaController, href: String, progression: Double) {
        val args = Bundle().apply {
            putString(NarrationService.LIBRARY, libraryId); putString(NarrationService.BOOK, bookId)
            putString(NarrationService.TITLE, bookTitle); putString(NarrationService.PROGRESS_KEY, progressKey)
            putString(NarrationService.HREF, href); putDouble(NarrationService.PROGRESSION, if (progression.isFinite()) progression else 0.0)
            putString(NarrationService.VOICE, ui.voice.ifEmpty { defaultVoice }); putFloat(NarrationService.SPEED, ui.speed)
            putAll(sleepArguments(sleepChoice))
        }
        c.sendCustomCommand(NarrationService.START, args)
    }

    override fun stop() {
        controller?.run { if (belongsHere(currentMediaItem)) { stop(); clearMediaItems() } }
        lastPlace = null
        onPlace(null)
        refresh()
    }

    /** Changes voice: the sentence being read is read again in the new voice. */
    override fun readFromSelection(href: String, progression: Double) {
        withController { start(it, href, progression) }
    }

    override fun setVoice(id: String) {
        preferences.saveVoice(id)
        ui = ui.copy(voice = id)
        val place = lastPlace
        if (ui.active && place != null) withController { start(it, place.href, place.progression) }
    }

    override fun setSpeed(speed: Float) {
        preferences.saveSpeed(speed)
        ui = ui.copy(speed = speed)
        controller?.setPlaybackSpeed(speed) // otherwise it is sent when reading starts
    }

    override fun setSleep(sleep: NarrationSleep) {
        sleepChoice = sleep
        controller?.takeIf { ui.active }?.sendCustomCommand(NarrationService.SLEEP, sleepArguments(sleep)) // otherwise it is sent when reading starts
        ui = ui.copy(sleepText = when (sleep) {
            NarrationSleep.Off -> ""
            is NarrationSleep.After -> "Sleep timer: ${sleep.minutes} min"
            NarrationSleep.EndOfChapter -> "Sleep timer: end of chapter"
        })
    }

    private fun sleepArguments(sleep: NarrationSleep) = Bundle().apply {
        when (sleep) {
            NarrationSleep.Off -> putString(NarrationService.SLEEP_KIND, "off")
            is NarrationSleep.After -> { putString(NarrationService.SLEEP_KIND, "after"); putInt(NarrationService.SLEEP_MINUTES, sleep.minutes) }
            NarrationSleep.EndOfChapter -> putString(NarrationService.SLEEP_KIND, "chapter")
        }
    }

    /** The place being read right now, for catching the reader up after it was away. */
    override fun currentPlace() { controller?.let { follow(it.currentMediaItem) } }

    override fun release() {
        released = true
        connectingFuture?.cancel(false)
        connectingFuture = null
        waiting.clear()
        controller?.release()
        controller = null
    }
}
