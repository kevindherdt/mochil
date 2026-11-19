package net.kdhmedia.mochil.wiki

import java.net.HttpURLConnection

/** Owns the connection until request setup succeeds; the response reader then owns it. */
internal fun prepareWikiMutationConnection(
    connection: HttpURLConnection,
    method: String,
    token: String,
    body: String?,
): HttpURLConnection {
    try {
        connection.requestMethod = method
        connection.connectTimeout = 10_000
        connection.readTimeout = 20_000
        connection.instanceFollowRedirects = false
        connection.setRequestProperty("Authorization", "Bearer $token")
        connection.setRequestProperty("Accept", "application/json")
        if (body != null) {
            connection.doOutput = true
            connection.setRequestProperty("Content-Type", "application/json")
            connection.outputStream.use { it.write(body.encodeToByteArray()) }
        }
        return connection
    } catch (failure: Throwable) {
        try {
            connection.disconnect()
        } catch (cleanup: Throwable) {
            if (cleanup !== failure) failure.addSuppressed(cleanup)
        }
        throw failure
    }
}
