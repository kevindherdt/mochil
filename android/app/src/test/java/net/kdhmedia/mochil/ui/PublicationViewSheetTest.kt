package net.kdhmedia.mochil.ui

import net.kdhmedia.mochil.catalogue.PublicationGrouping
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class PublicationViewSheetTest {
    @Test
    fun theUngroupedListIsOnForBothItsOrdersAndTheOldRecentView() {
        assertTrue(publicationGroupSelected(PublicationGrouping.Books, PublicationGrouping.Books))
        assertTrue(publicationGroupSelected(PublicationGrouping.Recent, PublicationGrouping.Books))
        assertFalse(publicationGroupSelected(PublicationGrouping.Series, PublicationGrouping.Books))
    }

    @Test
    fun theChoicesAreNamedForWhatYouSee() {
        assertEquals(listOf("Series", "Authors", "None"), publicationGroupChoices.map(::publicationGroupLabel))
        assertEquals(listOf("Title", "Author", "Recently added"), publicationOrderChoices.map { it.second })
    }
}
