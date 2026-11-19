package net.kdhmedia.mochil.webdav

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class WebDavParserTest {
    @Test fun parsesFoldersAndSupportedMediaWhileIgnoringSelfAndUnknownFiles() {
        val page = WebDavParser.parse(MULTISTATUS.trimIndent().encodeToByteArray(), "https://cloud.example/dav/books/")
        assertEquals(listOf("Novels", "Book.epub"), page.entries.map(WebDavEntry::name))
        assertTrue(page.entries.first().collection)
        assertEquals(1234, page.entries.last().contentLength)
        assertEquals("\"book-2\"", page.entries.last().etag)
    }

    @Test fun acceptsNameOnlyApacheDoctypeWithoutEnablingEntities() {
        val declared = MULTISTATUS.trimIndent().replace("<?xml version=\"1.0\"?>", "<?xml version=\"1.0\"?>\n<!DOCTYPE multistatus>")
        assertEquals(listOf("Novels", "Book.epub"), WebDavParser.parse(declared.encodeToByteArray(), "https://cloud.example/dav/books/").entries.map { it.name })
    }

    @Test fun rejectsExternalAndEntityBearingDoctypes() {
        val xml = MULTISTATUS.trimIndent()
        val external = xml.replace("<?xml version=\"1.0\"?>", "<?xml version=\"1.0\"?>\n<!DOCTYPE multistatus SYSTEM \"https://attacker.example/a.dtd\">")
        val entity = xml.replace("<?xml version=\"1.0\"?>", "<?xml version=\"1.0\"?>\n<!DOCTYPE multistatus [<!ENTITY x \"expanded\">]>")
        assertTrue(runCatching { WebDavParser.parse(external.encodeToByteArray(), "https://cloud.example/dav/books/") }.isFailure)
        assertTrue(runCatching { WebDavParser.parse(entity.encodeToByteArray(), "https://cloud.example/dav/books/") }.isFailure)
        assertEquals("webdav_response_error", webDavDiagnosticName(WebDavRequestException("response", "private detail")))
    }

    @Test fun rejectsMultipleDocumentDeclarations() {
        val xml = MULTISTATUS.trimIndent().replace(
            "<?xml version=\"1.0\"?>",
            "<?xml version=\"1.0\"?>\n<!DOCTYPE multistatus>\n<!DOCTYPE multistatus>",
        )
        val error = runCatching { WebDavParser.parse(xml.encodeToByteArray(), "https://cloud.example/dav/books/") }.exceptionOrNull()
        assertTrue(error is WebDavRequestException)
        assertEquals("response", (error as WebDavRequestException).category)
    }

    @Test fun refusesUnsafeAddressesAndSeparatesAccountIdentity() {
        assertTrue(runCatching { validatedWebDavUrl("https://cloud.example/dav/files/alice/") }.isSuccess)
        assertFalse(runCatching { validatedWebDavUrl("http://cloud.example/dav") }.isSuccess)
        assertFalse(runCatching { validatedWebDavUrl("https://alice:secret@cloud.example/dav") }.isSuccess)
        assertFalse(webDavConnectionId("https://cloud.example/dav", "alice") == webDavConnectionId("https://cloud.example/dav", "bob"))
    }

    @Test fun providesActionableProtocolErrors() {
        assertEquals("The WebDAV server rejected these credentials.", webDavStatusMessage(401))
        assertTrue(webDavStatusMessage(405).contains("endpoint"))
        assertEquals("webdav_auth_error", webDavDiagnosticName(webDavStatusFailure(401)))
        assertEquals("webdav_endpoint_error", webDavDiagnosticName(webDavStatusFailure(405)))
        assertEquals("webdav_range_error", webDavDiagnosticName(WebDavRequestException("range", "private detail")))
        assertEquals("webdav_reference_error", webDavDiagnosticName(WebDavRequestException("reference", "private detail")))
        assertEquals("webdav_transport_error", webDavDiagnosticName(java.io.IOException("private detail")))
    }

    private companion object {
        const val MULTISTATUS = """
            <?xml version="1.0"?>
            <d:multistatus xmlns:d="DAV:">
              <d:response><d:href>/dav/books/</d:href><d:propstat><d:prop><d:displayname>Books</d:displayname><d:resourcetype><d:collection/></d:resourcetype></d:prop><d:status>HTTP/1.1 200 OK</d:status></d:propstat></d:response>
              <d:response><d:href>/dav/books/Novels/</d:href><d:propstat><d:prop><d:displayname>Novels</d:displayname><d:resourcetype><d:collection/></d:resourcetype></d:prop><d:status>HTTP/1.1 200 OK</d:status></d:propstat></d:response>
              <d:response><d:href>/dav/books/Book.epub</d:href><d:propstat><d:prop><d:displayname>Book.epub</d:displayname><d:resourcetype/><d:getcontentlength>1234</d:getcontentlength><d:getcontenttype>application/epub+zip</d:getcontenttype><d:getetag>&quot;book-2&quot;</d:getetag></d:prop><d:status>HTTP/1.1 200 OK</d:status></d:propstat></d:response>
              <d:response><d:href>/dav/books/notes.txt</d:href><d:propstat><d:prop><d:displayname>notes.txt</d:displayname><d:resourcetype/></d:prop><d:status>HTTP/1.1 200 OK</d:status></d:propstat></d:response>
            </d:multistatus>
        """
    }
}
