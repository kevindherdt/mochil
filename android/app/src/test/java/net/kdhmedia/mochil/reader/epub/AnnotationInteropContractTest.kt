package net.kdhmedia.mochil.reader.epub

import org.json.JSONObject
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class AnnotationInteropContractTest {
    @Test fun sharedKoreaderAndReadiumFixturesDecodeAndRoundTrip() {
        val stream = requireNotNull(javaClass.classLoader?.getResourceAsStream("annotation_interop.json"))
        val contract = JSONObject(stream.bufferedReader().use { it.readText() })
        assertEquals(1, contract.getInt("version"))
        val fixtures = contract.getJSONArray("annotations")
        assertEquals(2, fixtures.length())

        repeat(fixtures.length()) { index ->
            val fixture = fixtures.getJSONObject(index)
            val annotation = decodeSyncedAnnotation(fixture.getJSONObject("annotation"))
            assertEquals(fixture.getInt("expected_spine_index"), annotation.anchor?.spineIndex)
            assertEquals(annotation.anchor?.exact, annotation.quote)
            assertTrue(annotation.id.startsWith(fixture.getString("origin") + ":"))

            val roundTrip = decodeSyncedAnnotation(encodeSyncedAnnotation(annotation))
            assertEquals(annotation, roundTrip)
            fixture.optString("expected_readium_href").takeIf(String::isNotBlank)?.let { href ->
                val locator = JSONObject(annotation.locators.getValue("readium"))
                assertEquals(href, locator.getString("href"))
                assertEquals(annotation.quote, locator.getJSONObject("text").getString("highlight"))
            }
            fixture.optString("expected_koreader_css_selector").takeIf(String::isNotBlank)?.let { selector ->
                val native = JSONObject(annotation.locators.getValue("koreader"))
                val parsed = KoReaderXPointer.parse(native.getString("pos0"))
                assertEquals(fixture.getInt("expected_spine_index"), parsed?.spineIndex)
                assertEquals(selector, parsed?.cssSelector)
                assertEquals(fixture.getInt("expected_koreader_text_offset"), parsed?.textOffset)
            }
        }
    }
}
