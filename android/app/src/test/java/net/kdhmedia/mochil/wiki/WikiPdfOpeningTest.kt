package net.kdhmedia.mochil.wiki

import java.io.File
import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.runBlocking
import kotlinx.coroutines.yield
import net.kdhmedia.mochil.auth.DeviceCredential
import org.junit.Assert.*
import org.junit.Test

class WikiPdfOpeningTest {
    private val credential = DeviceCredential("https://fixture.invalid", "device", "token")
    private val entry = WikiEntry("page", "pdf", "Notes/Manual.pdf", "Manual", "pdf", revision = "r1")
    private val state = WikiState(libraryId = "vault", entries = listOf(entry), selectedPath = "Notes/Page.md")

    @Test fun `PDF opening uses a local-only progress identity and clears loading`() = runBlocking {
        val statuses = mutableListOf<String?>()
        val opened = mutableListOf<String>()
        val owner = WikiPdfOpening(this, { _, library, target ->
            assertEquals("vault", library); assertEquals(entry, target); File("fixture.pdf")
        }, { _, _, key -> opened += key }, statuses::add, { fail(it) })
        assertFalse(owner.openLink("Other note", state, credential))
        assertFalse(owner.openLink("Manual.pdf.md#Details|Label", state, credential))
        assertTrue(owner.openLink("Manual.pdf", state, credential))
        yield()
        assertTrue(opened.single().startsWith("local:wiki-pdf-"))
        assertEquals(listOf(null, "Opening PDF…", null), statuses)
    }

    @Test fun `leaving the note cancels pending opening without reporting an error`() = runBlocking {
        val entered = CompletableDeferred<Unit>()
        val reply = CompletableDeferred<File>()
        var opened = false
        var status: String? = null
        val owner = WikiPdfOpening(this, { _, _, _ -> entered.complete(Unit); reply.await() },
            { _, _, _ -> opened = true }, { status = it }, { fail(it) })
        owner.openLink("Manual.pdf", state, credential)
        entered.await()
        owner.cancel()
        reply.complete(File("fixture.pdf"))
        yield()
        assertFalse(opened)
        assertNull(status)
    }

    @Test fun `unmatched PDF never downloads or proposes creating a Markdown note`() = runBlocking {
        val errors = mutableListOf<String>()
        val owner = WikiPdfOpening(this, { _, _, _ -> error("Unexpected download") },
            { _, _, _ -> fail("Unexpected open") }, {}, errors::add)
        assertTrue(owner.openLink("Missing.pdf", state, credential))
        assertEquals(1, errors.size)
    }
}
