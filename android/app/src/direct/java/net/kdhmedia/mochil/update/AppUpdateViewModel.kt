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
                    // Nothing reached the installer without being checked first: the installer's own
                    // failure is "App not installed" for a missing file, a downgrade and a wrong key
                    // alike, which tells the reader nothing about which one it was.
                    val problem = rejection(apk)
                    if (problem == null) {
                        mutableState.value = mutableState.value.copy(downloading = false, message = "Update downloaded. Continue in the Android installer.")
                        openInstaller(apk)
                        return@onSuccess
                    }
                    // The server offered this update, so the file it served should have been usable.
                    // A self-hosted copy of the APK is the one piece of an update that goes stale
                    // silently, so fall back once to the published release instead of dead-ending
                    // the reader on a message about their server's file.
                    apk.delete()
                    mutableState.value = mutableState.value.copy(message = "The server's update file is out of date. Trying the published release…")
                    runCatching { withContext(Dispatchers.IO) { downloadPublished(apk) } }
                        .onSuccess {
                            val publishedProblem = rejection(apk)
                            if (publishedProblem != null) {
                                apk.delete()
                                mutableState.value = AppUpdateState(message = "$problem The published release could not be used either: $publishedProblem")
                                return@onSuccess
                            }
                            mutableState.value = mutableState.value.copy(
                                downloading = false,
                                message = "Update downloaded from the published release. Continue in the Android installer.",
                            )
                            openInstaller(apk)
                        }
                        .onFailure { failure ->
                            mutableState.value = AppUpdateState(
                                message = "$problem The published release could not be reached either: ${failure.message ?: "no connection"}.",
                            )
                        }
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

    /** Null when the file can be installed, otherwise the reason it cannot. */
    private fun rejection(apk: File): String? {
        val app = getApplication<Application>()
        return updateRejection(
            expectedPackage = app.packageName,
            installedVersionCode = BuildConfig.VERSION_CODE.toLong(),
            installedSignature = installedSignature(app.packageManager, app.packageName),
            downloaded = readDownloadedApk(app.packageManager, apk),
        )
    }

    /** The published release asset, for when the paired server cannot produce a usable build. */
    private fun downloadPublished(apk: File): File = downloadTo(BuildConfig.RELEASE_APK_URL, BuildConfig.RELEASE_APK_URL, apk, token = null)

    private fun download(serverUrl: String, token: String, path: String): File {
        val origin = URI(serverUrl)
        val target = origin.resolve(path)
        require(sameOrigin(origin, target)) { "The server returned an invalid update address." }
        return downloadTo(serverUrl, target.rawPath, null, token)
    }

    private fun downloadTo(baseUrl: String, path: String, existing: File?, token: String?): File {
        val connection = if (token == null) {
            // The published release redirects to its asset host, so the anonymous download follows
            // redirects. The server download never does: it carries the device token, and following a
            // redirect would hand that token to whatever host the response names.
            (URI(path).toURL().openConnection() as HttpURLConnection).apply {
                connectTimeout = 10_000
                readTimeout = 60_000
                instanceFollowRedirects = true
            }
        } else {
            open(baseUrl, path, token)
        }
        val directory = File(getApplication<Application>().cacheDir, "updates").apply { mkdirs() }
        val temporary = File(directory, "mochil-update.apk.part")
        val apk = existing ?: File(directory, "mochil-update.apk")
        try {
            connection.useResponseStream { input -> temporary.outputStream().use(input::copyTo) }
            require(temporary.length() > 0) { "The downloaded update is empty." }
            if (apk.exists() && !apk.delete()) error("Could not replace the previous update.")
            if (!temporary.renameTo(apk)) error("Could not finish the update download.")
            return apk
        } finally {
            temporary.delete()
            connection.disconnect()
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
