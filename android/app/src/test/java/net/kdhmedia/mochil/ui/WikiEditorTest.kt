package net.kdhmedia.mochil.ui

import androidx.compose.ui.text.TextRange
import androidx.compose.ui.text.input.TextFieldValue
import org.junit.Assert.assertEquals
import org.junit.Test

class WikiEditorTest {
    @Test fun `block formatting starts at the current line instead of the cursor`() {
        val result = applyWikiLinePrefix(TextFieldValue("Before\nPlan text\nAfter", TextRange(12)), "# ", "Heading")
        assertEquals("Before\n# Plan text\nAfter", result.text)
    }

    @Test fun `block formatting prefixes every selected line`() {
        val result = applyWikiLinePrefix(TextFieldValue("one\ntwo\nthree", TextRange(0, 7)), "> ", "quote")
        assertEquals("> one\n> two\nthree", result.text)
    }

    @Test fun `block formatting inserts and selects a placeholder on an empty line`() {
        val result = applyWikiLinePrefix(TextFieldValue("", TextRange.Zero), "- [ ] ", "task")
        assertEquals("- [ ] task", result.text)
        assertEquals(TextRange(6, 10), result.selection)
    }

    @Test fun `markup wraps the current selection and leaves the cursor after it`() {
        val result = applyWikiMarkup(TextFieldValue("make this bold", TextRange(5, 9)), "**", "**", "bold text")
        assertEquals("make **this** bold", result.text)
        assertEquals(TextRange(13), result.selection)
    }

    @Test fun `markup selects its placeholder when there is no selection`() {
        val result = applyWikiMarkup(TextFieldValue("", TextRange.Zero), "[[", "]]", "Page name")
        assertEquals("[[Page name]]", result.text)
        assertEquals(TextRange(2, 11), result.selection)
    }

    @Test fun `fenced code wraps complete selected lines`() {
        val result = applyWikiFencedBlock(TextFieldValue("Before\nfirst\nsecond\nAfter", TextRange(9, 19)), "kotlin")
        assertEquals("Before\n```kotlin\nfirst\nsecond\n```\nAfter", result.text)
        assertEquals(TextRange(33), result.selection)
    }

    @Test fun `fenced code creates an editable placeholder on an empty line`() {
        val result = applyWikiFencedBlock(TextFieldValue("", TextRange.Zero))
        assertEquals("```\ncode\n```", result.text)
        assertEquals(TextRange(4, 8), result.selection)
    }
}
