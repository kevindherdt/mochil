package net.kdhmedia.mochil.reader

import android.content.Context
import java.io.File
import java.io.FileOutputStream
import org.json.JSONObject

internal enum class FixedLayoutReaderKind(val preferencesName: String, val fileName: String) {
    Pdf("pdf_reader", "fixed-layout-progress-pdf.json"),
    Comic("comic_reader", "fixed-layout-progress-comic.json"),
}

/**
 * Shared exact-page persistence; remote revision reconciliation remains in
 * PdfProgressViewModel.
 *
 * Backed by a single JSON file written with the codebase's standard
 * temp-file + atomic rename pattern (see LocalMediaIndexStore, the Go
 * reading-state store, and the KOReader plugin's staged writes): an
 * interruption mid-write (backgrounded, crashed, killed, power loss) can
 * leave at most the previous complete file in place, never a partial one.
 * A missing/corrupt/truncated file degrades to "page 0" (read at the start
 * of the book) rather than poisoning the store the way a zero-length
 * SharedPreferences XML file does (NotCommitWrittenException on every
 * later edit for the rest of the process).
 */
internal class FixedLayoutLocalProgressStore(private val context: Context, private val kind: FixedLayoutReaderKind) {
    private val file = File(context.noBackupFilesDir, kind.fileName)

    fun load(progressKey: String, maximumPage: Int): Int =
        read().optInt(fixedLayoutPageKey(progressKey), 0).coerceIn(0, maximumPage.coerceAtLeast(0))

    fun save(progressKey: String, page: Int) {
        write(read().put(fixedLayoutPageKey(progressKey), page.coerceAtLeast(0)), durable = false)
    }

    fun saveNow(progressKey: String, page: Int): Boolean =
        write(read().put(fixedLayoutPageKey(progressKey), page.coerceAtLeast(0)), durable = true)

    /** One-time lift of a page (and the old, now-unused "finished" flag) out of the legacy SharedPreferences file. */
    fun migrateLegacyKey(legacyKey: String, progressKey: String) {
        val current = read()
        if (legacyKey == progressKey || current.has(fixedLayoutPageKey(progressKey))) return

        val legacyPreferences = legacyPreferences()
        val legacyPage = legacyPreferences.getInt("page:$legacyKey", Int.MIN_VALUE).takeIf { it != Int.MIN_VALUE }
        val legacyFinished = if (legacyPreferences.contains("finished:$legacyKey")) {
            legacyPreferences.getBoolean("finished:$legacyKey", false)
        } else null
        if (legacyPage == null && legacyFinished == null) return

        if (legacyPage != null) current.put(fixedLayoutPageKey(progressKey), legacyPage)
        write(current, durable = false)
        legacyPreferences.edit().remove("page:$legacyKey").remove("finished:$legacyKey").apply()
    }

    private fun read(): JSONObject = runCatching {
        val raw = file.takeIf { it.isFile && it.length() in 1..MAX_STATE_BYTES }?.readText()
        if (raw != null) JSONObject(raw) else JSONObject()
    }.getOrDefault(JSONObject())

    /**
     * Writes [values] to a sibling temp file and atomically renames it over
     * [file]. When [durable] is true (the explicit pre-backgrounding save
     * path), the temp file's bytes are fsync'd before the rename, so "saved"
     * means "on storage", not just "in the kernel page cache" -- matching
     * the Go reading-state store's temporary.Sync() before os.Rename().
     */
    private fun write(values: JSONObject, durable: Boolean): Boolean {
        val parent = file.parentFile ?: return false
        if (!parent.isDirectory) parent.mkdirs()
        val temporary = File(parent, ".${file.name}.part")
        return runCatching {
            FileOutputStream(temporary).use { stream ->
                stream.write(values.toString().toByteArray())
                if (durable) {
                    stream.flush()
                    stream.fd.sync()
                }
            }
            if (!temporary.renameTo(file)) temporary.copyTo(file, overwrite = true)
            true
        }.getOrElse {
            temporary.delete()
            false
        }
    }

    private fun legacyPreferences() = context.getSharedPreferences(kind.preferencesName, Context.MODE_PRIVATE)

    private companion object {
        const val MAX_STATE_BYTES = 1 shl 20 // 1 MiB; this file only ever holds small page-number entries.
    }
}

internal fun fixedLayoutPageKey(progressKey: String) = "page:$progressKey"
