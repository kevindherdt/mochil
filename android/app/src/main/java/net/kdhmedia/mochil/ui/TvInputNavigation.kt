package net.kdhmedia.mochil.ui

import android.view.KeyEvent

internal enum class TvLibraryDirection { Previous, Next }

internal enum class TvPublicationBackTarget { Collection, Series, Author, Home }

internal fun tvLibraryDirection(keyCode: Int): TvLibraryDirection? = when (keyCode) {
    KeyEvent.KEYCODE_BUTTON_L1,
    KeyEvent.KEYCODE_BUTTON_L2 -> TvLibraryDirection.Previous
    KeyEvent.KEYCODE_BUTTON_R1,
    KeyEvent.KEYCODE_BUTTON_R2 -> TvLibraryDirection.Next
    else -> null
}

internal fun adjacentLibraryIndex(currentIndex: Int, count: Int, direction: TvLibraryDirection): Int {
    if (count <= 1) return -1
    if (currentIndex !in 0 until count) {
        return when (direction) {
            TvLibraryDirection.Previous -> count - 1
            TvLibraryDirection.Next -> 0
        }
    }
    return when (direction) {
        TvLibraryDirection.Previous -> (currentIndex - 1 + count) % count
        TvLibraryDirection.Next -> (currentIndex + 1) % count
    }
}

/** Stable hierarchy identity used for focus recovery without reacting to filters or paging. */
internal fun tvHierarchyKey(vararg levels: String?): String = levels
    .joinToString("|") { value -> value.orEmpty().let { "${it.length}:$it" } }

/** Content replacement identity: switching shelves or hierarchy replaces every focus target. */
internal fun tvContentRouteKey(libraryId: String?, hierarchyKey: String): String =
    tvHierarchyKey(libraryId, hierarchyKey)

internal fun tvPublicationBackTarget(
    hasCollection: Boolean,
    hasSeries: Boolean,
    hasAuthor: Boolean,
): TvPublicationBackTarget = when {
    hasCollection -> TvPublicationBackTarget.Collection
    hasSeries -> TvPublicationBackTarget.Series
    hasAuthor -> TvPublicationBackTarget.Author
    else -> TvPublicationBackTarget.Home
}

internal class TvNowPlayingFocusSession {
    private var presented = false
    private var queueWasVisible = false

    /**
     * Focus transport on entry and when Queue closes. Track transitions and
     * ordinary recompositions must never pull the remote away from its owner.
     */
    fun shouldFocusTransport(queueVisible: Boolean): Boolean {
        val shouldFocus = !queueVisible && (!presented || queueWasVisible)
        presented = true
        queueWasVisible = queueVisible
        return shouldFocus
    }
}

/** Both the on-screen and system Back follow the same queue-first hierarchy. */
internal fun navigateTvNowPlayingBack(queueVisible: Boolean, closeQueue: () -> Unit, onBack: () -> Unit) {
    if (queueVisible) closeQueue() else onBack()
}

/** What one direction press does while the whole description is open over the page. */
internal enum class TvSynopsisMove { ScrollBack, ScrollForward, ToClose, ToStory }

/**
 * The description overlay shares its window with the page it covers, so it has to answer every arrow itself: an
 * unanswered one is resolved by the focus system against the page underneath, which is how the remote used to end up
 * on Play or an episode while the synopsis was still open. Returns null when the press is still consumed but does
 * nothing (there is nowhere left to go in that direction).
 */
internal fun tvSynopsisMove(
    action: TvRemoteAction,
    closeFocused: Boolean,
    canScrollBack: Boolean,
    canScrollForward: Boolean,
): TvSynopsisMove? = when (action) {
    // Down reads the text; once there is nothing left below, the next stop in reading order is Close.
    TvRemoteAction.Down -> when {
        closeFocused -> null
        canScrollForward -> TvSynopsisMove.ScrollForward
        else -> TvSynopsisMove.ToClose
    }
    TvRemoteAction.Up -> when {
        closeFocused -> TvSynopsisMove.ToStory
        canScrollBack -> TvSynopsisMove.ScrollBack
        else -> null
    }
    // Left/Right is the way between the text and the button that ends it, and never leaves the overlay.
    TvRemoteAction.Left, TvRemoteAction.Right -> if (closeFocused) TvSynopsisMove.ToStory else TvSynopsisMove.ToClose
    TvRemoteAction.Select, TvRemoteAction.PlayPause -> null
}
