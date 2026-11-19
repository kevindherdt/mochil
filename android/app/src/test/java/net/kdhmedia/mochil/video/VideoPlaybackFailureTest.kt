package net.kdhmedia.mochil.video

import androidx.media3.common.PlaybackException
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class VideoPlaybackFailureTest {
    @Test fun identifiesUnsupportedDecodersWithoutExposingTechnicalCodes() {
        val failure = classifyVideoPlaybackFailure(PlaybackException.ERROR_CODE_DECODING_FORMAT_UNSUPPORTED)

        assertEquals(VideoPlaybackFailure.UnsupportedCodec, failure)
        assertEquals("video_codec_error", videoPlaybackDiagnosticName(failure))
        assertTrue(videoPlaybackFailureMessage(failure).contains("codec"))
        assertTrue(!videoPlaybackFailureMessage(failure).contains(PlaybackException.ERROR_CODE_DECODING_FORMAT_UNSUPPORTED.toString()))
    }

    @Test fun separatesNetworkServerAndInvalidMediaRecovery() {
        assertEquals(VideoPlaybackFailure.Network, classifyVideoPlaybackFailure(PlaybackException.ERROR_CODE_IO_NETWORK_CONNECTION_TIMEOUT))
        assertEquals(VideoPlaybackFailure.Server, classifyVideoPlaybackFailure(PlaybackException.ERROR_CODE_IO_BAD_HTTP_STATUS))
        assertEquals(VideoPlaybackFailure.InvalidMedia, classifyVideoPlaybackFailure(PlaybackException.ERROR_CODE_PARSING_CONTAINER_MALFORMED))
        assertTrue(videoPlaybackFailureMessage(VideoPlaybackFailure.Network).contains("Check the network"))
    }

    @Test fun unknownFailuresRetainSafeDiagnosticRecovery() {
        assertEquals(VideoPlaybackFailure.Unknown, classifyVideoPlaybackFailure(Int.MAX_VALUE))
        assertEquals("playback_error", videoPlaybackDiagnosticName(VideoPlaybackFailure.Unknown))
        assertTrue(videoPlaybackFailureMessage(VideoPlaybackFailure.Unknown).contains("diagnostic"))
    }

    @Test fun retriesOnlyBoundedTransientPlaybackFailures() {
        assertTrue(shouldRetryVideoPlayback(VideoPlaybackFailure.Network, null, 0))
        assertTrue(shouldRetryVideoPlayback(VideoPlaybackFailure.Server, 503, 2))
        assertTrue(!shouldRetryVideoPlayback(VideoPlaybackFailure.Server, 404, 0))
        assertTrue(!shouldRetryVideoPlayback(VideoPlaybackFailure.Network, null, MAX_VIDEO_RECONNECT_ATTEMPTS))
        assertEquals(1_500L, videoReconnectDelayMillis(1))
        assertEquals(6_000L, videoReconnectDelayMillis(3))
    }

    @Test fun manualRetryIsOfferedOnlyWhenRepreparingCanHelp() {
        assertTrue(offersManualVideoRetry(VideoPlaybackFailure.Network))
        assertTrue(offersManualVideoRetry(VideoPlaybackFailure.Server))
        assertTrue(offersManualVideoRetry(VideoPlaybackFailure.Unknown))
        assertTrue(!offersManualVideoRetry(VideoPlaybackFailure.UnsupportedCodec))
        assertTrue(!offersManualVideoRetry(VideoPlaybackFailure.InvalidMedia))
        assertTrue(!offersManualVideoRetry(VideoPlaybackFailure.Storage))
    }
}
