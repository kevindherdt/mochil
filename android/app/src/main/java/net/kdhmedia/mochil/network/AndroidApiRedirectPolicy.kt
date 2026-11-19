package net.kdhmedia.mochil.network

import java.net.HttpURLConnection
import java.net.URI
import net.kdhmedia.mochil.auth.DeviceCredential

internal const val MAX_ANDROID_API_REDIRECTS = 3

/**
 * Opens a read-only Android API request and follows only a bounded canonical
 * redirect. Device credentials never cross a host, port downgrade, or leave
 * the versioned Android API namespace.
 */
internal fun openAndroidApiGet(
    credential: DeviceCredential,
    path: String,
    connectTimeoutMillis: Int = 10_000,
    readTimeoutMillis: Int = 20_000,
): HttpURLConnection = openAndroidApiGet(
    credential,
    URI(credential.serverUrl + path),
    connectTimeoutMillis,
    readTimeoutMillis,
    MAX_ANDROID_API_REDIRECTS,
)

private fun openAndroidApiGet(
    credential: DeviceCredential,
    uri: URI,
    connectTimeoutMillis: Int,
    readTimeoutMillis: Int,
    redirectsRemaining: Int,
): HttpURLConnection {
    val connection = (uri.toURL().openConnection() as HttpURLConnection).apply {
        requestMethod = "GET"
        connectTimeout = connectTimeoutMillis
        readTimeout = readTimeoutMillis
        instanceFollowRedirects = false
        setRequestProperty("Authorization", "Bearer ${credential.token}")
        setRequestProperty("Accept", "application/json")
    }
    val status = connection.responseCode
    reportIfExpiredDeviceSession(status)
    val redirect = connection.getHeaderField("Location")?.let(uri::resolve)
    if (redirectsRemaining > 0 && status in 300..399 && redirect != null && isSafeAndroidApiRedirect(uri, redirect)) {
        connection.disconnect()
        return openAndroidApiGet(credential, redirect, connectTimeoutMillis, readTimeoutMillis, redirectsRemaining - 1)
    }
    return connection
}

internal fun isSafeAndroidApiRedirect(from: URI, to: URI): Boolean =
    from.host.equals(to.host, ignoreCase = true) &&
        (sameOrigin(from, to) || isStandardHttpsUpgrade(from, to)) &&
        to.path.startsWith("/api/android/v1/")

private fun effectivePort(uri: URI): Int = when {
    uri.port >= 0 -> uri.port
    uri.scheme.equals("https", ignoreCase = true) -> 443
    else -> 80
}

private fun sameOrigin(from: URI, to: URI) =
    from.scheme.equals(to.scheme, ignoreCase = true) && effectivePort(from) == effectivePort(to)

private fun isStandardHttpsUpgrade(from: URI, to: URI) =
    from.scheme.equals("http", ignoreCase = true) && to.scheme.equals("https", ignoreCase = true) &&
        effectivePort(from) == 80 && effectivePort(to) == 443
