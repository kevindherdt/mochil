package net.kdhmedia.mochil.ui

import java.util.Locale

/**
 * The one byte-count -> human string conversion for every screen that shows a file or library size
 * (book details, WebDAV browser, playlist summaries, track info). These used to be four separate
 * implementations; two were byte-for-byte identical and one (WebDavScreen's) disagreed on unit suffixes
 * -- GiB/MiB/KiB instead of the GB/MB/KB the other three already used -- which was a real, user-visible
 * inconsistency. This adopts the majority convention (binary-sized buckets, decimal-looking GB/MB/KB
 * labels) so every screen now reads the same way for the same byte count.
 */
internal fun formatByteSize(bytes: Long): String = when {
    bytes <= 0L -> ""
    bytes >= 1_073_741_824L -> String.format(Locale.US, "%.1f GB", bytes / 1_073_741_824.0)
    bytes >= 1_048_576L -> String.format(Locale.US, "%.1f MB", bytes / 1_048_576.0)
    bytes >= 1_024L -> String.format(Locale.US, "%.1f KB", bytes / 1_024.0)
    else -> "$bytes B"
}
