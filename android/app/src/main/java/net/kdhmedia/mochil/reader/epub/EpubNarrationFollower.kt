package net.kdhmedia.mochil.reader.epub

import android.os.SystemClock
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.Lifecycle
import net.kdhmedia.mochil.narration.NarrationPlace
import net.kdhmedia.mochil.narration.NarrationPlan
import net.kdhmedia.mochil.readalong.ReadAlongClip
import org.readium.r2.navigator.Decoration
import org.readium.r2.navigator.epub.EpubNavigatorFragment
import org.readium.r2.navigator.preferences.Theme
import org.readium.r2.shared.ExperimentalReadiumApi
import org.readium.r2.shared.publication.Locator
import org.readium.r2.shared.publication.Publication

/** Draws the sentence being read aloud and keeps the page on it while the reader is on screen. */
@OptIn(ExperimentalReadiumApi::class)
internal class EpubNarrationFollower(
    private val navigator: () -> EpubNavigatorFragment?,
    private val publication: () -> Publication?,
    private val theme: () -> Theme,
    private val lifecycle: Lifecycle,
) {
    /** True while the reader has turned away from the sentence being read: narration then leaves the page alone. */
    var browsing by mutableStateOf(false)
        private set
    private var lastClip: ReadAlongClip? = null
    private var lastLocator: Locator? = null
    private var lastMoveAt = 0L

    /** Marks and follows [place]; null clears the mark. */
    suspend fun show(place: NarrationPlace?) {
        val reader = navigator() ?: return
        if (place == null) {
            runCatching { reader.applyDecorations(emptyList(), GROUP) }
            return
        }
        val opened = publication() ?: return
        val wanted = NarrationPlan.normalizedHref(place.href)
        val link = opened.readingOrder.firstOrNull { NarrationPlan.normalizedHref(it.url().toString()) == wanted } ?: return
        val locator = opened.locatorFromLink(link)
            ?.copyWithLocations(progression = place.progression.coerceIn(0.0, 1.0))
            ?.copy(text = Locator.Text(place.before, place.text, place.after)) ?: return
        runCatching { reader.applyDecorations(listOf(Decoration(ID, locator, Decoration.Style.Highlight(tint(theme())))), GROUP) }
        // Only a reader that is on screen turns pages; a locked phone just plays.
        if (lifecycle.currentState.isAtLeast(Lifecycle.State.STARTED)) runCatching { reader.go(locator, animated = false) }
    }

    /**
     * Marks the element a recorded sentence belongs to and, normally, keeps the page on it. [progression] is how
     * far through its chapter it is. If the reader has deliberately turned to another page, the page is left
     * alone and [browsing] is set instead, until the sentence being read is on the page again.
     */
    suspend fun show(clip: ReadAlongClip?, progression: Double = 0.0) {
        val reader = navigator() ?: return
        if (clip == null) {
            lastClip = null; lastLocator = null; browsing = false
            runCatching { reader.applyDecorations(emptyList(), GROUP) }
            return
        }
        val opened = publication() ?: return
        val wanted = NarrationPlan.normalizedHref(clip.href)
        val link = opened.readingOrder.firstOrNull { NarrationPlan.normalizedHref(it.url().toString()) == wanted } ?: return
        // The element is found by its id; where the page cannot find it, the share of the chapter still lands near it.
        val selector = "[id=\"" + clip.id.replace("\\", "\\\\").replace("\"", "\\\"") + "\"]"
        val locator = opened.locatorFromLink(link)
            ?.copyWithLocations(fragments = listOf(clip.id), progression = progression.coerceIn(0.0, 1.0), otherLocations = mapOf("cssSelector" to selector))
            ?: return
        runCatching { reader.applyDecorations(listOf(Decoration(ID, locator, Decoration.Style.Highlight(tint(theme())))), GROUP) }
        lastLocator = locator
        val visible = lifecycle.currentState.isAtLeast(Lifecycle.State.STARTED)
        val onPage = isOnPage(reader, link, clip.id)
        if (onPage) browsing = false
        else if (!browsing) {
            // The sentence before was on the page when narration was following; if it no longer is, the reader turned away.
            val previous = lastClip
            val settled = SystemClock.elapsedRealtime() - lastMoveAt > SETTLE_MS
            browsing = previous != null && settled && !isClipOnPage(reader, opened, previous)
        }
        lastClip = clip
        if (!onPage && !browsing && visible) {
            lastMoveAt = SystemClock.elapsedRealtime()
            runCatching { reader.go(locator, animated = false) }
        }
    }

    /** Takes the page back to the sentence being read, after the reader turned away. */
    suspend fun showCurrent() {
        val locator = lastLocator ?: return
        browsing = false
        lastMoveAt = SystemClock.elapsedRealtime()
        runCatching { navigator()?.go(locator, animated = false) }
    }

    private suspend fun isClipOnPage(reader: EpubNavigatorFragment, opened: Publication, clip: ReadAlongClip): Boolean {
        val wanted = NarrationPlan.normalizedHref(clip.href)
        val link = opened.readingOrder.firstOrNull { NarrationPlan.normalizedHref(it.url().toString()) == wanted } ?: return false
        return isOnPage(reader, link, clip.id)
    }

    /** The index in [ids] of the element the reader has selected (long-pressed), or -1. */
    suspend fun selected(ids: List<String>): Int {
        val reader = navigator() ?: return -1
        if (ids.isEmpty()) return -1
        val script = "(function(ids){var s=window.getSelection();if(!s||!s.rangeCount)return -1;var n=s.anchorNode;" +
            "while(n&&n.nodeType!==1)n=n.parentNode;for(;n&&n!==document;n=n.parentElement){var i=ids.indexOf(n.id);if(i>=0)return i;}return -1;})(" +
            org.json.JSONArray(ids).toString() + ")"
        return runCatching { reader.evaluateJavascript(script)?.trim()?.trim('"')?.toIntOrNull() }.getOrNull() ?: -1
    }

    /** The index in [ids] of the first element showing on the current page, or -1 when none is or the page cannot say. */
    suspend fun firstOnPage(ids: List<String>): Int {
        val reader = navigator() ?: return -1
        if (ids.isEmpty()) return -1
        val list = org.json.JSONArray(ids).toString()
        val script = "(function(ids){for(var i=0;i<ids.length;i++){var e=document.getElementById(ids[i]);if(!e)continue;" +
            "var r=e.getBoundingClientRect();if(r.right>0&&r.left<window.innerWidth&&r.bottom>0&&r.top<window.innerHeight)return i;}return -1;})($list)"
        return runCatching { reader.evaluateJavascript(script)?.trim()?.trim('"')?.toIntOrNull() }.getOrNull() ?: -1
    }

    private suspend fun isOnPage(reader: EpubNavigatorFragment, link: org.readium.r2.shared.publication.Link, id: String): Boolean {
        val shown = reader.currentLocator.value.href
        if (NarrationPlan.normalizedHref(shown.toString()) != NarrationPlan.normalizedHref(link.url().toString())) return false
        return firstOnPage(listOf(id)) == 0
    }

    private fun tint(theme: Theme): Int = when (theme) {
        Theme.DARK -> 0xFF7FA6D9.toInt()
        Theme.SEPIA -> 0xFFC9B27C.toInt()
        Theme.LIGHT -> 0xFFA8C8F0.toInt()
    }

    private companion object {
        const val SETTLE_MS = 1500L
        const val GROUP = "mochil-narration"
        const val ID = "narration"
    }
}
