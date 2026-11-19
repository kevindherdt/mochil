package net.kdhmedia.mochil.reader

import org.junit.Assert.assertEquals
import org.junit.Test

class ReaderDeviceNameTest {
    @Test fun `device label combines distinct manufacturer and model`() {
        assertEquals("Samsung Galaxy Tab", readerDeviceName("Samsung", "Galaxy Tab"))
    }

    @Test fun `device label does not repeat a manufacturer already in the model`() {
        assertEquals("Samsung Galaxy S", readerDeviceName("Samsung", "Samsung Galaxy S"))
    }

    @Test fun `device label is bounded and always useful`() {
        assertEquals("Android device", readerDeviceName(" ", ""))
        assertEquals(120, readerDeviceName("Maker", "x".repeat(200)).length)
    }
}
