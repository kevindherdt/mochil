package net.kdhmedia.mochil.offline

import java.io.IOException
import java.net.HttpURLConnection
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class ResumableHttpDownloadTest {
    @Test fun `matching partial response appends at the requested byte`() {
        assertEquals(DownloadResponseMode.Append, downloadResponseMode(HttpURLConnection.HTTP_PARTIAL, 4096, "bytes 4096-8191/12000"))
    }

    @Test fun `full response restarts when the server ignores range`() {
        assertEquals(DownloadResponseMode.Restart, downloadResponseMode(HttpURLConnection.HTTP_OK, 4096, null))
    }

    @Test(expected = IOException::class)
    fun `mismatched partial range is rejected instead of corrupting the file`() {
        downloadResponseMode(HttpURLConnection.HTTP_PARTIAL, 4096, "bytes 0-8191/12000")
    }

    @Test fun `only partials with validator state survive startup cleanup`() {
        val root = kotlin.io.path.createTempDirectory("mochil-resume").toFile()
        try {
            val directory = root.resolve("music").apply { mkdirs() }
            val resumable = directory.resolve("valid.mp3.part").apply { writeText("some bytes") }
            ResumableHttpDownload.metadataFile(resumable).writeText("{\"validator\":\"etag-1\"}")
            val orphan = directory.resolve("orphan.mp3.part").apply { writeText("old bytes") }
            val removed = PartialDownloadCleanup.clean(root)
            assertEquals(1, removed)
            assertTrue(resumable.exists())
            assertTrue(ResumableHttpDownload.metadataFile(resumable).exists())
            assertFalse(orphan.exists())
        } finally { root.deleteRecursively() }
    }
}
