package net.kdhmedia.mochil.catalogue

import org.junit.Assert.assertEquals
import org.junit.Test

class StartingViewTest {
    @Test
    fun aLibraryNeverOpensOnTheShortTripViews() {
        assertEquals(PublicationGrouping.Series, PublicationGrouping.Continue.startingView())
        assertEquals(PublicationGrouping.Series, PublicationGrouping.Collections.startingView())
    }

    @Test
    fun theFallbackIsTheDevicesOwnView() {
        assertEquals(PublicationGrouping.Books, PublicationGrouping.Continue.startingView(PublicationGrouping.Books))
        assertEquals(PublicationGrouping.Books, PublicationGrouping.Collections.startingView(PublicationGrouping.Books))
        assertEquals(PublicationGrouping.Authors, PublicationGrouping.Authors.startingView(PublicationGrouping.Books))
    }

    @Test
    fun theOtherViewsAreKept() {
        assertEquals(PublicationGrouping.Authors, PublicationGrouping.Authors.startingView())
        assertEquals(PublicationGrouping.Books, PublicationGrouping.Books.startingView())
        assertEquals(PublicationGrouping.Recent, PublicationGrouping.Recent.startingView())
    }
}
