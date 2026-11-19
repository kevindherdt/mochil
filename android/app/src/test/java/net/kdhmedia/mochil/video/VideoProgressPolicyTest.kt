package net.kdhmedia.mochil.video

import net.kdhmedia.mochil.auth.DeviceCredential
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotEquals
import org.junit.Assert.assertTrue
import org.junit.Test
import org.json.JSONObject

class VideoProgressPolicyTest {
    @Test fun `progress cache is isolated by paired device credential`() {
        val alice = DeviceCredential("https://books.example", "alice-device", "secret-a")
        val bob = DeviceCredential("https://books.example", "bob-device", "secret-b")
        assertNotEquals(videoProgressCacheKey(alice, "movies", "film"), videoProgressCacheKey(bob, "movies", "film"))
        assertEquals(videoProgressCacheKey(alice, "movies", "film"), videoProgressCacheKey(alice.copy(token = "rotated"), "movies", "film"))
    }

    @Test fun `passive stale playback cannot replace newer remote progress`() {
        assertFalse(shouldUploadVideoProgress(900.0, false, 300.0, 1_800.0, false))
    }

    @Test fun `active playback waits until it catches newer remote progress`() {
        assertFalse(shouldUploadVideoProgress(303.0, false, 300.0, 1_800.0, false))
        assertTrue(shouldUploadVideoProgress(303.0, false, 304.0, 1_800.0, false))
    }

    @Test fun `equivalent passive checkpoints do not manufacture a new revision`() {
        assertFalse(shouldUploadVideoProgress(303.0, false, 303.0, 1_800.0, false))
        assertFalse(shouldUploadVideoProgress(303.0, false, 303.2, 1_800.0, false))
    }

    @Test fun `finished remote state remains authoritative`() {
        assertFalse(shouldUploadVideoProgress(1_800.0, true, 400.0, 1_800.0, false))
        assertFalse(shouldUploadVideoProgress(1_800.0, true, 1_800.0, 1_800.0, true))
    }

    @Test fun `locally reaching the end can finish an unfinished remote item`() {
        assertTrue(shouldUploadVideoProgress(1_700.0, false, 1_800.0, 1_800.0, true))
    }

    @Test fun `an explicit seek may intentionally move progress backward`() {
        assertTrue(shouldUploadVideoProgress(900.0, false, 300.0, 1_800.0, false, explicitSeek = true))
        assertTrue(shouldUploadVideoProgress(1_800.0, true, 60.0, 1_800.0, false, explicitSeek = true))
        assertFalse(shouldUploadVideoProgress(900.0, false, 300.0, 0.0, false, explicitSeek = true))
    }

    @Test fun `Android video progress identifies its paired device`() {
        val value = JSONObject(videoProgressRequestBody(
            positionSeconds = 91.0,
            durationSeconds = 100.0,
            finished = false,
            revision = 12,
            explicitSeek = true,
            deviceId = "living-room-streamer",
        ))

        assertEquals(91.0, value.getDouble("position_seconds"), 0.0)
        assertEquals(12L, value.getLong("expected_revision"))
        assertTrue(value.getBoolean("explicit_seek"))
        assertEquals("living-room-streamer", value.getString("device_id"))
    }

    @Test fun `video device attribution is bounded before upload`() {
        val value = JSONObject(videoProgressRequestBody(1.0, 10.0, false, 0, false, "x".repeat(400)))
        assertEquals(256, value.getString("device_id").length)
    }

    @Test fun `known media revision accompanies progress without changing legacy payloads`() {
        val legacy = JSONObject(videoProgressRequestBody(1.0, 10.0, false, 0, false, "tv"))
        assertFalse(legacy.has("expected_media_revision"))
        val guarded = JSONObject(videoProgressRequestBody(1.0, 10.0, false, 0, false, "tv", mediaRevision = "file-r1"))
        assertEquals("file-r1", guarded.getString("expected_media_revision"))
    }
}
