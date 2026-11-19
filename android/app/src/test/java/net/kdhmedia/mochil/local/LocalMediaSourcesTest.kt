package net.kdhmedia.mochil.local

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class LocalMediaSourcesTest {
    @Test fun classifiesSupportedLocalMediaWithoutGuessingUnknownFiles() {
        assertEquals(LocalMediaKind.Publications, localMediaKind("EPUB"))
        assertEquals(LocalMediaKind.Audio, localMediaKind("m4b"))
        assertEquals(LocalMediaKind.Video, localMediaKind("mkv"))
        assertNull(localMediaKind("exe"))
    }
}
