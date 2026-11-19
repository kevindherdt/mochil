package net.kdhmedia.mochil.wiki

import java.io.IOException
import org.junit.Assert.assertEquals
import org.junit.Assert.fail
import org.junit.Test

class WikiOfflineCommitTest {
    @Test fun `payload failure leaves index and previous caches untouched`() {
        val calls = mutableListOf<String>()
        try {
            commitWikiOfflineCache({ calls += "payload"; false }, { calls += "index"; true }, { calls += "evict" })
            fail("failure not reported")
        } catch (_: IOException) { }
        assertEquals(listOf("payload"), calls)
    }

    @Test fun `index failure does not evict previous caches`() {
        val calls = mutableListOf<String>()
        try {
            commitWikiOfflineCache({ calls += "payload"; true }, { calls += "index"; false }, { calls += "evict" })
            fail("failure not reported")
        } catch (_: IOException) { }
        assertEquals(listOf("payload", "index"), calls)
    }

    @Test fun `successful replacement commits before eviction`() {
        val calls = mutableListOf<String>()
        commitWikiOfflineCache({ calls += "payload"; true }, { calls += "index"; true }, { calls += "evict" })
        assertEquals(listOf("payload", "index", "evict"), calls)
    }
}
