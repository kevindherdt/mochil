package net.kdhmedia.mochil.reader.epub

import org.junit.Assert.assertEquals
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.RuntimeEnvironment
import org.robolectric.annotation.Config

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35])
class ReaderTapZoneDefaultTest {
    private val context get() = RuntimeEnvironment.getApplication()
    private fun prefs() = context.getSharedPreferences("epub_reader_preferences", 0).edit()

    @Test fun aNewInstallTurnsPagesFromTheEdgesLikeTheWeb() {
        prefs().clear().commit()
        assertEquals(TapZones.Edges, EpubReaderPreferencesStore(context).load().tapZones)
        assertEquals(TapZones.Edges, ReaderPreferences().tapZones)
    }

    @Test fun anExplicitSavedChoiceIsKeptIncludingOff() {
        TapZones.entries.forEach { zones ->
            prefs().clear().putString("tap_zones", zones.name).commit()
            assertEquals(zones, EpubReaderPreferencesStore(context).load().tapZones)
        }
    }

    @Test fun anUnreadableSavedValueFallsBackToEdges() {
        prefs().clear().putString("tap_zones", "Nonsense").commit()
        assertEquals(TapZones.Edges, EpubReaderPreferencesStore(context).load().tapZones)
    }
}
