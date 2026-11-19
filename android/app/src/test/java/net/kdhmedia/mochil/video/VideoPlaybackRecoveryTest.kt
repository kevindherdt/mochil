package net.kdhmedia.mochil.video

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotEquals
import org.junit.Test

class VideoPlaybackRecoveryTest {
    @Test fun playbackIdentityDoesNotAliasColonContainingIds() {
        assertNotEquals(videoPlaybackIdentityKey("a:b", "c"), videoPlaybackIdentityKey("a", "b:c"))
    }

    @Test fun transientErrorsRetryThenUseFallbackOnce() {
        val recovery = VideoPlaybackRecovery()
        assertEquals(VideoRecoveryAction.Retry(1), recovery.onError(VideoPlaybackFailure.Network, null, true))
        assertEquals(VideoRecoveryAction.Retry(2), recovery.onError(VideoPlaybackFailure.Network, null, true))
        assertEquals(VideoRecoveryAction.Retry(3), recovery.onError(VideoPlaybackFailure.Network, null, true))
        assertEquals(VideoRecoveryAction.Fallback, recovery.onError(VideoPlaybackFailure.Network, null, true))
        assertEquals(VideoRecoveryAction.Retry(1), recovery.onError(VideoPlaybackFailure.Network, null, true))
        assertEquals(VideoRecoveryAction.Retry(2), recovery.onError(VideoPlaybackFailure.Network, null, true))
        assertEquals(VideoRecoveryAction.Retry(3), recovery.onError(VideoPlaybackFailure.Network, null, true))
        assertEquals(VideoRecoveryAction.Terminal(VideoPlaybackFailure.Network), recovery.onError(VideoPlaybackFailure.Network, null, true))
    }

    @Test fun permanentFailureUsesFallbackWithoutPointlessRetries() {
        val recovery = VideoPlaybackRecovery()
        assertEquals(VideoRecoveryAction.Fallback, recovery.onError(VideoPlaybackFailure.UnsupportedCodec, null, true))
        assertEquals(VideoRecoveryAction.Terminal(VideoPlaybackFailure.UnsupportedCodec), recovery.onError(VideoPlaybackFailure.UnsupportedCodec, null, true))
    }

    @Test fun stallsAndErrorsShareOneRetryBudget() {
        val recovery = VideoPlaybackRecovery()
        assertEquals(VideoRecoveryAction.Retry(1), recovery.onStall(false))
        assertEquals(VideoRecoveryAction.Retry(2), recovery.onError(VideoPlaybackFailure.Network, null, false))
        assertEquals(VideoRecoveryAction.Retry(3), recovery.onStall(false))
        assertEquals(VideoRecoveryAction.Terminal(null), recovery.onStall(false))
    }

    @Test fun readyAndManualRetryResetAttemptsButOnlyANewItemRestoresFallback() {
        val recovery = VideoPlaybackRecovery()
        recovery.transition("episode-1")
        assertEquals(VideoRecoveryAction.Fallback, recovery.onError(VideoPlaybackFailure.InvalidMedia, null, true))
        recovery.ready()
        assertEquals(VideoRecoveryAction.Terminal(VideoPlaybackFailure.InvalidMedia), recovery.onError(VideoPlaybackFailure.InvalidMedia, null, true))
        recovery.manualRetry()
        assertEquals(VideoRecoveryAction.Retry(1), recovery.onError(VideoPlaybackFailure.Network, null, true))
        recovery.transition("episode-1")
        assertEquals(VideoRecoveryAction.Terminal(VideoPlaybackFailure.InvalidMedia), recovery.onError(VideoPlaybackFailure.InvalidMedia, null, true))
        recovery.transition("episode-2")
        assertEquals(VideoRecoveryAction.Fallback, recovery.onError(VideoPlaybackFailure.InvalidMedia, null, true))
    }
}
