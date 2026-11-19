package net.kdhmedia.mochil.video

import org.junit.Assert.assertEquals
import org.junit.Test

class SubtitlePreferencesTest {
    @Test fun `unsafe imported values are bounded`() {
        val normalized = SubtitlePreferences(textScale = 4f, backgroundOpacity = -2f).normalized()

        assertEquals(1.5f, normalized.textScale)
        assertEquals(0f, normalized.backgroundOpacity)
    }

    @Test fun `all presets have distinct stable labels`() {
        assertEquals(SubtitlePreset.entries.size, SubtitlePreset.entries.map(SubtitlePreset::label).distinct().size)
    }
}
