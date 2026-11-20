package net.kdhmedia.mochil.reader.pdf

import android.graphics.Bitmap
import android.graphics.Matrix
import android.graphics.pdf.PdfRenderer
import android.os.ParcelFileDescriptor
import java.io.Closeable
import java.io.File
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock

class PdfDocument(file: File) : Closeable {
    private val descriptor = ParcelFileDescriptor.open(file, ParcelFileDescriptor.MODE_READ_ONLY)
    private val renderer = try {
        PdfRenderer(descriptor)
    } catch (failure: Throwable) {
        descriptor.close()
        throw failure
    }
    private val renderLock = Mutex()
    val pageCount: Int get() = renderer.pageCount

    suspend fun render(index: Int, targetWidth: Int): Bitmap = renderLock.withLock {
        require(index in 0 until pageCount) { "PDF page is outside the document." }
        renderer.openPage(index).use { page ->
            val size = boundedPdfRenderSize(page.width, page.height, targetWidth)
            Bitmap.createBitmap(size.width, size.height, Bitmap.Config.ARGB_8888).also { bitmap ->
                bitmap.eraseColor(android.graphics.Color.WHITE)
                val matrix = Matrix().apply { setScale(size.width.toFloat() / page.width, size.height.toFloat() / page.height) }
                page.render(bitmap, null, matrix, PdfRenderer.Page.RENDER_MODE_FOR_DISPLAY)
            }
        }
    }

    override fun close() {
        renderer.close()
        descriptor.close()
    }

    companion object {
        internal const val MAX_EDGE = 3_072
        internal const val MAX_PIXELS = 4_000_000L
    }
}

internal data class PdfRenderSize(val width: Int, val height: Int)

internal fun boundedPdfRenderSize(pageWidth: Int, pageHeight: Int, targetWidth: Int): PdfRenderSize {
    val sourceWidth = pageWidth.coerceAtLeast(1)
    val sourceHeight = pageHeight.coerceAtLeast(1)
    var width = targetWidth.coerceIn(320, PdfDocument.MAX_EDGE)
    var height = (sourceHeight.toDouble() * width / sourceWidth).toInt().coerceAtLeast(1)
    if (height > PdfDocument.MAX_EDGE) {
        width = (width.toDouble() * PdfDocument.MAX_EDGE / height).toInt().coerceAtLeast(1)
        height = PdfDocument.MAX_EDGE
    }
    val pixels = width.toLong() * height
    if (pixels > PdfDocument.MAX_PIXELS) {
        val reduction = kotlin.math.sqrt(PdfDocument.MAX_PIXELS.toDouble() / pixels)
        width = (width * reduction).toInt().coerceAtLeast(1)
        height = (height * reduction).toInt().coerceAtLeast(1)
    }
    return PdfRenderSize(width, height)
}
