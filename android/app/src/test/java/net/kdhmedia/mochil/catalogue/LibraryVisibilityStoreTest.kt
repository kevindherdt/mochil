package net.kdhmedia.mochil.catalogue

import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class LibraryVisibilityStoreTest {
    @Test fun televisionDefaultsHideReadingButKeepPlaybackLibraries() {
        assertFalse(defaultLibraryVisible(ClientDeviceClass.Television, "light_novels"))
        assertFalse(defaultLibraryVisible(ClientDeviceClass.Television, "manga"))
        assertTrue(defaultLibraryVisible(ClientDeviceClass.Television, "movies"))
        assertTrue(defaultLibraryVisible(ClientDeviceClass.Television, "podcasts"))
    }

    @Test fun einkDefaultsHideVideoButKeepReadingAndAudioLibraries() {
        assertFalse(defaultLibraryVisible(ClientDeviceClass.EInk, "tv-shows"))
        assertTrue(defaultLibraryVisible(ClientDeviceClass.EInk, "books"))
        assertTrue(defaultLibraryVisible(ClientDeviceClass.EInk, "audiobooks"))
    }

    @Test fun explicitChoiceOverridesDeviceDefault() {
        assertTrue(resolveLibraryVisibility("manga", false, setOf("manga"), emptySet()))
        assertFalse(resolveLibraryVisibility("movies", true, emptySet(), setOf("movies")))
    }

    @Test fun einkDetectionUsesFeaturesAndKnownDedicatedReaders() {
        assertTrue(isEInkDevice(setOf("com.vendor.hardware.e_ink"), "", "", ""))
        assertTrue(isEInkDevice(emptySet(), "ONYX", "BOOX", "Page"))
        assertFalse(isEInkDevice(emptySet(), "Samsung", "Galaxy", "S24"))
        assertFalse(isEInkDevice(emptySet(), "Amazon", "Amazon", "AFTMM"))
    }
}
