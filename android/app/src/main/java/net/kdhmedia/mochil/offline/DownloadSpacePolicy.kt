package net.kdhmedia.mochil.offline

import java.io.File
import java.io.IOException
import java.io.InputStream
import java.io.OutputStream

class InsufficientStorageException(message: String = "There is not enough free storage for this download.") : IOException(message)

object DownloadSpacePolicy {
    const val RESERVE_BYTES = 64L * 1024 * 1024
    private const val SPACE_CHECK_INTERVAL_BYTES = 8L * 1024 * 1024

    fun requireCapacity(directory: File, expectedBytes: Long, maximumBytes: Long) {
        validate(expectedBytes, directory.usableSpace, maximumBytes)?.let { throw it }
    }

    fun copyBounded(input: InputStream, output: OutputStream, directory: File, maximumBytes: Long, onProgress: (Long) -> Unit = {}): Long {
        var written = 0L
        var nextSpaceCheck = 0L
        val buffer = ByteArray(DEFAULT_BUFFER_SIZE)
        while (true) {
            val count = input.read(buffer)
            if (count < 0) break
            written += count
            if (written > maximumBytes) throw IOException("This file is too large for offline download.")
            if (written >= nextSpaceCheck) {
                if (directory.usableSpace <= RESERVE_BYTES) throw InsufficientStorageException()
                nextSpaceCheck = written + SPACE_CHECK_INTERVAL_BYTES
            }
            output.write(buffer, 0, count)
            onProgress(written)
        }
        return written
    }

    internal fun validate(expectedBytes: Long, availableBytes: Long, maximumBytes: Long): IOException? = when {
        expectedBytes > maximumBytes -> IOException("This file is too large for offline download.")
        availableBytes <= RESERVE_BYTES -> InsufficientStorageException()
        expectedBytes > 0 && expectedBytes > availableBytes - RESERVE_BYTES -> InsufficientStorageException()
        else -> null
    }
}
