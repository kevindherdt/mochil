package net.kdhmedia.mochil.reader.epub

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class TextSharpnessTest {
    @Test fun togglesBetweenOffAndOn() {
        assertEquals(listOf(TextSharpness.On, TextSharpness.Off, TextSharpness.On), generateSequence(TextSharpness.Off) { it.next() }.drop(1).take(3).toList())
    }

    @Test fun offLeavesTheThemeColoursAlone() {
        assertNull(sharpTextColors(TextSharpness.Off, dark = false, sepia = false))
        assertNull(sharpTextColors(TextSharpness.Off, dark = true, sepia = false))
    }

    @Test fun sharpTextIsPureBlackOnWhiteAndWhiteOnBlack() {
        assertEquals(0xFF000000.toInt() to 0xFFFFFFFF.toInt(), sharpTextColors(TextSharpness.On, dark = false, sepia = false))
        assertEquals(0xFFFFFFFF.toInt() to 0xFF000000.toInt(), sharpTextColors(TextSharpness.On, dark = true, sepia = false))
    }

    @Test fun sepiaKeepsItsOwnColours() = assertNull(sharpTextColors(TextSharpness.On, dark = false, sepia = true))

    @Test fun theStyleCarriesTheStrokeAndOffRemovesIt() {
        assertTrue(sharpTextScript(TextSharpness.On).contains("-webkit-text-stroke:0.2px"))
        assertTrue(sharpTextScript(TextSharpness.On).contains("-webkit-text-stroke:0.2px"))
        assertFalse(sharpTextScript(TextSharpness.Off).contains("text-stroke"))
        assertTrue(sharpTextScript(TextSharpness.Off).contains("s.textContent=''"))
    }
}
