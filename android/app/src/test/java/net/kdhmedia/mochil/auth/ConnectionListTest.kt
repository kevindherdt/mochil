package net.kdhmedia.mochil.auth

import org.junit.Assert.assertEquals
import org.junit.Assert.assertThrows
import org.junit.Test

class ConnectionListTest {
    @Test fun newConnectionPastTheCapIsRefusedNotDropped() {
        val full = List(3) { "c$it" }
        assertThrows(IllegalArgumentException::class.java) { connectionsWith(full, "extra", 3) { it } }
    }

    @Test fun replacingAnExistingConnectionAtTheCapIsAllowed() {
        val full = listOf("a", "b", "c")
        assertEquals(listOf("a", "c", "b"), connectionsWith(full, "b", 3) { it })
    }

    @Test fun failedCommitIsReportedAsAnError() {
        assertThrows(IllegalStateException::class.java) { requireSaved(false) }
        requireSaved(true)
    }
}
