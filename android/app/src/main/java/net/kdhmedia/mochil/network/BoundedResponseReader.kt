package net.kdhmedia.mochil.network

import java.io.ByteArrayOutputStream
import java.io.IOException
import java.io.InputStream

internal const val MAX_JSON_RESPONSE_BYTES = 16 * 1024 * 1024
internal const val MAX_ERROR_RESPONSE_BYTES = 64 * 1024

internal fun InputStream.readBoundedResponseText(maximum: Int = MAX_JSON_RESPONSE_BYTES): String {
    require(maximum > 0)
    val output = ByteArrayOutputStream(minOf(8192, maximum))
    val buffer = ByteArray(8192)
    var total = 0
    while (true) {
        val remaining = maximum - total
        if (remaining == 0) {
            if (read() < 0) return output.toByteArray().decodeToString()
            throw IOException("The server response was too large.")
        }
        val count = read(buffer, 0, minOf(buffer.size, remaining))
        if (count < 0) return output.toByteArray().decodeToString()
        total += count
        output.write(buffer, 0, count)
    }
}
