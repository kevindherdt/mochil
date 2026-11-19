package net.kdhmedia.mochil.reader.pdf

import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class PdfDocumentTest {
    @Test fun renderSizePreservesOrdinaryPageRatio() {
        assertEquals(PdfRenderSize(1_200, 1_600), boundedPdfRenderSize(600, 800, 1_200))
    }

    @Test fun renderSizeBoundsExtremePagesAndDeviceWidths() {
        val size = boundedPdfRenderSize(1, 100_000, 20_000)
        assertTrue(size.width in 1..PdfDocument.MAX_EDGE)
        assertTrue(size.height in 1..PdfDocument.MAX_EDGE)
        assertTrue(size.width.toLong() * size.height <= PdfDocument.MAX_PIXELS)
    }
}
