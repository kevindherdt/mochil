package net.kdhmedia.mochil.ui

import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import net.kdhmedia.mochil.catalogue.BookStatusFilter
import net.kdhmedia.mochil.catalogue.PublicationGrouping
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

@RunWith(RobolectricTestRunner::class)
@GraphicsMode(GraphicsMode.Mode.NATIVE)
@Config(sdk = [35], qualifiers = "w390dp-h844dp-xxhdpi")
class PublicationEmptyTest {
    @get:Rule val compose = createComposeRule()

    @Test fun `an empty shelf says what is missing, in the web's words`() {
        assertEquals("This shelf has no books.", publicationEmptyMessage(PublicationGrouping.Books, false, BookStatusFilter.All))
        assertEquals("No books in progress.", publicationEmptyMessage(PublicationGrouping.Books, false, BookStatusFilter.InProgress))
        assertEquals("Every item here has been opened.", publicationEmptyMessage(PublicationGrouping.Recent, false, BookStatusFilter.Unread))
    }

    @Test fun `the older views keep their own lines`() {
        assertEquals("Books you start will appear here.", publicationEmptyMessage(PublicationGrouping.Continue, false, BookStatusFilter.All))
        assertEquals("No recently added books were found.", publicationEmptyMessage(PublicationGrouping.Recent, false, BookStatusFilter.All))
    }

    @Test fun `a search that found nothing is the reason, whatever chip is on`() {
        assertEquals("Nothing matches \"dune\".", publicationEmptyMessage(PublicationGrouping.Books, false, BookStatusFilter.InProgress, "dune"))
    }

    @Test fun `downloads only is about downloads, whatever the chip says`() {
        assertEquals("No downloaded books on this shelf.", publicationEmptyMessage(PublicationGrouping.Books, true, BookStatusFilter.All))
        assertEquals("No downloaded books on this shelf.", publicationEmptyMessage(PublicationGrouping.Books, true, BookStatusFilter.InProgress))
    }

    @Test fun `an empty state is one muted line, and the one action when there is one`() {
        var taps = 0
        compose.setContent { MochilTheme(AppThemeSelection(AppPalette.Tide, AppThemeMode.Dark), glow = false) { PublicationEmpty("No books in progress.", "Show all") { taps++ } } }
        compose.onNodeWithText("No books in progress.").assertExists()
        compose.onNodeWithText("Show all").performClick()
        assertEquals(1, taps)
    }

    @Test fun `without an action there is no button`() {
        compose.setContent { MochilTheme(AppThemeSelection(AppPalette.Tide, AppThemeMode.Dark), glow = false) { PublicationEmpty("This shelf has no books.") } }
        compose.onNodeWithText("This shelf has no books.").assertExists()
        compose.onNodeWithText("Show all").assertDoesNotExist()
    }
}
