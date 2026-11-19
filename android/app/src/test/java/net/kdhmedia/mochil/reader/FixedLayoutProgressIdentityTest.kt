package net.kdhmedia.mochil.reader

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertNotEquals
import org.junit.Test

class FixedLayoutProgressIdentityTest {
    @Test fun localStorageKeysCannotAliasColonContainingIds() {
        assertNotEquals(fixedLayoutStorageKey("a:b", "c"), fixedLayoutStorageKey("a", "b:c"))
    }
    @Test fun explicitIdsPreserveColonsWithoutParsing() {
        assertEquals(
            FixedLayoutProgressIdentity("webdav:family", "folder:book.cbz"),
            fixedLayoutProgressIdentity("webdav:family", "folder:book.cbz", "ignored:key"),
        )
    }

    @Test fun legacyServerDownloadsRemainCompatible() {
        assertEquals(
            FixedLayoutProgressIdentity("manga", "book:chapter-2"),
            fixedLayoutProgressIdentity("", "", "manga:book:chapter-2"),
        )
    }

    @Test fun localOnlySourcesNeverAttemptServerProgress() {
        assertNull(fixedLayoutProgressIdentity("", "", "local:document"))
        assertNull(fixedLayoutProgressIdentity("", "", "webdav:document"))
        assertNull(fixedLayoutProgressIdentity("", "", "invalid"))
    }
}
