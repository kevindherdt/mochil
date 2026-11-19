package net.kdhmedia.mochil.ui

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class PublicationDensityTest {
    @Test fun `series cover badge keeps only the compact count`() = assertEquals("12", seriesCountLabel(12))

    @Test fun `expanded volume hides the same series author`() {
        assertFalse(shouldShowBookAuthors(listOf("Natsume Akatsuki"), listOf("Natsume Akatsuki")))
    }

    @Test fun `expanded volume retains a changed or additional author`() {
        assertTrue(shouldShowBookAuthors(listOf("Guest Writer"), listOf("Series Writer")))
        assertTrue(shouldShowBookAuthors(listOf("Series Writer", "Illustrator"), listOf("Series Writer")))
    }

    @Test fun `author comparison ignores order case and surrounding space`() {
        assertFalse(shouldShowBookAuthors(listOf(" B ", "a"), listOf("A", "b")))
    }
}
