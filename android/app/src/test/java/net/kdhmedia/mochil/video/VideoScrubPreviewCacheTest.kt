package net.kdhmedia.mochil.video

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertSame
import org.junit.Test

/**
 * Repeated scrubbing lands on the same quantized bucket; the cache must serve it without a
 * re-fetch. [BoundedLruCache] is generic precisely so this eviction logic (the thing that matters
 * for the fix) can be exercised with plain values instead of a real decoded [android.graphics.Bitmap],
 * which the JVM unit test SDK stub can neither construct nor meaningfully mock.
 */
class BoundedLruCacheTest {
    @Test fun missThenHitOnTheSameKey() {
        val cache = BoundedLruCache<String, String>(maxEntries = 4)
        assertNull(cache.get("bucket-10"))
        cache.put("bucket-10", "frame-10")
        assertSame("frame-10", cache.get("bucket-10"))
    }

    @Test fun evictsTheLeastRecentlyUsedEntryOnceBoundExceeded() {
        val cache = BoundedLruCache<String, String>(maxEntries = 2)
        cache.put("a", "frame-a")
        cache.put("b", "frame-b")
        cache.get("a") // touch "a" so "b" becomes least-recently-used
        cache.put("c", "frame-c") // over capacity: evicts "b", not "a"

        assertEquals(2, cache.size())
        assertEquals("frame-a", cache.get("a"))
        assertNull("The least-recently-used entry must be the one evicted.", cache.get("b"))
        assertEquals("frame-c", cache.get("c"))
    }

    @Test fun bucketKeyDistinguishesItemsAndRevisionsButNotTheQuantizedSecond() {
        val entry = VideoPlaybackEntry("lib", "item", "Title", "u", "", 0L, 600.0, "", revision = "r1")
        val reEncoded = entry.copy(revision = "r2")
        assertEquals(videoScrubPreviewCacheKey(entry, 30), videoScrubPreviewCacheKey(entry, 30))
        assertFalse(videoScrubPreviewCacheKey(entry, 30) == videoScrubPreviewCacheKey(entry, 40))
        assertFalse(videoScrubPreviewCacheKey(entry, 30) == videoScrubPreviewCacheKey(reEncoded, 30))
    }
}
