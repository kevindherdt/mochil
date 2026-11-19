package net.kdhmedia.mochil.wiki

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Test

class WikiOfflineAssetBudgetTest {
    @Test fun `retained and incoming images share one total byte budget`() {
        val candidates = (0 until 16).map { "asset-$it" to MAX_OFFLINE_ASSET_BYTES.toLong() }
        assertEquals((0 until 8).map { "asset-$it" }.toSet(), boundedWikiAssetNames(candidates))
    }

    @Test fun `invalid and duplicate files do not consume budget`() {
        val candidates = listOf(
            "empty" to 0L, "negative" to -1L,
            "oversize" to MAX_OFFLINE_ASSET_BYTES.toLong() + 1,
            "one" to 1L, "one" to 1L, "two" to 1L,
        )
        assertEquals(setOf("one", "two"), boundedWikiAssetNames(candidates))
    }

    @Test fun `small images still respect the file count limit`() {
        assertEquals(MAX_OFFLINE_ASSETS, boundedWikiAssetNames((0 until 100).map { "$it" to 1L }).size)
    }

    @Test fun `a later smaller image can fit the remaining budget`() {
        val large = MAX_OFFLINE_ASSET_BYTES.toLong()
        val candidates = (0 until 7).map { "$it" to large } + listOf(
            "partial" to large - 1, "too-big" to 2L, "fits" to 1L,
        )
        val keep = boundedWikiAssetNames(candidates)
        assertFalse("too-big" in keep)
        assertEquals("fits", keep.last())
    }
}
