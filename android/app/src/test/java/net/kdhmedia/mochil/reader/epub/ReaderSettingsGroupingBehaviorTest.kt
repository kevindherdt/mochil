package net.kdhmedia.mochil.reader.epub

import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.test.assertCountEquals
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.onAllNodesWithText
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.onNodeWithContentDescription
import androidx.compose.ui.test.performClick
import org.junit.Assert.assertEquals
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config

/**
 * The Settings panel used to be a single flat FlowRow/Column stack of ~12 unlabeled toggle
 * buttons. That crowds the panel and gives controls like "A-"/"A+" no textual meaning. This test
 * pins the fixed structure: settings are split into labelled tab sections (Text, Layout,
 * Appearance) mirroring the web reader's reorganized settings, with only the selected
 * section's controls shown at once, and the font-size controls carry descriptive labels.
 */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35])
class ReaderSettingsGroupingBehaviorTest {
    @get:Rule val compose = createComposeRule()

    @Test fun settingsAreGroupedIntoLabelledTabsShowingOneSectionAtATime() {
        compose.setContent {
            var preferences by mutableStateOf(ReaderPreferences())
            MaterialTheme {
                EpubReaderChrome(
                    visible = true, title = "Book", progression = null, readingStatistics = ReadingStatistics(),
                    preferences = preferences, customFonts = emptyList(), onPreferences = { preferences = it(preferences) },
                    chapters = emptyList(), onChapter = {}, searchState = EpubSearchState(), onSearch = {}, onSearchResult = {},
                    bookmarks = emptyList(), annotations = emptyList(), annotationConflicts = emptyList(), currentBookmarked = false,
                    onToggleBookmark = {}, onBookmark = {}, onAnnotation = {}, onEditAnnotationNote = { _, _ -> },
                    onDeleteAnnotation = {}, onKeepLocalAnnotation = {}, onUseRemoteAnnotation = {},
                    annotationNotesAvailable = false, annotationNotesExporting = false, onExportAnnotationNotes = {},
                    readingSyncStatus = ReaderSyncStatus(), syncMessage = null, syncMessageIsError = false,
                    onDismiss = {}, onClose = {},
                )
            }
        }
        compose.onNodeWithText("Aa").performClick()

        // Default section (Layout) shows the paging control immediately, but not the Text
        // section's font-size controls.
        compose.onNodeWithText("Paged").assertIsDisplayed()

        // Switching to the Text tab reveals font-size controls (with descriptive labels) that
        // were hidden before.
        compose.onNodeWithText("Text").performClick()
        compose.onNodeWithContentDescription("Decrease font size").assertIsDisplayed()
        compose.onNodeWithContentDescription("Increase font size").assertIsDisplayed()

        // Switching to Appearance reveals theme/sharpness/status-bar controls grouped together.
        compose.onNodeWithText("Appearance").performClick()
        compose.onNodeWithText("Light").assertIsDisplayed()

        // Reading progress and status icons live in Appearance too; there is no Display tab.
        compose.onNodeWithText("Reading progress: none").assertIsDisplayed()
        compose.onNodeWithText("Status icons: off").assertIsDisplayed()
        compose.onAllNodesWithText("Display").assertCountEquals(0)
    }

    @Test fun readingProgressStepsNoneBarTimeFromTheAppearanceTab() {
        var preferences by mutableStateOf(ReaderPreferences())
        compose.setContent {
            MaterialTheme {
                EpubReaderChrome(
                    visible = true, title = "Book", progression = null, readingStatistics = ReadingStatistics(),
                    preferences = preferences, customFonts = emptyList(), onPreferences = { preferences = it(preferences) },
                    chapters = emptyList(), onChapter = {}, searchState = EpubSearchState(), onSearch = {}, onSearchResult = {},
                    bookmarks = emptyList(), annotations = emptyList(), annotationConflicts = emptyList(), currentBookmarked = false,
                    onToggleBookmark = {}, onBookmark = {}, onAnnotation = {}, onEditAnnotationNote = { _, _ -> },
                    onDeleteAnnotation = {}, onKeepLocalAnnotation = {}, onUseRemoteAnnotation = {},
                    annotationNotesAvailable = false, annotationNotesExporting = false, onExportAnnotationNotes = {},
                    readingSyncStatus = ReaderSyncStatus(), syncMessage = null, syncMessageIsError = false,
                    onDismiss = {}, onClose = {},
                )
            }
        }
        compose.onNodeWithText("Aa").performClick()
        compose.onNodeWithText("Appearance").performClick()
        compose.onNodeWithText("Reading progress: none").performClick()
        compose.onNodeWithText("Reading progress: bar").performClick()
        compose.onNodeWithText("Reading progress: time").performClick()
        compose.onNodeWithText("Reading progress: none").assertIsDisplayed()
        compose.onNodeWithText("Sharper text: off").performClick()
        compose.onNodeWithText("Sharper text: on").assertIsDisplayed()
        assertEquals(TextSharpness.On, preferences.sharpness)
    }

    @Test fun togglingAPreferenceInASectionPersistsAcrossTabSwitches() {
        compose.setContent {
            var preferences by mutableStateOf(ReaderPreferences())
            MaterialTheme {
                EpubReaderChrome(
                    visible = true, title = "Book", progression = null, readingStatistics = ReadingStatistics(),
                    preferences = preferences, customFonts = emptyList(), onPreferences = { preferences = it(preferences) },
                    chapters = emptyList(), onChapter = {}, searchState = EpubSearchState(), onSearch = {}, onSearchResult = {},
                    bookmarks = emptyList(), annotations = emptyList(), annotationConflicts = emptyList(), currentBookmarked = false,
                    onToggleBookmark = {}, onBookmark = {}, onAnnotation = {}, onEditAnnotationNote = { _, _ -> },
                    onDeleteAnnotation = {}, onKeepLocalAnnotation = {}, onUseRemoteAnnotation = {},
                    annotationNotesAvailable = false, annotationNotesExporting = false, onExportAnnotationNotes = {},
                    readingSyncStatus = ReaderSyncStatus(), syncMessage = null, syncMessageIsError = false,
                    onDismiss = {}, onClose = {},
                )
            }
        }
        compose.onNodeWithText("Aa").performClick()
        compose.onNodeWithText("Layout").performClick()
        compose.onNodeWithText("Paged").performClick()
        compose.onNodeWithText("Scroll").assertIsDisplayed()

        // Leaving and returning to the Layout tab keeps the toggled state (scroll mode).
        compose.onNodeWithText("Text").performClick()
        compose.onNodeWithText("Layout").performClick()
        compose.onNodeWithText("Scroll").assertIsDisplayed()
    }
}
