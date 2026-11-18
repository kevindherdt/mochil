package net.kdhmedia.mochil.reader.epub

import java.nio.file.Files
import java.nio.file.Paths
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * Closing the reader used to throw away the one position save that matters: onPause launches the flush on a scope
 * that outlives the lifecycle, and onDestroy cancelled that same scope a moment later. The local position survived
 * (saveNow runs synchronously) but the server kept the old spot, so the reader's next device resumed at the wrong
 * page. Proving the flush actually completes needs a device with a real server, so the ordering is pinned here.
 */
class ReaderFlushLifecycleSourceTest {
    private val source = Files.readString(Paths.get("src/main/java/net/kdhmedia/mochil/reader/epub/EpubReaderActivity.kt"))

    /** Comments explain the rule and quote the call they forbid, so only executable lines count. */
    private val code = source.lineSequence().filterNot { it.trimStart().startsWith("//") }.joinToString("\n")

    @Test fun `the position flush is not cancelled when the reader closes`() {
        assertFalse("onDestroy must not cancel the flush onPause just started", code.contains("flushScope.cancel("))
        assertTrue("onPause still launches the flush", code.contains("flushScope.launch"))
    }

    @Test fun `the reader still releases the publication on close`() {
        val onDestroy = code.substringAfter("override fun onDestroy()").substringBefore("companion object")
        assertTrue("the publication must still be closed", onDestroy.contains("publication?.close()"))
    }
}
