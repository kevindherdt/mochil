package net.kdhmedia.mochil.opds

import java.io.File
import org.junit.Assert.assertEquals
import org.junit.Test

/** The artwork cache trim must track its total incrementally, not re-scan the whole directory on
 *  every write (tested here by changing the directory behind the tracker's back after the first
 *  scan and confirming it keeps reporting the pre-change total). */
class ArtworkCacheSizeTest {
    @Test fun scansOnceThenTracksIncrementally() {
        val directory = createTempDirectory()
        try {
            File(directory, "a.jpg").writeBytes(ByteArray(100))
            File(directory, "b.jpg").writeBytes(ByteArray(50))
            val size = ArtworkCacheSize(directory)

            assertEquals("The first call must scan the real directory contents.", 150L, size.add(0))

            // A write lands on disk independently of the tracker; it must be reflected by the delta, not a re-scan.
            File(directory, "c.jpg").writeBytes(ByteArray(25))
            assertEquals(175L, size.add(25))

            // Something else touches the directory without going through the tracker (e.g. the OS). The
            // tracker must NOT silently pick this up by re-scanning - it stays authoritative from deltas only.
            File(directory, "untracked.jpg").writeBytes(ByteArray(1_000))
            assertEquals("A later add() must not re-scan and discover the untracked file.", 185L, size.add(10))
        } finally {
            directory.deleteRecursively()
        }
    }

    @Test fun removalsAreSubtracted() {
        val directory = createTempDirectory()
        try {
            File(directory, "a.jpg").writeBytes(ByteArray(100))
            val size = ArtworkCacheSize(directory)
            size.add(0) // warm the scan
            assertEquals(40L, size.add(-60))
        } finally {
            directory.deleteRecursively()
        }
    }

    private fun createTempDirectory(): File = File.createTempFile("artwork-cache-size-", "").apply { delete(); mkdirs() }
}
