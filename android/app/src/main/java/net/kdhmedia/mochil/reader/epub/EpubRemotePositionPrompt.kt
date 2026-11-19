package net.kdhmedia.mochil.reader.epub

import android.app.AlertDialog
import android.content.Context
import kotlin.coroutines.resume
import kotlinx.coroutines.suspendCancellableCoroutine

/** Asks whether to jump to a newer reading position found on the server; false keeps the current one. */
internal suspend fun Context.confirmRemotePosition(position: ServerReadingPosition): Boolean = suspendCancellableCoroutine { continuation ->
    val dialog = AlertDialog.Builder(this)
        .setTitle("Newer reading position")
        .setMessage(readingPositionPromptMessage(position))
        .setPositiveButton("Go there") { _, _ -> if (continuation.isActive) continuation.resume(true) }
        .setNegativeButton("Keep this position") { _, _ -> if (continuation.isActive) continuation.resume(false) }
        .setOnCancelListener { if (continuation.isActive) continuation.resume(false) }
        .create()
    continuation.invokeOnCancellation { dialog.dismiss() }
    dialog.show()
}
