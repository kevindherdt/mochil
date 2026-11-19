package net.kdhmedia.mochil.readalong

import net.kdhmedia.mochil.catalogue.LibraryState
import net.kdhmedia.mochil.ui.openingText
import org.junit.Assert.assertEquals
import org.junit.Test

class OpeningTextTest {
    private fun state(done: Long, total: Long) = LibraryState(openingBookId = "b", openingTitle = "Leviathan Wakes", openingDone = done, openingTotal = total)

    @Test fun saysOpeningBeforeTheFirstBytes() = assertEquals("Opening Leviathan Wakes…", openingText(state(0, 0)))
    @Test fun showsHowMuchOfTheBookHasArrived() = assertEquals("Opening Leviathan Wakes: 120.0 MB of 549.0 MB", openingText(state(125_829_120, 575_668_224)))
    @Test fun showsJustTheSizeWhenTheTotalIsNotKnown() = assertEquals("Opening Leviathan Wakes: 512 KB", openingText(state(524_288, 0)))
}
