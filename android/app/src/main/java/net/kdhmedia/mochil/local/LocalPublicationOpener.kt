package net.kdhmedia.mochil.local

import android.content.Context
import android.content.Intent
import android.net.Uri
import androidx.core.content.FileProvider
import java.io.File
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import net.kdhmedia.mochil.offline.DownloadSpacePolicy
import net.kdhmedia.mochil.reader.comic.ComicReaderActivity
import net.kdhmedia.mochil.reader.epub.EpubReaderActivity
import net.kdhmedia.mochil.reader.pdf.PdfReaderActivity

class LocalPublicationOpener(private val context: Context) {
    suspend fun prepare(item: LocalMediaItem): Intent = withContext(Dispatchers.IO) {
        val directory = File(context.filesDir, "publications").apply { mkdirs() }
        val target = File(directory, "local-${item.stableId}.${item.extension}")
        if (!target.isFile || (item.size > 0 && target.length() != item.size)) copy(item, target)
        val progressKey = "local:${item.stableId}"
        when (item.extension) {
            "epub" -> Intent(context, EpubReaderActivity::class.java).apply {
                putExtra(EpubReaderActivity.EXTRA_PATH, target.absolutePath); putExtra(EpubReaderActivity.EXTRA_TITLE, item.title); putExtra(EpubReaderActivity.EXTRA_PROGRESS_KEY, progressKey)
            }
            "pdf" -> Intent(context, PdfReaderActivity::class.java).apply {
                putExtra(PdfReaderActivity.EXTRA_PATH, target.absolutePath); putExtra(PdfReaderActivity.EXTRA_TITLE, item.title); putExtra(PdfReaderActivity.EXTRA_PROGRESS_KEY, progressKey)
            }
            "cbz" -> Intent(context, ComicReaderActivity::class.java).apply {
                putExtra(ComicReaderActivity.EXTRA_PATH, target.absolutePath); putExtra(ComicReaderActivity.EXTRA_TITLE, item.title); putExtra(ComicReaderActivity.EXTRA_PROGRESS_KEY, progressKey)
            }
            else -> Intent(Intent.ACTION_VIEW).apply {
                setDataAndType(FileProvider.getUriForFile(context, "${context.packageName}.files", target), item.mimeType.ifBlank { "application/octet-stream" })
                addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION)
            }
        }.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
    }

    private fun copy(item: LocalMediaItem, target: File) {
        val temporary = File(target.parentFile, ".${target.name}.part")
        try {
            DownloadSpacePolicy.requireCapacity(requireNotNull(target.parentFile), item.size, MAX_LOCAL_PUBLICATION_BYTES)
            context.contentResolver.openInputStream(Uri.parse(item.documentUri))?.use { input -> temporary.outputStream().buffered().use { output ->
                DownloadSpacePolicy.copyBounded(input, output, requireNotNull(target.parentFile), MAX_LOCAL_PUBLICATION_BYTES)
            } } ?: error("This publication is no longer available.")
            require(temporary.length() > 0) { "This publication is empty." }
            if (!temporary.renameTo(target)) {
                target.delete()
                require(temporary.renameTo(target)) { "Could not cache this publication." }
            }
        } finally { temporary.delete() }
    }

    private companion object { const val MAX_LOCAL_PUBLICATION_BYTES = 8L * 1024 * 1024 * 1024 }
}
