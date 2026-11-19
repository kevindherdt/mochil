package net.kdhmedia.mochil.ui

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class TvQueueFocusTest {
    @Test fun `focus follows live position when queue changed while menu was open`() {
        assertEquals("next", tvQueueFocusAfterRemoval(
            listOf("new", "first", "target", "next"), listOf("new", "first", "next"), "target", { it },
        ))
        assertEquals("first", tvQueueFocusAfterRemoval(
            listOf("first", "target"), listOf("first"), "target", { it },
        ))
    }

    @Test fun `no op removal retains target and missing target has a safe fallback`() {
        assertEquals("target", tvQueueFocusAfterRemoval(listOf("target"), listOf("target"), "target", { it }))
        assertEquals("first", tvQueueFocusAfterRemoval(listOf("first"), listOf("first"), "gone", { it }))
        assertNull(tvQueueFocusAfterRemoval(emptyList<String>(), emptyList(), "gone", { it }))
    }

    @Test fun `removing first or middle item focuses the next surviving row`() {
        assertEquals("second", tvQueueFocusAfterRemoval(listOf("second", "third"), 0))
        assertEquals("third", tvQueueFocusAfterRemoval(listOf("first", "third"), 1))
    }

    @Test fun `removing last row focuses previous row and empty queues have no target`() {
        assertEquals("second", tvQueueFocusAfterRemoval(listOf("first", "second"), 2))
        assertNull(tvQueueFocusAfterRemoval(emptyList<String>(), 0))
    }
}
