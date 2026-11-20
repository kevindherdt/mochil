package net.kdhmedia.mochil.wiki

import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class WikiPageSessionTest {
    @Test fun `leaving and reopening invalidates every response from the earlier visit`() {
        val session = WikiPageSession()
        val saving = session.capture()
        assertTrue(session.isCurrent(saving))
        session.reset() // Leave the page.
        val otherPage = session.capture()
        session.reset() // Return to the same page ID and revision.
        assertFalse(session.isCurrent(saving))
        assertFalse(session.isCurrent(otherPage))
        assertTrue(session.isCurrent(session.capture()))
    }
}
