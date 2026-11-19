package net.kdhmedia.mochil.reader.epub

import java.io.File

/**
 * Resolves [requestedPath] to a real `.epub` file strictly inside [root], rejecting null paths,
 * path traversal out of [root], non-EPUB extensions, and missing files. Null on any rejection.
 */
internal fun validatedEpubPath(requestedPath: String?, root: File): File? {
    val requested = requestedPath?.let(::File)?.canonicalFile ?: return null
    val canonicalRoot = root.canonicalFile
    return requested.takeIf {
        it.isFile && it.path.startsWith(canonicalRoot.path + File.separator) && it.extension.equals("epub", true)
    }
}
