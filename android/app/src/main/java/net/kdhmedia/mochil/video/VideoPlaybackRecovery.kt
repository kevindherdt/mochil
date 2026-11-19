package net.kdhmedia.mochil.video

internal sealed interface VideoRecoveryAction {
    data class Retry(val attempt: Int) : VideoRecoveryAction
    data object Fallback : VideoRecoveryAction
    data class Terminal(val failure: VideoPlaybackFailure?) : VideoRecoveryAction
}

/** Owns the bounded retry and one-shot compatibility-fallback lifecycle. */
internal class VideoPlaybackRecovery {
    private var attempts = 0
    private var fallbackAttempted = false
    private var mediaId = ""

    fun onError(failure: VideoPlaybackFailure, httpStatus: Int?, hasFallback: Boolean): VideoRecoveryAction {
        if (shouldRetryVideoPlayback(failure, httpStatus, attempts)) {
            attempts += 1
            return VideoRecoveryAction.Retry(attempts)
        }
        return fallbackOrTerminal(hasFallback, failure)
    }

    fun onStall(hasFallback: Boolean): VideoRecoveryAction {
        if (attempts < MAX_VIDEO_RECONNECT_ATTEMPTS) {
            attempts += 1
            return VideoRecoveryAction.Retry(attempts)
        }
        return fallbackOrTerminal(hasFallback, null)
    }

    fun ready() {
        attempts = 0
    }

    fun manualRetry() {
        attempts = 0
    }

    fun transition(nextMediaId: String) {
        if (nextMediaId == mediaId) return
        mediaId = nextMediaId
        attempts = 0
        fallbackAttempted = false
    }

    private fun fallbackOrTerminal(hasFallback: Boolean, failure: VideoPlaybackFailure?): VideoRecoveryAction {
        if (hasFallback && !fallbackAttempted) {
            fallbackAttempted = true
            attempts = 0
            return VideoRecoveryAction.Fallback
        }
        return VideoRecoveryAction.Terminal(failure)
    }
}
