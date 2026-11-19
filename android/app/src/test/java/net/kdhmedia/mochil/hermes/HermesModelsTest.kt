package net.kdhmedia.mochil.hermes

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotEquals
import org.junit.Test

/**
 * HermesChatScreen keys its LazyColumn items off [HermesMessage.id] (not list index) so that a
 * streaming bubble keeps its Compose identity while its text/cards are appended in place.
 * These are plain data-identity assertions (no Compose test harness is wired into this module's
 * unit test task); they pin down the exact contract the screen's `key = HermesMessage::id` relies on.
 */
class HermesModelsTest {
    @Test fun eachMessageGetsItsOwnStableId() {
        val first = HermesMessage("user", "Hello")
        val second = HermesMessage("user", "Hello")
        assertNotEquals("Two distinct messages must not share a key, even with identical content.", first.id, second.id)
    }

    @Test fun streamingUpdatesPreserveIdentityForKeying() {
        val message = HermesMessage("assistant", "")
        val afterFirstChunk = message.copy(text = "Hel")
        val afterSecondChunk = afterFirstChunk.copy(text = "Hello", cards = listOf(HermesCard("1", "lib", "book", "Title", "")))
        assertEquals("copy() must keep the same id as chunks stream in, or the list key churns per bubble.", message.id, afterFirstChunk.id)
        assertEquals(message.id, afterSecondChunk.id)
    }
}
