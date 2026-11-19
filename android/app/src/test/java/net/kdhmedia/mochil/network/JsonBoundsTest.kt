package net.kdhmedia.mochil.network

import java.io.IOException
import org.json.JSONArray
import org.junit.Assert.assertSame
import org.junit.Assert.assertThrows
import org.junit.Test

class JsonBoundsTest {
    @Test fun `array accepts its exact item ceiling`() {
        val values = JSONArray("[1,2,3]")
        assertSame(values, values.requireAtMost(3, "Catalogue"))
    }

    @Test fun `array rejects one item beyond its ceiling`() {
        assertThrows(IOException::class.java) { JSONArray("[1,2,3]").requireAtMost(2, "Catalogue") }
    }
}
