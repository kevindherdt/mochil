package net.kdhmedia.mochil.playback

import java.io.ByteArrayInputStream
import java.io.IOException
import org.junit.Assert.*
import org.junit.Test

class PlaybackProgressResponseTest {
    @Test fun `single progress records retain fields at exact byte boundary`() {
        val json = """{"revision":42,"position_seconds":123.5,"finished":false,"device_id":"読書"}""".encodeToByteArray()
        val padded = json + ByteArray(MAX_PLAYBACK_PROGRESS_RESPONSE_BYTES - json.size) { 32 }
        val value = ByteArrayInputStream(padded).readPlaybackProgressResponse()
        assertEquals(42L, value.getLong("revision"))
        assertEquals(123.5, value.getDouble("position_seconds"), 0.0)
        assertFalse(value.getBoolean("finished"))
        assertEquals("読書", value.getString("device_id"))
    }

    @Test fun `oversized otherwise valid JSON is rejected before parsing`() {
        val bytes = "{}".encodeToByteArray() + ByteArray(MAX_PLAYBACK_PROGRESS_RESPONSE_BYTES) { 32 }
        assertThrows(IOException::class.java) { ByteArrayInputStream(bytes).readPlaybackProgressResponse() }
    }
}
