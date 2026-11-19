package net.kdhmedia.mochil.wiki

import org.junit.Assert.assertEquals
import org.junit.Test

class WikiTableBoundsTest {
    @Test fun `very wide rows retain only the first twenty cells`() {
        val expected = List(20) { "Column $it" }
        val row = "|" + expected.joinToString("|") + "|" + "surplus|".repeat(100_000)
        assertEquals(expected, parseWikiTableRow(row))
    }

    @Test fun `escaped pipes do not consume column capacity`() {
        val cells = List(20) { "Cell $it \\| value" }
        val row = cells.joinToString("|") + "|discard"
        assertEquals(List(20) { "Cell $it | value" }, parseWikiTableRow(row))
    }

    @Test fun `exact column limit retains final cell and trailing backslash`() {
        val cells = List(19) { "Cell $it" } + "Final\\"
        assertEquals(cells, parseWikiTableRow(cells.joinToString("|")))
    }
}
