package net.kdhmedia.mochil.reader.epub

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotEquals
import org.junit.Test

class EpubPublicationLoaderTest {
    @Test fun retrievalAndParsingFailuresRemainActionableAndDistinct() {
        assertEquals("This EPUB could not be opened.", EpubPublicationFailure.Retrieve.userMessage)
        assertEquals("This EPUB could not be read.", EpubPublicationFailure.Parse.userMessage)
        assertNotEquals(EpubPublicationFailure.Retrieve.userMessage, EpubPublicationFailure.Parse.userMessage)
    }
}
