package net.kdhmedia.mochil.ui

import org.junit.Assert.assertEquals
import org.junit.Test

class DashboardJumpBackInTest {
    @Test
    fun kindsAreMixedInTurnAndEachKeepsItsOwnOrder() {
        assertEquals(
            listOf("book1", "video1", "audio1", "book2", "video2", "video3"),
            dashboardJumpBackIn(listOf("book1", "book2"), listOf("video1", "video2", "video3"), listOf("audio1")),
        )
        assertEquals(emptyList<String>(), dashboardJumpBackIn(emptyList<String>(), emptyList()))
    }
}
