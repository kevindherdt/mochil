package net.kdhmedia.mochil.readalong

import android.content.ComponentName
import android.content.Context
import android.os.Bundle
import android.os.Handler
import android.os.Looper
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.core.content.ContextCompat
import androidx.media3.common.MediaItem
import androidx.media3.common.Player
import androidx.media3.session.MediaController
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.launch
import androidx.media3.session.SessionToken
import net.kdhmedia.mochil.auth.DeviceCredential
import net.kdhmedia.mochil.narration.NarrationPreferences
import net.kdhmedia.mochil.narration.NarrationService
import net.kdhmedia.mochil.narration.NarrationSleep
import net.kdhmedia.mochil.narration.NarrationUi
import net.kdhmedia.mochil.narration.ReadAloud
import net.kdhmedia.mochil.narration.recordingSpeeds
import net.kdhmedia.mochil.reader.epub.EpubLocatorStore
import org.json.JSONObject

/**
 * The reader's end of a read-along book's own recording. The service plays the audio; this works out,
 * from where the audio has got to, which sentence is being spoken, so the reader can mark it and keep
 * the page on it, and it remembers the place so the book reopens there.
 */
class ReadAlongController(
    private val context: Context,
    private val preferences: NarrationPreferences,
    private val credential: DeviceCredential,
    private val overlay: ReadAlongOverlay,
    private val libraryId: String,
    private val bookId: String,
    private val progressKey: String,
    private val bookTitle: String,
    private val scope: CoroutineScope,
    /** The chapters in reading order, for finding where to start from a page that has no narration. */
    private val chapterOrder: () -> List<String>,
    /** Which of these elements is the first showing on the page being read, as an index, or -1. */
    private val firstOnPage: suspend (List<String>) -> Int,
    /** Which of these elements holds the text the reader selected, as an index, or -1. */
    private val selectedOf: suspend (List<String>) -> Int,
    /** The clip being spoken, and how far through its chapter it is; null when none. */
    private val onClip: (ReadAlongClip?, Double) -> Unit,
) : ReadAloud {
    override var ui by mutableStateOf(
        NarrationUi(available = true, speeds = recordingSpeeds, chapterSleep = false, speed = preferences.recordingSpeed()),
    )
        private set
    private var controller: MediaController? = null
    private var connecting = false
    private var connectingFuture: com.google.common.util.concurrent.ListenableFuture<MediaController>? = null
    private var released = false
    private val waiting = mutableListOf<(MediaController) -> Unit>()
    private var sleepChoice: NarrationSleep = NarrationSleep.Off
    private var clip = -1
    private val handler = Handler(Looper.getMainLooper())
    private val tick = object : Runnable {
        override fun run() {
            follow()
            if (controller != null) handler.postDelayed(this, TICK_MS)
        }
    }

    init {
        // Back in a book that is still being read: pick the reading up.
        if (preferences.activeBook() == "$libraryId/$bookId") withController { }
    }

    private fun withController(action: (MediaController) -> Unit) {
        if (released) return
        controller?.let { action(it); return }
        waiting += action
        if (connecting) return
        connecting = true
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
            // it here would revive a live MediaController/service connection and restart the repeating
            // `tick` Handler loop forever, with nothing left to ever call release() a second time.
            if (released) { connected.release(); waiting.clear(); return@addListener }
            controller = connected
            connected.addListener(object : Player.Listener {
                override fun onIsPlayingChanged(isPlaying: Boolean) { refresh() }
                override fun onPlaybackStateChanged(playbackState: Int) { refresh() }
                override fun onMediaItemTransition(mediaItem: MediaItem?, reason: Int) { follow() }
            })
            refresh()
            handler.post(tick)
            val pending = waiting.toList()
            waiting.clear()
            pending.forEach { it(connected) }
        }, ContextCompat.getMainExecutor(context))
    }

    private fun belongsHere(item: MediaItem?): Boolean {
        val extras = item?.mediaMetadata?.extras ?: return false
        return item.mediaId.startsWith("recording:") &&
            extras.getString(NarrationService.BOOK) == bookId && extras.getString(NarrationService.LIBRARY) == libraryId
    }

    private fun refresh() {
        val c = controller ?: return
        val here = belongsHere(c.currentMediaItem)
        ui = ui.copy(active = here && c.playbackState != Player.STATE_IDLE && c.playbackState != Player.STATE_ENDED, playing = here && c.isPlaying)
    }

    /** Finds the sentence being spoken from the audio's position and tells the reader when it changes. */
    private fun follow() {
        val c = controller ?: return
        if (!belongsHere(c.currentMediaItem)) {
            if (clip != -1) { clip = -1; onClip(null, 0.0) }
            return
        }
        val found = overlay.clipAt(c.currentMediaItemIndex, c.currentPosition / 1000.0, clip.coerceAtLeast(0))
        if (found == clip || found < 0) return
        clip = found
        val spoken = overlay.clips[found]
        onClip(spoken.takeIf { it.end - it.begin <= ReadAlongOverlay.MAX_SENTENCE_SECONDS }, overlay.progressionOf(found))
        rememberPlace(found)
    }

    /** Keeps the place on this device, so opening the book later resumes where the recording got to. */
    private fun rememberPlace(index: Int) {
        runCatching {
            val locator = JSONObject()
                .put("href", overlay.clips[index].href).put("type", "application/xhtml+xml")
                .put("locations", JSONObject().put("progression", overlay.progressionOf(index)))
            EpubLocatorStore(context).save(progressKey, org.readium.r2.shared.publication.Locator.fromJSON(locator) ?: return@runCatching)
        }
    }

    /** Pauses what is playing; otherwise plays from the first narrated sentence on the page being read. */
    override fun toggle(href: String, progression: Double) {
        withController { c ->
            val running = isRunning(c)
            if (running && c.isPlaying) { c.pause(); return@withController }
            scope.launch {
                val spoken = if (running) clip else -1
                val own = overlay.clipsOf(href)
                val onPage = own.getOrNull(firstOnPage(own.map { overlay.clips[it].id }))
                when {
                    // Paused where the page still is: carry on.
                    running && spoken >= 0 && own.contains(spoken) && onPage != null && onPage == spoken -> c.play()
                    // The page was turned while paused, or nothing was playing: read from this page.
                    else -> start(c, onPage ?: overlay.startClipFrom(chapterOrder(), href, progression), running)
                }
            }
        }
    }

    private fun isRunning(c: MediaController) =
        belongsHere(c.currentMediaItem) && c.playbackState != Player.STATE_IDLE && c.playbackState != Player.STATE_ENDED

    /** Reads from the sentence the reader long-pressed, whether or not something is already being read. */
    override fun readFromSelection(href: String, progression: Double) {
        withController { c ->
            scope.launch {
                val own = overlay.clipsOf(href)
                val picked = own.getOrNull(selectedOf(own.map { overlay.clips[it].id }))
                start(c, picked ?: overlay.startClipFrom(chapterOrder(), href, progression), isRunning(c))
            }
        }
    }

    private fun start(c: MediaController, from: Int, running: Boolean) {
        val first = overlay.clips[from]
        if (running) {
            c.seekTo(first.audio, (first.begin * 1000).toLong())
            c.play()
            return
        }
        val args = Bundle().apply {
            putString(NarrationService.LIBRARY, libraryId); putString(NarrationService.BOOK, bookId)
            putString(NarrationService.TITLE, bookTitle)
            putStringArrayList(NarrationService.URLS, ArrayList(overlay.audio.map { ReadAlongClient.audioUrl(credential, libraryId, bookId, it) }))
            putInt(NarrationService.ITEM, first.audio); putLong(NarrationService.POSITION_MS, (first.begin * 1000).toLong())
            putFloat(NarrationService.SPEED, ui.speed)
            putAll(sleepArguments(sleepChoice))
        }
        c.sendCustomCommand(NarrationService.START_AUDIO, args)
    }

    override fun stop() {
        controller?.run { if (belongsHere(currentMediaItem)) { stop(); clearMediaItems() } }
        clip = -1
        onClip(null, 0.0)
        refresh()
    }

    /** A recording has one voice. */
    override fun setVoice(id: String) = Unit

    override fun setSpeed(speed: Float) {
        preferences.saveRecordingSpeed(speed)
        ui = ui.copy(speed = speed)
        controller?.setPlaybackSpeed(speed) // otherwise it is sent when reading starts
    }

    override fun setSleep(sleep: NarrationSleep) {
        val chosen = if (sleep == NarrationSleep.EndOfChapter) NarrationSleep.Off else sleep
        sleepChoice = chosen
        controller?.takeIf { ui.active }?.sendCustomCommand(NarrationService.SLEEP, sleepArguments(chosen))
        ui = ui.copy(sleepText = (chosen as? NarrationSleep.After)?.let { "Sleep timer: ${it.minutes} min" }.orEmpty())
    }

    private fun sleepArguments(sleep: NarrationSleep) = Bundle().apply {
        when (sleep) {
            is NarrationSleep.After -> { putString(NarrationService.SLEEP_KIND, "after"); putInt(NarrationService.SLEEP_MINUTES, sleep.minutes) }
            else -> putString(NarrationService.SLEEP_KIND, "off")
        }
    }

    override fun currentPlace() { clip = -1; follow() }

    override fun release() {
        released = true
        handler.removeCallbacks(tick)
        connectingFuture?.cancel(false)
        connectingFuture = null
        waiting.clear()
        controller?.release()
        controller = null
    }

    private companion object { const val TICK_MS = 250L }
}
