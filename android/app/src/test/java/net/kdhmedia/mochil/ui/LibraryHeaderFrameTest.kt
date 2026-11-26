package net.kdhmedia.mochil.ui

import androidx.compose.material3.Text
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.assertIsSelected
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithContentDescription
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import net.kdhmedia.mochil.catalogue.BookStatusFilter
import net.kdhmedia.mochil.catalogue.LibraryState
import net.kdhmedia.mochil.catalogue.LibrarySummary
import net.kdhmedia.mochil.catalogue.PublicationGrouping
import net.kdhmedia.mochil.ui.kit.LucideIcons
import net.kdhmedia.mochil.ui.theme.AppPalette
import net.kdhmedia.mochil.ui.theme.AppThemeMode
import net.kdhmedia.mochil.ui.theme.AppThemeSelection
import net.kdhmedia.mochil.ui.theme.MochilTheme
import org.junit.Assert.assertEquals
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import org.robolectric.annotation.GraphicsMode

/** The plumbing between a library screen and its header: the screen's own buttons travel up, and the chips follow the view. */
@RunWith(RobolectricTestRunner::class)
@GraphicsMode(GraphicsMode.Mode.NATIVE)
@Config(sdk = [35], qualifiers = "w390dp-h844dp-xxhdpi")
class LibraryHeaderFrameTest {
    @get:Rule val compose = createComposeRule()

    private val books = LibrarySummary("books", "Books", "books", 40)
    private val anime = LibrarySummary("anime", "Anime", "video", 9)
    private val music = LibrarySummary("music", "Music", "music", 500)
    private val flat = LibraryState(libraries = listOf(books, anime, music), selected = books, grouping = PublicationGrouping.Books, initialLoading = false)

    private fun frame(state: LibraryState, expanded: Boolean = false, onStatus: (BookStatusFilter) -> Unit = {}, onSelect: (LibrarySummary) -> Unit = {}, content: @androidx.compose.runtime.Composable () -> Unit) {
        compose.setContent {
            MochilTheme(AppThemeSelection(AppPalette.Tide, AppThemeMode.Dark), glow = false) {
                CompositionLocalProvider(LocalExpandedMobileNavigation provides expanded) { LibraryHeaderFrame(state, onSelect, {}, onStatus, content) }
            }
        }
    }

    @Test fun `the buttons a screen hands up appear in the header and work`() {
        var taps = 0
        frame(flat) { HeaderActions(listOf(HeaderAction(LucideIcons.Layers, "View options", { taps++ }))) }
        compose.onNodeWithContentDescription("View options").assertIsDisplayed()
        compose.onNodeWithContentDescription("View options").performClick()
        assertEquals(1, taps)
    }

    @Test fun `the status chips show on the plain list and a tap reaches the view model`() {
        var chosen: BookStatusFilter? = null
        frame(flat.copy(status = BookStatusFilter.Unread), onStatus = { chosen = it }) { Text("shelf") }
        compose.onNodeWithText("Unread").assertIsSelected()
        compose.onNodeWithText("In Progress").performClick()
        assertEquals(BookStatusFilter.InProgress, chosen)
    }

    @Test fun `a grouped view, a download list and an open book have no chips`() {
        frame(flat.copy(grouping = PublicationGrouping.Series)) { Text("shelf") }
        compose.onNodeWithText("In Progress").assertDoesNotExist()
    }

    @Test fun `a video shelf has pills and its buttons but no chips`() {
        frame(flat.copy(selected = anime)) { HeaderActions(listOf(HeaderAction(LucideIcons.Search, "Search videos", {}))) }
        compose.onNodeWithText("Anime").assertIsSelected()
        compose.onNodeWithContentDescription("Search videos").assertIsDisplayed()
        compose.onNodeWithText("In Progress").assertDoesNotExist()
    }

    @Test fun `choosing a pill opens that library`() {
        var opened: LibrarySummary? = null
        frame(flat, onSelect = { opened = it }) { Text("shelf") }
        compose.onNodeWithText("Music").performClick()
        assertEquals(music, opened)
    }

    @Test fun `the screen below the header is drawn`() {
        frame(flat) { Text("the covers") }
        compose.onNodeWithText("the covers").assertIsDisplayed()
    }

    @Test fun `the wide layout has no pills, and its chips sit on a row of their own`() {
        var chosen: BookStatusFilter? = null
        frame(flat, expanded = true, onStatus = { chosen = it }) { Text("shelf") }
        compose.onNodeWithText("Music").assertDoesNotExist()
        compose.onNodeWithText("Unread").performClick()
        assertEquals(BookStatusFilter.Unread, chosen)
    }
}
