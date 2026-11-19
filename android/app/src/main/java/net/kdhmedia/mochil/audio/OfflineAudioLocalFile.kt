package net.kdhmedia.mochil.audio

import java.io.File

/** Validate the local path independently of any external copy in the same index row. */
internal fun offlineAudioLocalFile(directory: File, path: String, includeMissing: Boolean = false): File? =
    path.takeIf(String::isNotBlank)?.let(::File)?.takeIf { file ->
        runCatching {
            file.canonicalFile.parentFile == directory.canonicalFile &&
                (file.isFile || (includeMissing && !file.exists()))
        }.getOrDefault(false)
    }
