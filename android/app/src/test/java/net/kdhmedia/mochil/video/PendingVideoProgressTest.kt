package net.kdhmedia.mochil.video

import net.kdhmedia.mochil.auth.DeviceCredential
import org.junit.Assert.*
import org.junit.Test

class PendingVideoProgressTest {
    private val credential = DeviceCredential("https://media.example", "device", "secret-token")
    private fun checkpoint() = PendingVideoProgress(videoProgressOwner(credential), "anime", "episode", 10.0, 600.0, 2, false)

    @Test fun `checkpoint round trip excludes playback and credential secrets`() {
        val value = checkpoint()
        val bytes = encodePendingVideoProgress(listOf(value))
        assertEquals(listOf(value), decodePendingVideoProgressStrict(bytes))
        assertFalse(bytes.decodeToString().contains(credential.token))
        assertFalse(bytes.decodeToString().contains(credential.serverUrl))
        assertEquals(value.owner, videoProgressOwner(credential.copy(token = "rotated")))
        assertNotEquals(value.owner, videoProgressOwner(credential.copy(deviceId = "other")))
        assertNotEquals(value.owner, videoProgressOwner(credential.copy(serverUrl = "https://other.example")))
    }

    @Test fun `latest position replaces same item and retains expiring seek intent`() {
        val first = checkpoint().copy(explicitSeekAtEpochMs = 100)
        val latest = checkpoint().copy(positionSeconds = 11.0)
        val result = mergePendingVideoProgress(listOf(first), latest).single()
        assertEquals(11.0, result.positionSeconds, 0.0)
        assertTrue(result.explicitSeekForReplay(600_100))
        assertFalse(result.explicitSeekForReplay(600_101))
        assertFalse(result.explicitSeekForReplay(99))
        assertEquals(200L, mergePendingVideoProgress(listOf(result), latest.copy(explicitSeekAtEpochMs = 200)).single().explicitSeekAtEpochMs)
    }

    @Test fun `journal bounds and invalid checkpoints fail closed`() {
        val value = checkpoint()
        val initial = List(64) { value.copy(itemId = "$it") }
        val merged = mergePendingVideoProgress(initial, value.copy(itemId = "latest"))
        assertEquals(64, merged.size)
        assertEquals("1", merged.first().itemId)
        assertEquals("latest", merged.last().itemId)
        assertThrows(IllegalArgumentException::class.java) { encodePendingVideoProgress(initial + value) }
        assertThrows(Exception::class.java) { decodePendingVideoProgressStrict(ByteArray(MAX_VIDEO_PROGRESS_JOURNAL_BYTES + 1)) }
        assertThrows(Exception::class.java) { decodePendingVideoProgressStrict("[{}]".encodeToByteArray()) }
        for (invalid in listOf(value.copy(positionSeconds = Double.NaN), value.copy(durationSeconds = 0.0),
            value.copy(positionSeconds = 601.0), value.copy(revision = -1), value.copy(owner = ""), value.copy(itemId = "x".repeat(513)))) {
            assertFalse(invalid.valid())
            assertThrows(IllegalArgumentException::class.java) { encodePendingVideoProgress(listOf(invalid)) }
        }
    }
}
