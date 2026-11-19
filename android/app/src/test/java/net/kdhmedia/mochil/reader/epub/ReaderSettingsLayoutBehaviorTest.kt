package net.kdhmedia.mochil.reader.epub

import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.width
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.semantics.SemanticsActions
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.compose.ui.text.TextLayoutResult
import androidx.compose.ui.unit.dp
import org.junit.Assert.assertEquals
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35])
class ReaderSettingsLayoutBehaviorTest {
    @get:Rule val compose = createComposeRule()

    @Test fun modeLabelFitsOnOneLineAndStillTogglesAtNarrowWidth() {
        compose.setContent {
            var preferences by androidx.compose.runtime.remember { mutableStateOf(ReaderPreferences()) }
            MaterialTheme {
                Box(Modifier.width(320.dp)) {
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
        }
        compose.onNodeWithText("Aa").performClick()
        assertSingleLine("Paged")
        compose.onNodeWithText("Paged").performClick()
        assertSingleLine("Scroll")
    }

    private fun assertSingleLine(label: String) {
        val node = compose.onNodeWithText(label, useUnmergedTree = true).fetchSemanticsNode()
        val layouts = mutableListOf<TextLayoutResult>()
        node.config[SemanticsActions.GetTextLayoutResult].action!!.invoke(layouts)
        assertEquals("$label should not wrap", 1, layouts.single().lineCount)
    }
}
