package net.kdhmedia.mochil.ui

import java.nio.file.Files
import java.nio.file.Paths
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * Video search sent every character straight through the ViewModel, so the field was re-written from a query that
 * arrives a frame late and a fast typist lost characters to the older value. The book and music shelves keep a local
 * draft and submit it on the IME action; video now does the same. Keystroke timing in a Compose text field cannot be
 * asserted without instrumentation, so the shape is pinned here.
 */
class VideoSearchDraftSourceTest {
    private val source = Files.readString(Paths.get("src/main/java/net/kdhmedia/mochil/ui/VideoScreen.kt"))

    @Test fun `the video search field keeps a local draft`() {
        assertTrue("the field must hold its own draft", source.contains("var draft by remember(state.libraryId, state.searchOpen)"))
        assertTrue("the field must draw the draft", source.contains("value = draft,"))
        assertFalse("keystrokes must not go straight to the ViewModel", source.contains("onValueChange = onSearch"))
    }

    @Test fun `the draft is submitted on the IME search action like the other shelves`() {
        assertTrue(source.contains("imeAction = ImeAction.Search"))
        assertTrue(source.contains("onSearch(draft)"))
    }
}
