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

        // The web reader's two tabs, as far as the phone has settings for them: Typography sets the text,
        // Display & theme sets the page around it. Narration has no settings of its own — its bar is above this sheet.
        compose.onNodeWithText("Typography").assertIsDisplayed()
        compose.onNodeWithText("Display & theme").assertIsDisplayed()
        // No Narration tab when this book cannot be read aloud: an empty tab is a dead one.
        compose.onAllNodesWithText("Narration").assertCountEquals(0)

        // Typography opens on the font-size controls, with descriptive labels.
        compose.onNodeWithContentDescription("Decrease font size").assertIsDisplayed()
        compose.onNodeWithContentDescription("Increase font size").assertIsDisplayed()

        // The page's own settings — paging, theme, sharpness, progress, status icons — share Display & theme.
        compose.onNodeWithText("Display & theme").performClick()
        compose.onNodeWithText("Paged").assertIsDisplayed()
        compose.onNodeWithText("Light").assertIsDisplayed()
        compose.onNodeWithText("Reading progress: none").assertIsDisplayed()
        compose.onNodeWithText("Status icons: off").assertIsDisplayed()
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
        compose.onNodeWithText("Display & theme").performClick()
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
        compose.onNodeWithText("Display & theme").performClick()
        compose.onNodeWithText("Paged").performClick()
        compose.onNodeWithText("Scroll").assertIsDisplayed()

        // Leaving and returning keeps the toggled state (scroll mode).
        compose.onNodeWithText("Typography").performClick()
        compose.onNodeWithText("Display & theme").performClick()
        compose.onNodeWithText("Scroll").assertIsDisplayed()
    }

    @Test
    fun aNarratableBookGetsItsNarrationTabWithTheSameSettingsAsTheBar() {
        compose.setContent {
            MaterialTheme {
                EpubReaderChrome(
                    visible = true, title = "Book", progression = null, readingStatistics = ReadingStatistics(),
                    preferences = ReaderPreferences(), customFonts = emptyList(), onPreferences = {},
                    chapters = emptyList(), onChapter = {}, searchState = EpubSearchState(), onSearch = {}, onSearchResult = {},
                    bookmarks = emptyList(), annotations = emptyList(), annotationConflicts = emptyList(), currentBookmarked = false,
                    onToggleBookmark = {}, onBookmark = {}, onAnnotation = {}, onEditAnnotationNote = { _, _ -> },
                    onDeleteAnnotation = {}, onKeepLocalAnnotation = {}, onUseRemoteAnnotation = {},
                    annotationNotesAvailable = false, annotationNotesExporting = false, onExportAnnotationNotes = {},
                    readingSyncStatus = ReaderSyncStatus(), syncMessage = null, syncMessageIsError = false,
                    onDismiss = {}, onClose = {},
                    narrationAvailable = true,
                    narrationSettings = {
                        EpubNarrationSettings(
                            ui = net.kdhmedia.mochil.narration.NarrationUi(available = true, speeds = listOf(1f, 1.5f), speed = 1.5f, sleepText = ""),
                            onVoice = {}, onSpeed = {}, onSleep = {}, onStop = {},
                        )
                    },
                )
            }
        }
        compose.onNodeWithText("Aa").performClick()
        compose.onNodeWithText("Narration").performClick()
        compose.onNodeWithText("Speed").assertIsDisplayed()
        compose.onNodeWithText("1.5× speed").assertIsDisplayed()
        compose.onNodeWithText("Sleep timer").assertIsDisplayed()
    }
}
