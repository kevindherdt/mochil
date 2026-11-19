package net.kdhmedia.mochil.ui

import androidx.compose.ui.text.TextRange
import androidx.compose.ui.text.input.TextFieldValue
import net.kdhmedia.mochil.wiki.MAX_WIKI_DRAFT_CHARS
import org.junit.Assert.*
import org.junit.Test

class WikiEditorLimitTest {
    @Test fun `oversized paste preserves previous text selection and composition`() {
        val previous = TextFieldValue("日本語\r\n", TextRange(1, 3), TextRange(0, 3))
        val pasted = TextFieldValue("x".repeat(MAX_WIKI_DRAFT_CHARS + 1))
        assertSame(previous, acceptWikiEditorValue(previous, pasted))
    }

    @Test fun `formatting at limit cannot truncate the existing document`() {
        val previous = TextFieldValue("x".repeat(MAX_WIKI_DRAFT_CHARS), TextRange(0, 1))
        for (next in listOf(
            applyWikiMarkup(previous, "**", "**", "bold"),
            applyWikiLinePrefix(previous, "# ", "Heading"),
            applyWikiFencedBlock(previous),
        )) assertSame(previous, acceptWikiEditorValue(previous, next))
    }

    @Test fun `valid boundary and deletion retain exact requested value`() {
        val previous = TextFieldValue("old")
        for (next in listOf(TextFieldValue(""), TextFieldValue("x".repeat(MAX_WIKI_DRAFT_CHARS)))) {
            assertSame(next, acceptWikiEditorValue(previous, next))
        }
    }
}
