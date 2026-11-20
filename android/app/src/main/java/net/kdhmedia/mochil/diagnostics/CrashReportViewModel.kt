package net.kdhmedia.mochil.diagnostics

import android.app.Application
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import java.net.HttpURLConnection
import java.net.URI
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import net.kdhmedia.mochil.auth.DeviceCredentialStore

data class CrashReportState(val pending: Boolean = false, val sending: Boolean = false, val message: String? = null)

class CrashReportViewModel(application: Application) : AndroidViewModel(application) {
    private val store = CrashReportStore(application)
    private val credential = DeviceCredentialStore(application).load()
    private val mutableState = MutableStateFlow(CrashReportState(pending = store.hasPending()))
    val state: StateFlow<CrashReportState> = mutableState.asStateFlow()

    fun send() {
        val report = store.pending() ?: return
        val saved = credential ?: return updateMessage("This device is no longer connected.")
        if (mutableState.value.sending) return
        mutableState.value = mutableState.value.copy(sending = true, message = null)
        viewModelScope.launch {
            runCatching { withContext(Dispatchers.IO) { upload(saved.serverUrl, saved.token, report) } }
                .onSuccess {
                    store.clear()
                    mutableState.value = CrashReportState(message = "Crash report sent.")
                }
                .onFailure { mutableState.value = CrashReportState(pending = true, message = "Could not send the crash report.") }
        }
    }

    fun discard() {
        store.clear()
        mutableState.value = CrashReportState(message = "Crash report deleted.")
    }

    private fun upload(serverUrl: String, token: String, report: String) {
        val connection = URI(serverUrl + "/api/android/v1/crash-report").toURL().openConnection() as HttpURLConnection
        connection.requestMethod = "POST"
        connection.doOutput = true
        connection.connectTimeout = 10_000
        connection.readTimeout = 20_000
        connection.instanceFollowRedirects = false
        connection.setRequestProperty("Authorization", "Bearer $token")
        connection.setRequestProperty("Content-Type", "application/json")
        try {
            connection.outputStream.use { it.write(report.toByteArray()) }
            if (connection.responseCode !in 200..299) error("The server returned ${connection.responseCode}.")
        } finally { connection.disconnect() }
    }

    private fun updateMessage(message: String) { mutableState.value = mutableState.value.copy(message = message) }
}
