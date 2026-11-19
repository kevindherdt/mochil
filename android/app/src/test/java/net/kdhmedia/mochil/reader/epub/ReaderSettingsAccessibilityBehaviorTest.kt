package net.kdhmedia.mochil.reader.epub

import androidx.compose.material3.MaterialTheme
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithContentDescription
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35])
class ReaderSettingsAccessibilityBehaviorTest {
    @get:Rule val compose = createComposeRule()

    @Test fun settingsControlHasDescriptiveLabelAndOpensSettings() {
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
                )
            }
        }
        compose.onNodeWithContentDescription("Reader settings").performClick()
        compose.onNodeWithText("Paged").assertIsDisplayed()
    }
}
