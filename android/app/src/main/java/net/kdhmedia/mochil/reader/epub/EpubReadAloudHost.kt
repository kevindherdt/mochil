package net.kdhmedia.mochil.reader.epub

import android.content.pm.PackageManager
import androidx.activity.ComponentActivity
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleEventObserver
import androidx.lifecycle.lifecycleScope
import kotlinx.coroutines.launch
import net.kdhmedia.mochil.auth.DeviceCredential
import net.kdhmedia.mochil.isTelevisionUi
import net.kdhmedia.mochil.narration.NarrationClient
import net.kdhmedia.mochil.narration.NarrationController
import net.kdhmedia.mochil.narration.NarrationPreferences
import net.kdhmedia.mochil.narration.ReadAloud
import net.kdhmedia.mochil.readalong.ReadAlongClient
import net.kdhmedia.mochil.readalong.ReadAlongController
import org.readium.r2.navigator.epub.EpubNavigatorFragment
import org.readium.r2.navigator.preferences.Theme
import org.readium.r2.shared.publication.Locator
import org.readium.r2.shared.publication.Publication

/**
 * Reading aloud in the EPUB reader: a book that carries its own recording plays that, any other book
 * is read by the server's voice when its narration module is on. Keeps this out of the reader itself.
 */
internal class EpubReadAloudHost(
    private val activity: ComponentActivity,
    navigator: () -> EpubNavigatorFragment?,
    private val publication: () -> Publication?,
    theme: () -> Theme,
) {
    private val follower = EpubNarrationFollower(navigator, publication, theme, activity.lifecycle)
    private var current by mutableStateOf<ReadAloud?>(null)

    init {
        activity.lifecycle.addObserver(LifecycleEventObserver { _, event ->
            when (event) {
                // Catches the page up with what was read while the screen was off.
                Lifecycle.Event.ON_RESUME -> current?.currentPlace()
                // Closing the book ends reading aloud; a reader the system recreates leaves it running.
                Lifecycle.Event.ON_DESTROY -> {
                    if (activity.isFinishing) current?.stop()
                    current?.release()
                }
                else -> Unit
            }
        })
    }

    fun setUp(credential: DeviceCredential?, libraryId: String, bookId: String, progressKey: String, title: String) {
        if (credential == null || libraryId.isEmpty() || bookId.isEmpty()) return
        // The television reader is left exactly as it was.
        if (isTelevisionUi(activity.resources.configuration.uiMode, activity.packageManager.hasSystemFeature(PackageManager.FEATURE_LEANBACK))) return
        val scope = activity.lifecycleScope
        val preferences = NarrationPreferences(activity)
        scope.launch {
            // A book that carries its own recording (a Storyteller book) plays that; any other is read by the server's voice.
            val recording = runCatching { ReadAlongClient().overlay(credential, libraryId, bookId) }.getOrNull()
            if (recording != null) {
                current = ReadAlongController(
                    context = activity, preferences = preferences, credential = credential, overlay = recording,
                    libraryId = libraryId, bookId = bookId, progressKey = progressKey, bookTitle = title, scope = scope,
                    chapterOrder = { publication()?.readingOrder?.map { it.url().toString() }.orEmpty() },
                    firstOnPage = follower::firstOnPage, selectedOf = follower::selected,
                    onClip = { clip, progression -> scope.launch { follower.show(clip, progression) } },
                )
                return@launch
            }
            val status = runCatching { NarrationClient().status(credential) }.getOrNull() ?: return@launch
            val reading = NarrationController(
                context = activity, preferences = preferences, bookId = bookId, libraryId = libraryId,
                progressKey = progressKey, bookTitle = title,
                onPlace = { place -> scope.launch { follower.show(place) } },
            )
            reading.connect(status)
            current = reading
        }
    }

    /** Whether reading aloud is on offer, so the text-selection menu can offer to read from the selection. */
    val available: Boolean get() = current?.ui?.available == true

    /** Reads aloud from the selected text. Only reached by long-pressing and choosing it, never by a tap. */
    fun readFromSelection(selection: Locator) {
        current?.readFromSelection(selection.href.toString(), selection.locations.progression ?: 0.0)
    }

    /** The narration settings as rows, for the settings sheet's Narration tab. */
    @Composable
    fun Settings() {
        current?.takeIf { it.ui.available }?.let { reading ->
            EpubNarrationSettings(ui = reading.ui, onVoice = reading::setVoice, onSpeed = reading::setSpeed, onSleep = reading::setSleep, onStop = reading::stop)
        }
    }

    /** The play button and its menu, once reading aloud is on offer. [page] is where the reader is now. */
    @Composable
    fun Controls(page: () -> Locator?) {
        current?.takeIf { it.ui.available }?.let { reading ->
            EpubNarrationControls(
                ui = reading.ui,
                onToggle = {
                    val here = page()
                    reading.toggle(here?.href?.toString().orEmpty(), here?.locations?.progression ?: 0.0)
                },
                onVoice = reading::setVoice,
                onSpeed = reading::setSpeed,
                onSleep = reading::setSleep,
                onStop = reading::stop,
                onBackToReading = if (follower.browsing) ({ activity.lifecycleScope.launch { follower.showCurrent() } }) else null,
            )
        }
    }
}
