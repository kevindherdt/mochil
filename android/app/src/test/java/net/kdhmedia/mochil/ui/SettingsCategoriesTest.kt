package net.kdhmedia.mochil.ui

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class SettingsCategoriesTest {
    @Test fun `the hub lists the phone's settings in the order people look for them`() {
        assertEquals(
            listOf("Server", "Appearance", "Libraries", "Sources", "Downloads", "Reading", "Playback", "Privacy", "About"),
            SettingsCategory.entries.map { it.label },
        )
    }

    @Test fun `every category says in one short line what is inside`() {
        SettingsCategory.entries.forEach {
            assertTrue("${it.label} has a description", it.description.isNotBlank())
            assertTrue("${it.label} description is one line (${it.description.length})", it.description.length <= 60)
            assertTrue("${it.label} description ends like a sentence", it.description.endsWith("."))
        }
    }

    @Test fun `every category has its own icon`() {
        assertEquals(SettingsCategory.entries.size, SettingsCategory.entries.map { it.iconId }.toSet().size)
    }

    @Test fun `a saved category is read back by name and an unknown or empty one means the hub`() {
        assertEquals(SettingsCategory.Playback, SettingsCategory.fromName("Playback"))
        assertEquals("a category from an older build", null, SettingsCategory.fromName("General"))
        assertNull(SettingsCategory.fromName(null))
        assertNull(SettingsCategory.fromName(""))
    }
}
