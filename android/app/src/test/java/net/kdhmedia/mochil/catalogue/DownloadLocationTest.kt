package net.kdhmedia.mochil.catalogue

import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class DownloadLocationTest {
    @Test fun ordinaryAndRecoveredMessagesAreNotErrorsByDefault() {
        assertFalse(DownloadLocation(message = "Recovered 2 downloads.").messageIsError)
    }

    @Test fun folderAccessFailuresCanBeAnnouncedAsErrors() {
        assertTrue(DownloadLocation(message = "Could not access folder.", messageIsError = true).messageIsError)
    }
}
