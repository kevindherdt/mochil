package net.kdhmedia.mochil.ui

import androidx.compose.runtime.saveable.Saver
import net.kdhmedia.mochil.catalogue.LibrarySummary
import net.kdhmedia.mochil.ui.kit.DockPlace
import net.kdhmedia.mochil.ui.kit.LucideIcons

/**
 * Where the phone shell can be. Library is the start, as on the web (plan §5.1: "/" is always a library shelf).
 * With a vault selected, Library *is* Notes (the dock follows the library's kind). The Notes destination exists only for
 * the moment before a vault is selected and for the "no vault" screen.
 * Collections and Sources are opened from Library's menu and belong to it.
 */
internal enum class MobileShellDestination { Library, Discover, Notes, Collections, Sources, Settings }

internal val MOBILE_SHELL_START = MobileShellDestination.Library

/**
 * Back leaves the app from a book library only. Every other root returns to Library, and so does a vault: Notes is the
 * Library screen with a vault selected, so Back from it has to select a book library or the same vault stays up.
 */
internal fun handlesMobileShellBack(destination: MobileShellDestination, vaultOpen: Boolean = false): Boolean =
    destination != MobileShellDestination.Library || vaultOpen

/** Whether a vault is on screen and there is a book library to go back to (an account of only vaults just leaves). */
internal fun vaultBackAvailable(libraries: List<LibrarySummary>, selected: LibrarySummary?, lastLibraryId: String?): Boolean =
    selected != null && isVault(selected) && libraryTabTarget(libraries, selected, lastLibraryId) != null

/** Saved by name, so renaming or dropping a destination (Home became Discover) falls back to the start instead of crashing the restore. */
internal val MobileShellDestinationSaver: Saver<MobileShellDestination, Any> = Saver(
    save = { it.name },
    restore = { saved -> MobileShellDestination.entries.firstOrNull { it.name == saved } ?: MOBILE_SHELL_START },
)

@Suppress("UNUSED_PARAMETER")
internal fun mobileShellBackTarget(destination: MobileShellDestination): MobileShellDestination =
    MobileShellDestination.Library

internal object DockIds {
    const val LIBRARY = "library"
    const val DISCOVER = "discover"
    const val AI = "ai"
    const val NOTES = "notes"
    const val SETTINGS = "settings"
}

/** The dock draws Lucide at 1.8 where the glyphs default to 2 (web/src/shell/Dock.tsx: strokeWidth 1.8). */
private const val DOCK_STROKE = 1.8f

private val LIBRARY_PLACE by lazy { DockPlace(DockIds.LIBRARY, "Library", LucideIcons.withStroke("book-open", DOCK_STROKE)) }
private val DISCOVER_PLACE by lazy { DockPlace(DockIds.DISCOVER, "Discover", LucideIcons.withStroke("compass", DOCK_STROKE)) }
private val AI_PLACE by lazy { DockPlace(DockIds.AI, "AI", LucideIcons.withStroke("sparkles", DOCK_STROKE), launcher = true) }
private val NOTES_PLACE by lazy { DockPlace(DockIds.NOTES, "Notes", LucideIcons.withStroke("notebook-text", DOCK_STROKE)) }
private val SETTINGS_PLACE by lazy { DockPlace(DockIds.SETTINGS, "Settings", LucideIcons.withStroke("settings", DOCK_STROKE)) }
private val WITH_CHAT by lazy { listOf(LIBRARY_PLACE, DISCOVER_PLACE, AI_PLACE, NOTES_PLACE, SETTINGS_PLACE) }
private val WITHOUT_CHAT by lazy { listOf(LIBRARY_PLACE, DISCOVER_PLACE, NOTES_PLACE, SETTINGS_PLACE) }

/** The web's order. The chat place exists only when this account can reach it, so nothing in the dock leads nowhere. */
internal fun dockPlaces(chatAvailable: Boolean): List<DockPlace> = if (chatAvailable) WITH_CHAT else WITHOUT_CHAT

/** The chat is its own activity, so its place launches it and has no destination here. */
internal fun dockDestination(id: String): MobileShellDestination? = when (id) {
    DockIds.LIBRARY -> MobileShellDestination.Library
    DockIds.DISCOVER -> MobileShellDestination.Discover
    DockIds.NOTES -> MobileShellDestination.Notes
    DockIds.SETTINGS -> MobileShellDestination.Settings
    else -> null
}

/**
 * The title row the shell draws above a page that has no header of its own: the way back, the name, and the tools the page hands up.
 * Library draws its pills, Discover and Settings their own headings, and no page keeps the old drop-down switcher: the dock is the switcher.
 */
internal fun shellPageTitle(destination: MobileShellDestination): String? = when (destination) {
    MobileShellDestination.Collections -> "Collections"
    // Sources is not here: SourceHubScreen and its OPDS / WebDAV pages draw their own back arrow and title.
    else -> null
}

internal fun isVault(library: LibrarySummary): Boolean = library.kind.equals("vault", ignoreCase = true)

/** A vault opened from the library menu is still Notes, so the dock follows what is on screen, not how it was reached. */
internal fun dockActiveId(destination: MobileShellDestination, selected: LibrarySummary?): String = when (destination) {
    MobileShellDestination.Library -> if (selected != null && isVault(selected)) DockIds.NOTES else DockIds.LIBRARY
    MobileShellDestination.Collections, MobileShellDestination.Sources -> DockIds.LIBRARY
    MobileShellDestination.Discover -> DockIds.DISCOVER
    MobileShellDestination.Notes -> DockIds.NOTES
    MobileShellDestination.Settings -> DockIds.SETTINGS
}

/** The vault Notes shows: the selected one if it is a vault, otherwise the first. Null when the server has none. */
internal fun notesLibrary(libraries: List<LibrarySummary>, selected: LibrarySummary?): LibrarySummary? =
    selected?.takeIf(::isVault) ?: libraries.firstOrNull(::isVault)

/** What the Library tab opens: the library already open, else the last one used, else the first that is not a vault. */
internal fun libraryTabTarget(libraries: List<LibrarySummary>, selected: LibrarySummary?, lastLibraryId: String?): LibrarySummary? =
    selected?.takeUnless(::isVault)
        ?: libraries.firstOrNull { it.id == lastLibraryId && !isVault(it) }
        ?: libraries.firstOrNull { !isVault(it) }

/**
 * The Notes screen with nothing to show: the server has libraries, none of them a vault (or this device hides it). With no
 * libraries at all the list has simply not loaded, and the library screen says so itself.
 */
internal fun showsNotesEmpty(libraries: List<LibrarySummary>): Boolean = libraries.isNotEmpty() && libraries.none(::isVault)

/** The wide layout has the drawer instead, and a bar of tabs under a keyboard is only in the way. */
internal fun dockVisible(expandedNavigation: Boolean, keyboardVisible: Boolean): Boolean =
    !expandedNavigation && !keyboardVisible
