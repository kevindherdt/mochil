package net.kdhmedia.mochil.update

import android.content.pm.PackageManager
import java.io.File

/** What a downloaded APK says about itself, read from the file rather than by installing it. */
internal data class DownloadedApk(
    val packageName: String,
    val versionCode: Long,
    val signature: ByteArray?,
)

/**
 * Reads the package name, version code and signing certificate out of an APK file without
 * installing it. Returns null when the file is not a readable Android package at all, which is what
 * a truncated download or an HTML login page saved under a .apk name looks like.
 */
internal fun readDownloadedApk(packageManager: PackageManager, apk: File): DownloadedApk? {
    @Suppress("DEPRECATION")
    val info = packageManager.getPackageArchiveInfo(apk.absolutePath, PackageManager.GET_SIGNATURES)
        ?: return null
    @Suppress("DEPRECATION")
    val signature: ByteArray? = info.signatures?.firstOrNull()?.toByteArray()
    return DownloadedApk(info.packageName, info.longVersionCodeCompat(), signature)
}

@Suppress("DEPRECATION")
private fun android.content.pm.PackageInfo.longVersionCodeCompat(): Long =
    if (android.os.Build.VERSION.SDK_INT >= android.os.Build.VERSION_CODES.P) longVersionCode else versionCode.toLong()

/**
 * Decides whether the downloaded file can be offered to the installer, and says why not in terms the
 * person holding the phone can act on.
 *
 * This exists because handing an unusable file to Android's installer produces the single least
 * helpful message in the platform -- "App not installed" -- whether the cause was a file that is not
 * an APK, a downgrade, or a different signing key. Every one of those is knowable before installing.
 */
internal fun updateRejection(
    expectedPackage: String,
    installedVersionCode: Long,
    installedSignature: ByteArray?,
    downloaded: DownloadedApk?,
): String? = when {
    downloaded == null ->
        "The server sent something that is not an Android app. The update file on the server is missing or out of date."
    downloaded.packageName != expectedPackage ->
        "The server sent ${downloaded.packageName}, which is not this app."
    downloaded.versionCode <= installedVersionCode ->
        "The server is offering build ${downloaded.versionCode}, which is not newer than the installed " +
            "build $installedVersionCode. The update file on the server is out of date."
    installedSignature != null && downloaded.signature != null &&
        !installedSignature.contentEquals(downloaded.signature) ->
        "The server's update was signed with a different key, so Android refuses to install it over this copy."
    else -> null
}

/** The signature bytes of the running app, or null when the platform will not report them. */
internal fun installedSignature(packageManager: PackageManager, packageName: String): ByteArray? = runCatching {
    @Suppress("DEPRECATION")
    packageManager.getPackageInfo(packageName, PackageManager.GET_SIGNATURES).signatures?.firstOrNull()?.toByteArray()
}.getOrNull()
