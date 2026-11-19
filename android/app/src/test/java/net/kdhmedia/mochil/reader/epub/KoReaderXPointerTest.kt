package net.kdhmedia.mochil.reader.epub

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class KoReaderXPointerTest {
    @Test fun `converts bounded CREngine path to Readium CSS selector`() {
        val parsed = KoReaderXPointer.parse("/body/DocFragment[4]/body/div[1]/p[3]/text().25")

        assertEquals(3, parsed?.spineIndex)
        assertEquals("body > div:nth-of-type(1) > p:nth-of-type(3)", parsed?.cssSelector)
        assertEquals(1, parsed?.textNodeIndex)
        assertEquals(25, parsed?.textOffset)
    }

    @Test fun `supports indexed text nodes and default element siblings`() {
        val parsed = KoReaderXPointer.parse("/body/DocFragment[1].0/body/section/p/text()[2].7")

        assertEquals(0, parsed?.spineIndex)
        assertEquals("body > section:nth-of-type(1) > p:nth-of-type(1)", parsed?.cssSelector)
        assertEquals(2, parsed?.textNodeIndex)
        assertEquals(7, parsed?.textOffset)
    }

    @Test fun `rejects malformed injectable and unbounded paths`() {
        assertNull(KoReaderXPointer.parse("/body/DocFragment[0]/body/p[1]"))
        assertNull(KoReaderXPointer.parse("/body/DocFragment[1]/body/p:nth-child(1)"))
        assertNull(KoReaderXPointer.parse("/body/DocFragment[1]/body/text().0/p[1]"))
        assertNull(KoReaderXPointer.parse("x".repeat(8_193)))
    }

    @Test fun `capture script bounds input and decoded result is revalidated`() {
        val script = KoReaderXPointer.captureScript(3, "body > div:nth-of-type(1)")
        assertEquals(true, script?.contains("DocFragment[4]"))
        assertEquals("/body/DocFragment[4]/body/div[1]", KoReaderXPointer.decodeJavascriptResult("\"/body/DocFragment[4]/body/div[1]\""))
        assertNull(KoReaderXPointer.decodeJavascriptResult("\"/body/DocFragment[0]/body/div[1]\""))
        assertNull(KoReaderXPointer.captureScript(-1, "body"))
    }
}
