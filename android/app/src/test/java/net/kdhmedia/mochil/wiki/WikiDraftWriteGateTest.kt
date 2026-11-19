package net.kdhmedia.mochil.wiki

import java.util.concurrent.CountDownLatch
import java.util.concurrent.Executors
import java.util.concurrent.TimeUnit
import org.junit.Assert.*
import org.junit.Test

class WikiDraftWriteGateTest {
    @Test fun `clear requested during recovery read suppresses its result`() {
        val gate = WikiDraftWriteGate()
        var pending: (() -> Unit)? = null
        val recovered = gate.readVisible {
            gate.clearLater({ pending = it }) {}
            "discarded text"
        }
        assertNull(recovered)
        assertNotNull(pending)
        pending!!()
    }

    @Test fun `new edit during recovery read supersedes old result`() {
        val gate = WikiDraftWriteGate()
        assertNull(gate.readVisible {
            gate.begin()
            "old draft"
        })
    }

    @Test fun `uninterrupted recovery read returns its result`() {
        assertEquals("draft", WikiDraftWriteGate().readVisible { "draft" })
    }

    @Test fun `queued clear hides draft immediately without doing storage on caller`() {
        val gate = WikiDraftWriteGate()
        var pending: (() -> Unit)? = null
        var stored = "old"
        gate.clearLater({ pending = it }) { stored = "" }
        assertEquals("old", stored)
        assertNull(gate.readVisible { stored })
        pending!!()
        assertEquals("", stored)
    }

    @Test fun `delayed clear cannot erase newer saved draft`() {
        val gate = WikiDraftWriteGate()
        var pending: (() -> Unit)? = null
        var stored = "old"
        gate.clearLater({ pending = it }) { stored = "" }
        val next = gate.begin()
        assertNull(gate.readVisible { stored })
        gate.write(next) { stored = "new" }
        pending!!()
        assertEquals("new", gate.readVisible { stored })
    }

    @Test fun `failed clear stays hidden for this process`() {
        val gate = WikiDraftWriteGate()
        try { gate.clear { throw IllegalStateException("disk") } }
        catch (_: IllegalStateException) { }
        assertNull(gate.readVisible { "discarded" })
    }

    @Test fun `delayed save cannot resurrect cleared text`() {
        val gate = WikiDraftWriteGate()
        val ticket = gate.begin()
        var stored = "old"
        gate.clear { stored = "" }
        gate.write(ticket) { stored = "cancelled draft" }
        assertEquals("", stored)
    }

    @Test fun `newest edit wins reversed execution order`() {
        val gate = WikiDraftWriteGate()
        val first = gate.begin()
        val second = gate.begin()
        var stored = ""
        gate.write(second) { stored = "new" }
        gate.write(first) { stored = "old" }
        assertEquals("new", stored)
    }

    @Test fun `clear follows a write already inside storage`() {
        val gate = WikiDraftWriteGate()
        val ticket = gate.begin()
        val entered = CountDownLatch(1)
        val release = CountDownLatch(1)
        val pool = Executors.newFixedThreadPool(2)
        var stored = ""
        try {
            val saving = pool.submit {
                gate.write(ticket) {
                    entered.countDown()
                    check(release.await(5, TimeUnit.SECONDS))
                    stored = "old"
                }
            }
            assertTrue(entered.await(5, TimeUnit.SECONDS))
            val clearing = pool.submit { gate.clear { stored = "" } }
            release.countDown()
            saving.get(5, TimeUnit.SECONDS)
            clearing.get(5, TimeUnit.SECONDS)
            assertEquals("", gate.read { stored })
        } finally {
            release.countDown()
            pool.shutdownNow()
        }
    }

    @Test fun `write exception releases storage lock for clear`() {
        val gate = WikiDraftWriteGate()
        try {
            gate.write(gate.begin()) { throw IllegalStateException("storage failed") }
            fail("exception swallowed")
        } catch (_: IllegalStateException) {
            var cleared = false
            gate.clear { cleared = true }
            assertTrue(cleared)
        }
    }
}
