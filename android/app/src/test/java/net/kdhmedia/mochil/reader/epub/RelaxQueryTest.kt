package net.kdhmedia.mochil.reader.epub

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotEquals
import org.junit.Test

/**
 * Search returned "No matches found" for `want to kill me?` in a book whose page reads
 * `"Yeah. He want to kill me, Julie?"`. Readium matches literally, so the remembered question mark
 * is what made it fail -- not the text. The retry strips punctuation instead of asking the reader to
 * guess it, and these cases pin what "the same words" means.
 */
class RelaxQueryTest {

    @Test
    fun aTrailingQuestionMarkBecomesAWordBreak() {
        assertEquals("want to kill me", relaxQuery("want to kill me?"))
    }

    @Test
    fun quotesCommasAndEllipsesCollapseToSingleSpaces() {
        assertEquals("He want to kill me Julie", relaxQuery("“He want to kill me, Julie?”"))
        assertEquals("and then nothing", relaxQuery("and then… nothing"))
        assertEquals("a b c", relaxQuery("a  ,  b ... c"))
    }

    @Test
    fun caseIsLeftAlone() {
        // Readium's search is case-insensitive; rewriting case here would only risk changing what
        // the reader typed for no gain.
        assertEquals("Want To Kill Me", relaxQuery("Want To Kill Me?"))
    }

    @Test
    fun apostrophesInsideWordsSurvive() {
        // Stripping the apostrophe would search for "don t", which matches nothing.
        assertEquals("Don't go", relaxQuery("Don't go!"))
    }

    @Test
    fun aQueryThatIsAlreadyPlainIsUntouched() {
        val plain = "want to kill me"
        assertEquals(plain, relaxQuery(plain))
    }

    @Test
    fun punctuationOnlyBecomesEmptySoNoSecondScanRuns() {
        assertEquals("", relaxQuery("?!…"))
    }

    @Test
    fun theRelaxedQueryActuallyDiffersForTheFailingCase() {
        // The point of the retry: if these were equal the search would scan twice for nothing.
        assertNotEquals("want to kill me?", relaxQuery("want to kill me?"))
    }
}
