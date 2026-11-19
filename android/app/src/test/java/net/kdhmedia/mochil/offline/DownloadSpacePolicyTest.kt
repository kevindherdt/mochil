package net.kdhmedia.mochil.offline

import java.io.ByteArrayInputStream
import java.io.ByteArrayOutputStream
import org.junit.Assert.assertArrayEquals
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class DownloadSpacePolicyTest {
    @Test fun `extreme declared size cannot overflow reserved space comparison`() {
        assertTrue(DownloadSpacePolicy.validate(Long.MAX_VALUE, Long.MAX_VALUE, Long.MAX_VALUE) is InsufficientStorageException)
        val fitting = Long.MAX_VALUE - DownloadSpacePolicy.RESERVE_BYTES
        assertNull(DownloadSpacePolicy.validate(fitting, Long.MAX_VALUE, Long.MAX_VALUE))
        assertTrue(DownloadSpacePolicy.validate(fitting + 1, Long.MAX_VALUE, Long.MAX_VALUE) is InsufficientStorageException)
    }

    @Test fun `unknown sizes still require reserve and invalid free space is rejected`() {
        for (available in listOf(Long.MIN_VALUE, -1L, 0L, DownloadSpacePolicy.RESERVE_BYTES)) {
            assertTrue(DownloadSpacePolicy.validate(-1, available, Long.MAX_VALUE) is InsufficientStorageException)
        }
        assertNull(DownloadSpacePolicy.validate(-1, DownloadSpacePolicy.RESERVE_BYTES + 1, Long.MAX_VALUE))
    }

    @Test fun `known download reserves space for safe app operation`() {
        val error = DownloadSpacePolicy.validate(100, DownloadSpacePolicy.RESERVE_BYTES + 99, 1_000)
        assertTrue(error is InsufficientStorageException)
        assertNull(DownloadSpacePolicy.validate(100, DownloadSpacePolicy.RESERVE_BYTES + 100, 1_000))
    }

    @Test fun `oversized response is rejected before transfer`() {
        assertEquals("This file is too large for offline download.", DownloadSpacePolicy.validate(1_001, Long.MAX_VALUE, 1_000)?.message)
    }

    @Test fun `bounded copy preserves exact bytes`() {
        val source = byteArrayOf(1, 2, 3, 4)
        val output = ByteArrayOutputStream()
        val directory = kotlin.io.path.createTempDirectory("mochil-space").toFile()
        try {
            assertEquals(source.size.toLong(), DownloadSpacePolicy.copyBounded(ByteArrayInputStream(source), output, directory, 8))
            assertArrayEquals(source, output.toByteArray())
        } finally { directory.deleteRecursively() }
    }
}
