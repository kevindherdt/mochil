package net.kdhmedia.mochil.reader.epub

import android.view.ActionMode
import android.view.Menu
import android.view.MenuItem
import android.view.View
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.launch
import kotlinx.coroutines.suspendCancellableCoroutine
import kotlin.coroutines.resume
import net.kdhmedia.mochil.diagnostics.PerformanceDiagnostics
import org.readium.r2.navigator.epub.EpubNavigatorFragment
import org.readium.r2.shared.ExperimentalReadiumApi
import org.readium.r2.shared.publication.Locator
import org.readium.r2.shared.publication.Publication

internal data class EpubSelectionCallbacks(
    val onDeleteRequested: (String) -> Unit,
    val onMessage: (String, Boolean) -> Unit,
    val onShowChrome: () -> Unit,
    val onApplyDecorations: suspend () -> Boolean,
    val onSynchronize: () -> Unit,
    val canReadAloud: () -> Boolean = { false },
    val onReadAloud: (Locator) -> Unit = {},
)

/** Owns Android's transient text-selection actions and stable highlight creation. */
@OptIn(ExperimentalReadiumApi::class)
internal class EpubSelectionActionController(
    private val scope: CoroutineScope,
    private val navigator: () -> EpubNavigatorFragment?,
    private val publication: () -> Publication?,
    private val annotations: EpubAnnotationSession,
    private val callbacks: EpubSelectionCallbacks,
) {
    fun onStarted(mode: ActionMode, chromeHasFocus: Boolean) {
        if (chromeHasFocus || mode.menu.findItem(HIGHLIGHT_ACTION_ID) != null) return
        scope.launch {
            val reader = navigator() ?: return@launch
            val selection = runCatching { reader.currentSelection() }.getOrNull()
            if (selection?.locator?.text?.highlight.isNullOrBlank() || mode.menu.findItem(HIGHLIGHT_ACTION_ID) != null) return@launch
            val existing = annotations.state.visible.firstOrNull { it.matchesSelection(selection.locator) }
            mode.menu.add(Menu.NONE, HIGHLIGHT_ACTION_ID, Menu.NONE, if (existing == null) "Highlight" else "Remove highlight").apply {
                setShowAsAction(MenuItem.SHOW_AS_ACTION_IF_ROOM)
                setOnMenuItemClickListener {
                    if (existing == null) createHighlight(mode)
                    else {
                        reader.clearSelection()
                        mode.finish()
                        callbacks.onDeleteRequested(existing.id)
                    }
                    true
                }
            }
            if (callbacks.canReadAloud() && mode.menu.findItem(READ_ALOUD_ACTION_ID) == null) {
                mode.menu.add(Menu.NONE, READ_ALOUD_ACTION_ID, Menu.NONE, "Read from here").apply {
                    setShowAsAction(MenuItem.SHOW_AS_ACTION_IF_ROOM)
                    setOnMenuItemClickListener {
                        callbacks.onReadAloud(selection.locator)
                        reader.clearSelection()
                        mode.finish()
                        true
                    }
                }
            }
            // Rebuilding WebView's floating action mode duplicates PROCESS_TEXT
            // entries on some Samsung releases. Refresh only the content bounds.
            mode.invalidateContentRect()
        }
    }

    fun onFinished() {
        PerformanceDiagnostics.record("reader", "highlight_action_mode_finished")
    }

    private fun createHighlight(mode: ActionMode) {
        val reader = navigator() ?: return
        scope.launch {
            val selection = runCatching { reader.currentSelection() }.getOrNull() ?: return@launch
            PerformanceDiagnostics.record("reader", "highlight_action_selected")
            val opened = publication() ?: return@launch
            val spineIndex = opened.readingOrder.indexOfFirst { it.url() == selection.locator.href }.coerceAtLeast(0)
            val stableLocator = stableSelectionLocator(opened, selection.locator)
            if (stableLocator == null) {
                PerformanceDiagnostics.record("reader", "highlight_locator_resolved", outcome = "error")
                callbacks.onMessage("This selection could not be anchored. Try highlighting a shorter sentence.", true)
                reader.clearSelection()
                mode.finish()
                callbacks.onShowChrome()
                return@launch
            }
            PerformanceDiagnostics.record("reader", "highlight_locator_resolved")
            val created = annotations.createHighlight(stableLocator, spineIndex)
            PerformanceDiagnostics.record("reader", "highlight_state_saved", outcome = if (created) "ok" else "error")

            // Readium repaints after Android closes its selection mode. Decorating
            // before the following frames makes the newest mark disappear.
            reader.clearSelection()
            mode.finish()
            awaitNavigatorFrame(reader.publicationView)
            awaitNavigatorFrame(reader.publicationView)
            val selectionCleared = runCatching { reader.currentSelection() }.getOrNull() == null
            PerformanceDiagnostics.record("reader", "highlight_selection_cleared", outcome = if (selectionCleared) "ok" else "error")
            if (created && callbacks.onApplyDecorations()) {
                PerformanceDiagnostics.record("reader", "highlight_decorations_applied")
                callbacks.onMessage("Highlight saved.", false)
            } else if (created) {
                PerformanceDiagnostics.record("reader", "highlight_decorations_applied", outcome = "error")
            }
            if (created) callbacks.onSynchronize()
        }
    }

    private suspend fun stableSelectionLocator(opened: Publication, selected: Locator): Locator? {
        val query = stableHighlightQuery(selected.text.highlight.orEmpty())
        if (query.isBlank()) return null
        val matches = runCatching { EpubSearch.find(opened, query, MAX_ANCHOR_MATCHES) }.getOrDefault(emptyList())
        return chooseStableSelectionLocator(selected, matches)
    }

    private companion object {
        const val HIGHLIGHT_ACTION_ID = 0x424F424F
        const val READ_ALOUD_ACTION_ID = 0x424F4252
        const val MAX_ANCHOR_MATCHES = 50
    }
}

internal fun chooseStableSelectionLocator(selected: Locator, matches: List<Locator>): Locator? {
    val candidates = stableSelectionCandidateIndexes(
        selected.href.toString(),
        matches.map { it.href.toString() },
    ).map(matches::get)
    return candidates.maxByOrNull { candidate -> highlightLocatorScore(selected, candidate) }
}

internal fun stableSelectionCandidateIndexes(selectedHref: String, matchHrefs: List<String>): List<Int> {
    val sameResource = matchHrefs.indices.filter { matchHrefs[it] == selectedHref }
    return sameResource.ifEmpty { if (matchHrefs.size == 1) listOf(0) else emptyList() }
}

internal suspend fun awaitNavigatorFrame(view: View) = suspendCancellableCoroutine { continuation ->
    view.postOnAnimation { if (continuation.isActive) continuation.resume(Unit) }
}
