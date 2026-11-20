package net.kdhmedia.mochil.network

import net.kdhmedia.mochil.auth.DeviceSessionEvents

/**
 * A 401 anywhere on the Android API surface means the device credential has been revoked by the
 * server, never just that one request failed. Every client that talks to this API must treat it
 * the same way, so this is the single place that recognizes it and tells [DeviceSessionEvents].
 * [openAndroidApiGet] calls this itself for every read; call sites that build their own mutation
 * connections (PUT/POST/PATCH/DELETE) call it explicitly once they have a response code.
 */
internal fun reportIfExpiredDeviceSession(status: Int): Boolean {
    if (status == HTTP_UNAUTHORIZED) {
        DeviceSessionEvents.expired()
        return true
    }
    return false
}

internal const val HTTP_UNAUTHORIZED = 401

/**
 * Background audio/video/narration playback never goes through [openAndroidApiGet]: ExoPlayer's
 * own HTTP data source rejects the stream and surfaces it as a [PlaybackException] with an
 * [androidx.media3.datasource.HttpDataSource.InvalidResponseCodeException] somewhere in its cause
 * chain. This walks that chain so a revoked device token stops silently retrying a dead stream too.
 */
internal fun httpStatusFromPlaybackError(error: Throwable): Int? =
    generateSequence(error) { it.cause }.take(6)
        .filterIsInstance<androidx.media3.datasource.HttpDataSource.InvalidResponseCodeException>()
        .firstOrNull()?.responseCode
