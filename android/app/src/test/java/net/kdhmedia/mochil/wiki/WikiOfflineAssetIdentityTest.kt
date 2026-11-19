package net.kdhmedia.mochil.wiki

import java.security.MessageDigest
import java.util.Locale
import org.junit.Assert.assertEquals
import org.junit.Test

class WikiOfflineAssetIdentityTest {
    @Test fun optimizedNamesMatchExistingCacheNamesByteForByte() {
        val values = listOf("", "cover", "日本語/表紙.png", "folder/emoji-📚.webp", "a\u0000b", " leading ")
        for ((index, value) in values.withIndex()) {
            val entry = WikiEntry(kind = "file", id = "id-$index", path = value,
                title = "Image", format = "png", revision = "revision-$value")
            val legacy = MessageDigest.getInstance("SHA-256")
                .digest("${entry.id}\u0000${entry.revision}\u0000${entry.path}".encodeToByteArray())
                .joinToString("") { String.format(Locale.ROOT, "%02x", it.toInt() and 255) } + ".bin"
            assertEquals(legacy, wikiOfflineAssetFileName(entry))
            assertEquals(68, wikiOfflineAssetFileName(entry).length)
        }
    }
}
