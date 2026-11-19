package net.kdhmedia.mochil.hermes

import org.junit.Assert.assertEquals
import org.junit.Test

/**
 * Regression test for a real production crash (server crash report, 2026-11-09, app 2026.11.101):
 * java.lang.IllegalStateException at android.media.AudioTrack.write, thrown from HermesSpeaker while
 * reading a message aloud. AudioTrack.write() throws IllegalStateException when the track was
 * paused/flushed/released concurrently on another thread - which [HermesSpeaker.stop] does routinely
 * (called from leave()/onCleared() when the user backs out of the chat, or from a new begin()
 * interrupting an old reply) while the play() coroutine is still mid-sentence on a background thread.
 * That race was previously uncaught, crashing the whole app process back to the library.
 */
class HermesSpeakerAudioTrackRaceTest {
    private fun newSpeaker() = HermesSpeaker(
        client = HermesClient(),
        credential = { null },
        voice = { null },
        onChange = {},
    )

    @Test fun writeBytesSwallowsIllegalStateExceptionFromAConcurrentlyReleasedTrack() {
        val speaker = newSpeaker()
        val bytes = ByteArray(8) { it.toByte() }

        // Simulates AudioTrack.write() throwing because stop()/release() ran on another thread mid-write -
        // exactly what the crash report showed. Must not propagate.
        val framesWritten = speaker.writeBytes(bytes) { _, _ -> throw IllegalStateException("AudioTrack has been released") }

        assertEquals("a track that throws mid-write contributes 0 frames, not a crash", 0L, framesWritten)
    }

    @Test fun writeBytesStopsCleanlyOnceTheSinkReportsNoMoreRoom() {
        val speaker = newSpeaker()
        val bytes = ByteArray(8) { it.toByte() }

        val framesWritten = speaker.writeBytes(bytes) { _, length -> length }

        assertEquals("a healthy sink that accepts everything writes the whole buffer", 4L, framesWritten)
    }

    @Test fun writeBytesHandlesAPartialWriteFollowedByARelease() {
        val speaker = newSpeaker()
        val bytes = ByteArray(8) { it.toByte() }
        var calls = 0

        // First call writes half, second call (the race) throws - must return the partial progress, not crash.
        val framesWritten = speaker.writeBytes(bytes) { _, length ->
            calls += 1
            if (calls == 1) length / 2 else throw IllegalStateException("AudioTrack has been released")
        }

        assertEquals("partial progress before the race is preserved", 2L, framesWritten)
    }
}
