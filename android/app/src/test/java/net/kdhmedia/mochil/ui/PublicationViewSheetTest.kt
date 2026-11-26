package net.kdhmedia.mochil.ui

import net.kdhmedia.mochil.catalogue.PublicationGrouping
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import net.kdhmedia.mochil.catalogue.LibraryState
import net.kdhmedia.mochil.catalogue.PublicationLayout
import net.kdhmedia.mochil.ui.theme.AppPalette
import net.kdhmedia.mochil.ui.theme.AppThemeMode
import net.kdhmedia.mochil.ui.theme.AppThemeSelection
import net.kdhmedia.mochil.ui.theme.MochilTheme
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import org.robolectric.annotation.GraphicsMode

@RunWith(RobolectricTestRunner::class)
@GraphicsMode(GraphicsMode.Mode.NATIVE)
@Config(sdk = [35], qualifiers = "w390dp-h844dp-xxhdpi")
class PublicationViewSheetTest {
    @get:Rule val compose = createComposeRule()

    @Test
    fun theUngroupedListIsOnForBothItsOrdersAndTheOldRecentView() {
        assertTrue(publicationGroupSelected(PublicationGrouping.Books, PublicationGrouping.Books))
        assertTrue(publicationGroupSelected(PublicationGrouping.Recent, PublicationGrouping.Books))
        assertFalse(publicationGroupSelected(PublicationGrouping.Series, PublicationGrouping.Books))
    }

    @Test
    fun theChoicesAreNamedForWhatYouSee() {
        assertEquals(listOf("Series", "Authors", "All books"), publicationGroupChoices.map(::publicationGroupLabel))
        assertEquals(listOf("Title", "Author", "Recently added"), publicationOrderChoices.map { it.second })
    }

    private fun sheet(collections: (() -> Unit)?) {
        compose.setContent {
            MochilTheme(AppThemeSelection(AppPalette.Tide, AppThemeMode.Dark), glow = false) {
                PublicationViewSheet(LibraryState(grouping = PublicationGrouping.Books), onGrouping = {}, onSort = {}, onLayout = { _: PublicationLayout -> }, onOpenCollections = collections, onClose = {})
            }
        }
    }

    @Test
    fun collectionsAreOneTapAwayFromTheMenuWhenTheAccountHasThem() {
        var opened = 0
        sheet { opened++ }
        compose.onNodeWithText("Collections").performClick()
        assertEquals(1, opened)
    }

    @Test
    fun noCollectionsEntryWhenTheyAreHidden() {
        sheet(null)
        compose.onNodeWithText("Collections").assertDoesNotExist()
    }

    @Test
    fun inProgressIsAChipNowNotAMenuEntry() {
        sheet(null)
        // The sheet is really on screen, so the absence below means something.
        compose.onNodeWithText("Layout").assertExists()
        compose.onNodeWithText("Only books in progress").assertDoesNotExist()
    }
}
