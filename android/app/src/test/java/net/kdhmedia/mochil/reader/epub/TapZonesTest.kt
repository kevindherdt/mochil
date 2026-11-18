package net.kdhmedia.mochil.reader.epub

import org.junit.Assert.assertEquals
import org.junit.Test

class TapZonesTest {
    private fun at(zones: TapZones, share: Float, rtl: Boolean = false) = tapAction(zones, share * 1000f, 1000f, rtl)

    @Test fun withZonesOffEveryTapOpensTheMenu() {
        listOf(0f, 0.1f, 0.5f, 0.9f, 1f).forEach { assertEquals(TapAction.Menu, at(TapZones.Off, it)) }
    }

    @Test fun edgesTurnThePageOnlyAtTheOuterQuarters() {
        assertEquals(TapAction.Back, at(TapZones.Edges, 0.1f))
        assertEquals(TapAction.Menu, at(TapZones.Edges, 0.3f))
        assertEquals(TapAction.Menu, at(TapZones.Edges, 0.5f))
        assertEquals(TapAction.Menu, at(TapZones.Edges, 0.7f))
        assertEquals(TapAction.Forward, at(TapZones.Edges, 0.9f))
    }

    @Test fun oneHandGoesBackFromTheLeftFifthAndForwardEverywhereElse() {
        assertEquals(TapAction.Back, at(TapZones.OneHand, 0.1f))
        assertEquals(TapAction.Forward, at(TapZones.OneHand, 0.3f))
        assertEquals(TapAction.Forward, at(TapZones.OneHand, 0.95f))
    }

    @Test fun everyChoiceKeepsAWayToTheMenu() {
        TapZones.entries.forEach { zones ->
            val actions = (0..100).map { at(zones, it / 100f) }.toSet()
            assertEquals("$zones can reach the menu", true, TapAction.Menu in actions)
        }
    }

    @Test fun aBookReadRightToLeftSwapsTheSides() {
        assertEquals(TapAction.Forward, at(TapZones.Edges, 0.1f, rtl = true))
        assertEquals(TapAction.Back, at(TapZones.Edges, 0.9f, rtl = true))
        assertEquals(TapAction.Menu, at(TapZones.Edges, 0.5f, rtl = true))
    }

    @Test fun anUnmeasuredViewNeverTurnsThePage() = assertEquals(TapAction.Menu, tapAction(TapZones.Edges, 10f, 0f, false))

    @Test fun stepsThroughTheChoicesAndBackToOff() {
        assertEquals(listOf(TapZones.Edges, TapZones.OneHand, TapZones.Off), generateSequence(TapZones.Off) { it.next() }.drop(1).take(3).toList())
    }
}
