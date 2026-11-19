package net.kdhmedia.mochil.webdav

import java.security.MessageDigest

/**
 * A stable identity for one WebDAV resource and version.
 *
 * This used to be `href.hashCode()`: a 32-bit Java string hash, so two different
 * paths could collide and the second one then reused the first one's cached file
 * without a request — the reader opened the wrong book. The digest covers the
 * whole resource identity, and the ETag keeps a replaced file from reusing the
 * copy cached before it changed.
 */
internal fun webDavResourceId(href: String, etag: String): String {
    val digest = MessageDigest.getInstance("SHA-256").digest("$href\u0000$etag".toByteArray(Charsets.UTF_8))
    return digest.joinToString("") { "%02x".format(it) }.take(32)
}
