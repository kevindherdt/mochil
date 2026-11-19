package net.kdhmedia.mochil.catalogue

import net.kdhmedia.mochil.auth.DeviceSessionEvents

internal fun Throwable.offlineDiagnostic(): String = when (this) {
    is CatalogueException -> "Server returned HTTP ${status.coerceIn(100, 599)}. Check the server and retry."
    is java.net.UnknownHostException -> "Server name could not be resolved. Check DNS or Wi-Fi and retry."
    is java.net.SocketTimeoutException -> "Connection timed out. Check your network and retry."
    is javax.net.ssl.SSLException -> "Secure connection failed. Check the server certificate and retry."
    is java.net.ConnectException -> "Could not connect to the server. Check your network and retry."
    else -> "Could not reach the server. Check your connection and retry."
}

internal fun Throwable.userMessage(): String {
    if (isExpiredDeviceSession()) DeviceSessionEvents.expired()
    return when (this) {
        is CatalogueException -> when (status) {
            401 -> "This device connection has expired. Sign in again."
            409 -> "This shelf changed while loading. Refresh it and try again."
            else -> message ?: "Could not load this shelf."
        }
        else -> "Could not reach the server. Check your connection and try again."
    }
}

internal fun Throwable.isExpiredDeviceSession(): Boolean = this is CatalogueException && status == 401
