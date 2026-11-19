package net.kdhmedia.mochil.playback

import org.junit.Assert.assertEquals
import org.junit.Test

class PlaybackDiagnosticsTest {
    @Test fun categorizesMediaErrorsWithoutRecordingPrivateTrackDetails() {
        assertEquals("audio_decoder_error", playbackDiagnosticName("ERROR_CODE_DECODING_FORMAT_UNSUPPORTED"))
        assertEquals("audio_network_error", playbackDiagnosticName("ERROR_CODE_IO_NETWORK_CONNECTION_TIMEOUT"))
        assertEquals("audio_source_error", playbackDiagnosticName("ERROR_CODE_IO_BAD_HTTP_STATUS"))
        assertEquals("audio_parsing_error", playbackDiagnosticName("ERROR_CODE_PARSING_CONTAINER_MALFORMED"))
        assertEquals("audio_unspecified_error", playbackDiagnosticName("ERROR_CODE_UNSPECIFIED"))
    }
}
