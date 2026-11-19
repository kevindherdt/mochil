package net.kdhmedia.mochil.catalogue

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * A proper [ReadingStatus] enum already existed and the UI used it correctly, but
 * CollectionViewModel.setReadingStatus / BookSummary.withReadingStatus / readingStatusMessage took raw
 * wire strings, and the literal "reading" was hardcoded in DashboardViewModel and HermesActions instead of
 * referencing [ReadingStatus.Reading].wire. This pins the enum-typed signatures and the wire-string
 * contract they still have to honour when talking to the server.
 */
class ReadingStatusMigrationTest {
    private val book = org.json.JSONObject("""{"id":"one","library_id":"books","title":"One"}""").toBookSummary()

    @Test fun `withReadingStatus and readingStatusMessage take the enum, not a raw string`() {
        assertEquals(ReadingStatus.Read, book.withReadingStatus(ReadingStatus.Read).readingStatus())
        assertEquals(ReadingStatus.Reading, book.withReadingStatus(ReadingStatus.Reading).readingStatus())
        assertEquals(ReadingStatus.OnHold, book.withReadingStatus(ReadingStatus.OnHold).readingStatus())
        assertEquals(ReadingStatus.Dropped, book.withReadingStatus(ReadingStatus.Dropped).readingStatus())
        assertEquals(ReadingStatus.Unread, book.withReadingStatus(ReadingStatus.Unread).readingStatus())
        assertEquals("Back in your reading list.", readingStatusMessage(ReadingStatus.Reading))
        assertEquals("Marked as read.", readingStatusMessage(ReadingStatus.Read))
    }

    @Test fun `every enum member still round-trips to the exact wire string the server expects`() {
        // withReadingStatus/readingStatusMessage are enum-typed now, but ReadingStatusClient.set still
        // posts a raw wire string to the server, so the enum's .wire values must stay stable.
        assertEquals("reading", ReadingStatus.Reading.wire)
        assertEquals("read", ReadingStatus.Read.wire)
        assertEquals("on_hold", ReadingStatus.OnHold.wire)
        assertEquals("dropped", ReadingStatus.Dropped.wire)
        assertEquals("unread", ReadingStatus.Unread.wire)
    }

    @Test fun `DashboardViewModel and HermesActions reference ReadingStatus Reading wire, not a hardcoded literal`() {
        val dashboard = source("dashboard", "DashboardViewModel.kt")
        val hermes = source("hermes", "HermesActions.kt")
        assertTrue(dashboard.contains("ReadingStatus.Reading.wire"))
        assertTrue(hermes.contains("ReadingStatus.Reading.wire"))
        // The old hardcoded literal calls should be gone from the exact spots this was fixed.
        assertFalse(dashboard.contains("book.id, \"reading\")"))
        assertFalse(hermes.contains("safe(\"reading\")"))
    }

    private fun source(pkg: String, name: String) = sequenceOf(
        java.io.File("src/main/java/net/kdhmedia/mochil/$pkg/$name"),
        java.io.File("app/src/main/java/net/kdhmedia/mochil/$pkg/$name"),
        java.io.File("android/app/src/main/java/net/kdhmedia/mochil/$pkg/$name"),
    ).first(java.io.File::isFile).readText()
}
