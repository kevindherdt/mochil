package net.kdhmedia.mochil.ui

import org.json.JSONArray
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class TvNavigationContractTest {
    @Test fun `TV bucket labels follow the shared server contract`() {
        val source = requireNotNull(javaClass.classLoader?.getResourceAsStream("navigation_buckets.json"))
            .bufferedReader().use { it.readText() }
        val cases = JSONArray(source)
        assertTrue(cases.length() > 0)
        for (index in 0 until cases.length()) {
            val case = cases.getJSONObject(index)
            assertEquals(case.getString("title"), case.getString("bucket"), tvNavigationBucket(case.getString("title")))
        }
    }
}
