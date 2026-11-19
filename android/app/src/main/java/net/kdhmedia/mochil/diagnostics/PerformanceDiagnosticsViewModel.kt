package net.kdhmedia.mochil.diagnostics

import android.app.Application
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import java.net.HttpURLConnection
import java.net.URI
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import net.kdhmedia.mochil.auth.DeviceCredentialStore

data class PerformanceDiagnosticsState(
    val enabled: Boolean = false,
    val pending: Int = 0,
    val sending: Boolean = false,
    val message: String? = null,
)

class PerformanceDiagnosticsViewModel(application: Application) : AndroidViewModel(application) {
    private val store = PerformanceDiagnosticsStore(application)
    private val credential = DeviceCredentialStore(application).load()
    private val mutableState = MutableStateFlow(snapshot())
    val state: StateFlow<PerformanceDiagnosticsState> = mutableState.asStateFlow()

    init {
        viewModelScope.launch {
            while (isActive) {
                delay(2_000)
                val current = mutableState.value
                if (!current.sending) mutableState.value = current.copy(enabled = store.enabled(), pending = store.pendingCount())
            }
        }
        viewModelScope.launch {
            while (isActive) {
                delay(AUTO_SEND_INTERVAL_MS)
                if (store.enabled() && store.pendingCount() > 0) send(automatic = true)
            }
        }
    }

    fun setEnabled(enabled: Boolean) {
        store.setEnabled(enabled)
        mutableState.value = snapshot(message = if (enabled) "Performance diagnostics enabled." else "Automatic diagnostics disabled.")
    }

    fun sendNow() = send(automatic = false)

    fun clear() {
        store.clear()
        mutableState.value = snapshot(message = "Saved diagnostics cleared.")
    }

    private fun send(automatic: Boolean) {
        val batch = store.batch() ?: return run { mutableState.value = snapshot(message = if (automatic) null else "No diagnostics are waiting.") }
        val saved = credential ?: return run { mutableState.value = snapshot(message = "This device is no longer connected.") }
        if (mutableState.value.sending) return
        mutableState.value = snapshot(sending = true)
        viewModelScope.launch {
            // A big backlog goes up in small requests. A request the server refuses can never succeed, so it is dropped
            // rather than left to block every later send.
            var next: DiagnosticBatch? = batch
            var skipped = 0
            var failure: Throwable? = null
            while (next != null && failure == null) {
                val current = next
                val result = runCatching { withContext(Dispatchers.IO) { upload(saved.serverUrl, saved.token, current.body) } }
                val error = result.exceptionOrNull()
                when {
                    error == null -> store.acknowledge(current.eventCount)
                    (error as? DiagnosticUploadException)?.status == 400 -> { store.acknowledge(current.eventCount); skipped += current.eventCount }
                    else -> failure = error
                }
                if (failure == null) next = store.batch()
            }
            mutableState.value = snapshot(message = when {
                automatic -> null
                failure != null -> diagnosticUploadMessage(failure)
                skipped > 0 -> "Diagnostics sent. $skipped events were not accepted by the server and were dropped."
                else -> "Diagnostics sent."
            })
        }
    }

    private fun upload(serverUrl: String, token: String, report: String) {
        val connection = URI(serverUrl).resolve("/api/android/v1/diagnostics").toURL().openConnection() as HttpURLConnection
        connection.requestMethod = "POST"
        connection.doOutput = true
        connection.connectTimeout = 10_000
        connection.readTimeout = 20_000
        connection.instanceFollowRedirects = false
        connection.setRequestProperty("Authorization", "Bearer $token")
        connection.setRequestProperty("Content-Type", "application/json")
        try {
            connection.outputStream.use { it.write(report.toByteArray()) }
            if (connection.responseCode !in 200..299) throw DiagnosticUploadException(connection.responseCode)
        } finally { connection.disconnect() }
    }

    private fun snapshot(sending: Boolean = false, message: String? = null) = PerformanceDiagnosticsState(store.enabled(), store.pendingCount(), sending, message)

    private companion object { const val AUTO_SEND_INTERVAL_MS = 10 * 60 * 1_000L }
}

internal class DiagnosticUploadException(val status: Int) : java.io.IOException("Diagnostics upload returned HTTP $status")

internal fun diagnosticUploadMessage(error: Throwable): String = when ((error as? DiagnosticUploadException)?.status) {
    400 -> "The saved diagnostics were rejected. Clear them, reproduce the problem, and send again."
    401, 403 -> "This device connection expired. Reconnect Mochil before sending diagnostics."
    404 -> "This server version does not accept Android diagnostics."
    429 -> "Too many requests reached the server. Wait a moment and try again."
    in 500..599 -> "The server could not save diagnostics. Try again shortly."
    else -> "Could not reach the server to send diagnostics."
}
