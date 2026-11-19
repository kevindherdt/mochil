package net.kdhmedia.mochil.wiki

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class WikiDraftStoreTest {
    @Test fun `draft owner identity is stable opaque and account scoped`() {
        val alice = wikiOwnerIdentity("alice-device")
        assertEquals(alice, wikiOwnerIdentity("alice-device"))
        assertFalse(alice.contains("alice"))
        assertFalse(alice == wikiOwnerIdentity("bob-device"))
    }

    @Test fun `only content different from the saved page needs recovery IO`() {
        assertFalse(shouldPersistWikiDraft("saved", "saved"))
        assertTrue(shouldPersistWikiDraft("changed", "saved"))
    }

    private val page = WikiPage("page-1", "Page", "saved", "revision-2", true)

    @Test fun `only a changed draft for the same vault page is recovered`() {
        val draft = WikiDraft("vault", "page-1", "revision-1", "unsaved", 10)
        assertEquals(draft, draft.recoverFor("vault", page))
        assertNull(draft.recoverFor("another-vault", page))
        assertNull(draft.copy(pageId = "page-2").recoverFor("vault", page))
        assertNull(draft.copy(content = "saved").recoverFor("vault", page))
    }

    @Test fun `recovered draft keeps the revision that originally owned it`() {
        val draft = WikiDraft("vault", "page-1", "revision-1", "unsaved", 10).recoverFor("vault", page)
        assertEquals("revision-1", draft?.revision)
    }

    @Test fun `only initial selection of the matching page retains a recovery candidate`() {
        val draft = WikiDraft("vault", "page-1", "revision-1", "unsaved", 10)
        assertEquals(draft, draft.retainForInitialSelection(null, "vault", "page-1"))
        assertNull(draft.retainForInitialSelection(page, "vault", "page-1"))
        assertNull(draft.retainForInitialSelection(null, "other", "page-1"))
        assertNull(draft.retainForInitialSelection(null, "vault", "page-2"))
    }
}
