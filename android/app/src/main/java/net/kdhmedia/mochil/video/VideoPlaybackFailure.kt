package net.kdhmedia.mochil.video

import androidx.media3.common.PlaybackException

internal enum class VideoPlaybackFailure {
    UnsupportedCodec,
    InvalidMedia,
    Network,
    Server,
    Storage,
    Unknown,
}

internal fun classifyVideoPlaybackFailure(errorCode: Int): VideoPlaybackFailure = when (errorCode) {
    PlaybackException.ERROR_CODE_DECODER_INIT_FAILED,
    PlaybackException.ERROR_CODE_DECODER_QUERY_FAILED,
    PlaybackException.ERROR_CODE_DECODING_FAILED,
    PlaybackException.ERROR_CODE_DECODING_FORMAT_EXCEEDS_CAPABILITIES,
    PlaybackException.ERROR_CODE_DECODING_FORMAT_UNSUPPORTED -> VideoPlaybackFailure.UnsupportedCodec

    PlaybackException.ERROR_CODE_PARSING_CONTAINER_MALFORMED,
    PlaybackException.ERROR_CODE_PARSING_MANIFEST_MALFORMED -> VideoPlaybackFailure.InvalidMedia

    PlaybackException.ERROR_CODE_IO_NETWORK_CONNECTION_FAILED,
    PlaybackException.ERROR_CODE_IO_NETWORK_CONNECTION_TIMEOUT -> VideoPlaybackFailure.Network

    PlaybackException.ERROR_CODE_IO_BAD_HTTP_STATUS,
    PlaybackException.ERROR_CODE_IO_INVALID_HTTP_CONTENT_TYPE -> VideoPlaybackFailure.Server

    PlaybackException.ERROR_CODE_IO_FILE_NOT_FOUND,
    PlaybackException.ERROR_CODE_IO_NO_PERMISSION -> VideoPlaybackFailure.Storage

    else -> VideoPlaybackFailure.Unknown
}

internal fun videoPlaybackFailureMessage(failure: VideoPlaybackFailure): String = when (failure) {
    VideoPlaybackFailure.UnsupportedCodec -> "This device cannot decode one of this video's audio or picture formats. Try another version of the video or a device that supports its codec."
    VideoPlaybackFailure.InvalidMedia -> "This video file is damaged or uses a container this device cannot read. Try another copy or open it in a different player."
    VideoPlaybackFailure.Network -> "Automatic reconnects were unsuccessful. Check the network and try again."
    VideoPlaybackFailure.Server -> "The server could not deliver a playable video stream. Try again in a moment."
    VideoPlaybackFailure.Storage -> "The video is no longer available at its saved location. Return to the library and open it again."
    VideoPlaybackFailure.Unknown -> "This video could not be played. Close it and try another file; a private diagnostic event has been saved."
}

internal fun offersManualVideoRetry(failure: VideoPlaybackFailure): Boolean =
    failure == VideoPlaybackFailure.Network || failure == VideoPlaybackFailure.Server || failure == VideoPlaybackFailure.Unknown

internal fun videoPlaybackDiagnosticName(failure: VideoPlaybackFailure): String = when (failure) {
    VideoPlaybackFailure.UnsupportedCodec -> "video_codec_error"
    VideoPlaybackFailure.InvalidMedia -> "video_media_error"
    VideoPlaybackFailure.Network -> "video_network_error"
    VideoPlaybackFailure.Server -> "video_server_error"
    VideoPlaybackFailure.Storage -> "video_storage_error"
    VideoPlaybackFailure.Unknown -> "playback_error"
}

internal const val MAX_VIDEO_RECONNECT_ATTEMPTS = 3

internal fun isTransientVideoHttpStatus(status: Int?): Boolean =
    status == 408 || status == 429 || status != null && status in 500..599

internal fun shouldRetryVideoPlayback(failure: VideoPlaybackFailure, httpStatus: Int?, completedAttempts: Int): Boolean =
    completedAttempts < MAX_VIDEO_RECONNECT_ATTEMPTS &&
        (failure == VideoPlaybackFailure.Network || failure == VideoPlaybackFailure.Server && isTransientVideoHttpStatus(httpStatus))

internal fun videoReconnectDelayMillis(attempt: Int): Long = when (attempt.coerceIn(1, MAX_VIDEO_RECONNECT_ATTEMPTS)) {
    1 -> 1_500L
    2 -> 3_000L
    else -> 6_000L
}
