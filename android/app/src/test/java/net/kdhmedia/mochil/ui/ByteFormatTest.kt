package net.kdhmedia.mochil.ui

import java.io.File
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * The four screens that show a byte count (book details, WebDAV browser, playlist summaries, track info)
 * used to carry four separate byte-formatting implementations. Two were byte-for-byte identical, and
 * WebDavScreen's disagreed on unit suffixes (GiB/MiB/KiB vs the GB/MB/KB the other three already used) --
 * a real, user-visible inconsistency. This test pins all four call sites to the one shared utility so they
 * can never drift apart again.
 */
class ByteFormatTest {
    private fun source(name: String) = sequenceOf(
        File("src/main/java/net/kdhmedia/mochil/ui/$name"),
        File("app/src/main/java/net/kdhmedia/mochil/ui/$name"),
        File("android/app/src/main/java/net/kdhmedia/mochil/ui/$name"),
    ).first(File::isFile).readText()

    @Test fun `binary thresholds pick the majority convention's unit`() {
        assertEquals("", formatByteSize(0))
        assertEquals("512 B", formatByteSize(512))
        assertEquals("1.0 KB", formatByteSize(1_024))
        assertEquals("24.0 MB", formatByteSize(25_165_824))
        assertEquals("1.0 GB", formatByteSize(1_073_741_824))
        assertEquals("2.5 GB", formatByteSize(2_684_354_560))
    }

    @Test fun `all four call sites delegate to the shared formatter instead of their own copy`() {
        val sites = listOf(
            "PublicationItems.kt" to "formatBytes",
            "WebDavScreen.kt" to "formatFileSize",
            "MusicPlaylistLibrary.kt" to "formatPlaylistBytes",
            "MusicTrackInfoDialog.kt" to "formatMusicFileSize",
        )
        for ((file, oldFunctionName) in sites) {
            val text = source(file)
            assertTrue("$file should call the shared formatByteSize", text.contains("formatByteSize("))
            assertFalse(
                "$file should no longer define its own $oldFunctionName byte-math (GiB/MiB/binary shifts)",
                text.contains("1L shl") || text.contains("GiB") || text.contains("MiB") || text.contains("KiB"),
            )
        }
    }

    @Test fun `WebDavScreen no longer disagrees with the other three on unit suffixes`() {
        // This is the regression the audit flagged: before the fix WebDavScreen alone used GiB/MiB/KiB.
        assertFalse(source("WebDavScreen.kt").contains("GiB"))
        assertEquals("1.0 GB", formatByteSize(1_073_741_824))
    }
}
