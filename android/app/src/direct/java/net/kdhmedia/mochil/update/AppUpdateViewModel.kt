package net.kdhmedia.mochil.update

import android.app.Application
import android.content.Intent
import androidx.core.content.FileProvider
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import java.io.File
import java.net.HttpURLConnection
import java.net.URI
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import net.kdhmedia.mochil.BuildConfig
import net.kdhmedia.mochil.auth.DeviceCredentialStore
import org.json.JSONObject

class AppUpdateViewModel(application: Application) : AndroidViewModel(application) {
    private val credential = DeviceCredentialStore(application).load()
    private val mutableState = MutableStateFlow(AppUpdateState())
    val state: StateFlow<AppUpdateState> = mutableState.asStateFlow()

    fun check() {
        val saved = credential ?: return setMessage("This device is no longer connected.")
        if (mutableState.value.checking || mutableState.value.downloading) return
        mutableState.value = AppUpdateState(checking = true)
        viewModelScope.launch {
            runCatching { withContext(Dispatchers.IO) { requestInfo(saved.serverUrl, saved.token) } }
                .onSuccess { info ->
                    mutableState.value = if (info.versionCode > BuildConfig.VERSION_CODE) {
                        AppUpdateState(availableVersion = info.versionName, downloadPath = info.downloadPath, message = "Version ${info.versionName} is available.", releaseNotes = info.releaseNotes)
                    } else AppUpdateState(message = "Mochil is up to date.")
                }
                .onFailure { setMessage(it.message ?: "Could not check for updates.") }
        }
    }

    fun downloadAndInstall() {
        val saved = credential ?: return setMessage("This device is no longer connected.")
        val path = mutableState.value.downloadPath ?: return
        if (mutableState.value.downloading) return
        mutableState.value = mutableState.value.copy(downloading = true, message = "Downloading update…")
        viewModelScope.launch {
            runCatching { withContext(Dispatchers.IO) { download(saved.serverUrl, saved.token, path) } }
                .onSuccess { apk ->
                    mutableState.value = mutableState.value.copy(downloading = false, message = "Update downloaded. Continue in the Android installer.")
                    openInstaller(apk)
                }
                .onFailure { mutableState.value = mutableState.value.copy(downloading = false, message = it.message ?: "Could not download the update.") }
        }
    }

    private fun requestInfo(serverUrl: String, token: String): UpdateInfo {
        val connection = open(serverUrl, "/api/android/v1/app", token)
        return connection.useResponse { body ->
            val json = JSONObject(body)
            UpdateInfo(
                json.getInt("version_code"), json.getString("version_name"), json.getString("download_url"),
                json.optJSONObject("release_notes").releaseNotes(),
            )
        }
    }

    private fun download(serverUrl: String, token: String, path: String): File {
        val origin = URI(serverUrl)
        val target = origin.resolve(path)
        require(sameOrigin(origin, target)) { "The server returned an invalid update address." }
        val connection = open(serverUrl, target.rawPath, token)
        val directory = File(getApplication<Application>().cacheDir, "updates").apply { mkdirs() }
        val temporary = File(directory, "mochil-update.apk.part")
        val apk = File(directory, "mochil-update.apk")
        try {
            connection.useResponseStream { input -> temporary.outputStream().use(input::copyTo) }
            require(temporary.length() > 0) { "The downloaded update is empty." }
            if (apk.exists() && !apk.delete()) error("Could not replace the previous update.")
            if (!temporary.renameTo(apk)) error("Could not finish the update download.")
            return apk
        } finally {
            temporary.delete()
        }
    }

    private fun open(serverUrl: String, path: String, token: String): HttpURLConnection =
        URI(serverUrl + path).toURL().openConnection().let { it as HttpURLConnection }.apply {
            connectTimeout = 10_000
            readTimeout = 60_000
            instanceFollowRedirects = false
            setRequestProperty("Authorization", "Bearer $token")
            setRequestProperty("Accept", "application/json")
        }

    private fun openInstaller(apk: File) {
        val app = getApplication<Application>()
        val uri = FileProvider.getUriForFile(app, "${app.packageName}.files", apk)
        app.startActivity(Intent(Intent.ACTION_VIEW).apply {
            setDataAndType(uri, "application/vnd.android.package-archive")
            addFlags(Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_GRANT_READ_URI_PERMISSION)
        })
    }

    private fun setMessage(message: String) { mutableState.value = AppUpdateState(message = message) }

    private fun sameOrigin(left: URI, right: URI): Boolean =
        left.scheme.equals(right.scheme, true) && left.host.equals(right.host, true) && effectivePort(left) == effectivePort(right)

    private fun effectivePort(uri: URI) = if (uri.port >= 0) uri.port else if (uri.scheme.equals("https", true)) 443 else 80
}

private data class UpdateInfo(val versionCode: Int, val versionName: String, val downloadPath: String, val releaseNotes: ReleaseNotes)

private inline fun <T> HttpURLConnection.useResponse(block: (String) -> T): T = try {
    if (responseCode !in 200..299) error("The server returned $responseCode while checking for updates.")
    block(inputStream.bufferedReader().use { it.readText() })
} finally { disconnect() }

private inline fun HttpURLConnection.useResponseStream(block: (java.io.InputStream) -> Unit) = try {
    if (responseCode !in 200..299) error("The server returned $responseCode while downloading the update.")
    inputStream.use(block)
} finally { disconnect() }
