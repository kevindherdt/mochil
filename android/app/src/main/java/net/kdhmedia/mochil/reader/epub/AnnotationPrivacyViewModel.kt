package net.kdhmedia.mochil.reader.epub

import android.app.Application
import android.net.Uri
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import java.net.HttpURLConnection
import java.net.URI
import java.io.File
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import net.kdhmedia.mochil.auth.DeviceCredential
import net.kdhmedia.mochil.network.readBoundedResponseText
import net.kdhmedia.mochil.auth.DeviceCredentialStore
import org.json.JSONObject

data class AnnotationPrivacyState(
    val exporting: Boolean = false,
    val deleting: Boolean = false,
    val message: String? = null,
    val isError: Boolean = false,
)

class AnnotationPrivacyViewModel(application: Application) : AndroidViewModel(application) {
    private val mutableState = MutableStateFlow(AnnotationPrivacyState())
    val state: StateFlow<AnnotationPrivacyState> = mutableState.asStateFlow()
    private val credentialStore = DeviceCredentialStore(application)

    fun exportTo(destination: Uri) {
        if (mutableState.value.exporting || mutableState.value.deleting) return
        mutableState.value = AnnotationPrivacyState(exporting = true)
        viewModelScope.launch {
            runCatching { export(destination) }
                .onSuccess { mutableState.value = AnnotationPrivacyState(message = "Annotation export saved.") }
                .onFailure {
                    mutableState.value = AnnotationPrivacyState(message = annotationPrivacyFailure((it as? AnnotationPrivacyException)?.status, "export"), isError = true)
                }
        }
    }

    fun deleteServerCopy(confirmation: String) {
        if (confirmation != DELETE_CONFIRMATION || mutableState.value.exporting || mutableState.value.deleting) return
        mutableState.value = AnnotationPrivacyState(deleting = true)
        viewModelScope.launch {
            runCatching { delete(confirmation) }
                .onSuccess { count -> mutableState.value = AnnotationPrivacyState(message = "$count server ${if (count == 1) "annotation" else "annotations"} deleted.") }
                .onFailure { mutableState.value = AnnotationPrivacyState(message = annotationPrivacyFailure((it as? AnnotationPrivacyException)?.status, "delete"), isError = true) }
        }
    }

    fun clearMessage() {
        if (!mutableState.value.exporting && !mutableState.value.deleting) mutableState.value = AnnotationPrivacyState()
    }

    private suspend fun export(destination: Uri) = withContext(Dispatchers.IO) {
        val credential = credentialStore.load() ?: throw AnnotationPrivacyException(401)
        val connection = connection(credential, "/api/android/v1/annotations/export", "GET")
        try {
            val status = connection.responseCode
            if (status !in 200..299) throw AnnotationPrivacyException(status)
            val declared = connection.contentLengthLong
            if (declared > MAX_EXPORT_BYTES) throw IllegalStateException("annotation export is too large")
            val application = getApplication<Application>()
            val temporary = File.createTempFile("annotation-export-", ".json", application.cacheDir)
            try {
                temporary.outputStream().buffered().use { target -> connection.inputStream.use { source ->
                    val buffer = ByteArray(DEFAULT_BUFFER_SIZE)
                    var written = 0L
                    while (true) {
                        val read = source.read(buffer)
                        if (read < 0) break
                        written += read
                        if (written > MAX_EXPORT_BYTES) error("annotation export is too large")
                        target.write(buffer, 0, read)
                    }
                    target.flush()
                } }
                val output = application.contentResolver.openOutputStream(destination, "w") ?: error("The selected file could not be opened.")
                output.use { target -> temporary.inputStream().buffered().use { source -> source.copyTo(target) } }
            } finally {
                temporary.delete()
            }
        } finally {
            connection.disconnect()
        }
    }

    private suspend fun delete(confirmation: String): Int = withContext(Dispatchers.IO) {
        val credential = credentialStore.load() ?: throw AnnotationPrivacyException(401)
        val body = JSONObject().put("confirmation", confirmation).toString().encodeToByteArray()
        val connection = connection(credential, "/api/android/v1/annotations/account", "DELETE").apply {
            doOutput = true
            setRequestProperty("Content-Type", "application/json")
            setFixedLengthStreamingMode(body.size)
        }
        try {
            connection.outputStream.use { it.write(body) }
            val status = connection.responseCode
            if (status !in 200..299) throw AnnotationPrivacyException(status)
            val response = connection.inputStream.use { it.readBoundedResponseText(MAX_DELETE_RESPONSE_BYTES) }
            JSONObject(response).optInt("deleted").coerceAtLeast(0)
        } finally {
            connection.disconnect()
        }
    }

    private fun connection(credential: DeviceCredential, path: String, method: String) =
        (URI(credential.serverUrl + path).toURL().openConnection() as HttpURLConnection).apply {
            requestMethod = method
            connectTimeout = 10_000
            readTimeout = 30_000
            instanceFollowRedirects = false
            setRequestProperty("Authorization", "Bearer ${credential.token}")
            setRequestProperty("Accept", "application/json")
        }

    companion object {
        const val DELETE_CONFIRMATION = "DELETE ALL ANNOTATIONS"
        private const val MAX_EXPORT_BYTES = 256L * 1024 * 1024
        private const val MAX_DELETE_RESPONSE_BYTES = 4096
    }
}

private class AnnotationPrivacyException(val status: Int) : RuntimeException()

internal fun annotationPrivacyFailure(status: Int?, action: String): String = when (status) {
    401 -> "Your session expired. Sign in again before trying to $action annotations."
    409 -> if (action == "export") "There are no annotations to export yet." else "Annotations changed while deletion was requested. Try again."
    413 -> "The annotation data is too large to $action on this device."
    else -> if (action == "export") "Annotations could not be exported. Check the connection and try again." else "The server copy could not be deleted. Check the connection and try again."
}
