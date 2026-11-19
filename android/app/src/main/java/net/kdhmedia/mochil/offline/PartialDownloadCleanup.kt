package net.kdhmedia.mochil.offline

import android.content.Context
import java.io.File
import java.util.concurrent.atomic.AtomicBoolean
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch

object PartialDownloadCleanup {
    private val completed = AtomicBoolean(false)
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)

    fun atProcessStart(context: Context) {
        if (completed.compareAndSet(false, true)) {
            val filesRoot = context.filesDir
            scope.launch { clean(filesRoot) }
        }
    }

    internal fun clean(filesRoot: File): Int = listOf("music", "publications").sumOf { directory ->
        val files = File(filesRoot, directory).listFiles().orEmpty()
        var removed = 0
        files.filter { it.isFile && it.name.endsWith(".part") }.forEach { partial ->
            if (!ResumableHttpDownload.isRecoverable(partial)) {
                ResumableHttpDownload.discard(partial)
                removed++
            }
        }
        files.filter { it.isFile && it.name.endsWith(".part.resume") }.forEach { metadata ->
            val partial = File(metadata.parentFile, metadata.name.removeSuffix(".resume"))
            if (!partial.exists() && metadata.delete()) removed++
        }
        removed
    }
}
