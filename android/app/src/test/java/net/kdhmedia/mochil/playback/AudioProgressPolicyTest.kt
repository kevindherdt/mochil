package net.kdhmedia.mochil.playback

import net.kdhmedia.mochil.auth.DeviceCredential
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotEquals
import org.junit.Assert.assertTrue
import org.junit.Test
import org.json.JSONObject

class AudioProgressPolicyTest {
    @Test fun `progress cache is isolated by paired device credential`() {
        val alice = DeviceCredential("https://books.example", "alice-device", "secret-a")
        val bob = DeviceCredential("https://books.example", "bob-device", "secret-b")
        assertNotEquals(audioProgressCacheKey(alice, "music", "track"), audioProgressCacheKey(bob, "music", "track"))
        assertEquals(audioProgressCacheKey(alice, "music", "track"), audioProgressCacheKey(alice.copy(token = "rotated"), "music", "track"))
    }

    @Test fun `newer remote playback is not overwritten by a stale client`() {
        assertFalse(shouldUploadAudioProgress(remotePosition = 240.0, remoteFinished = false, localPosition = 120.0, duration = 600.0))
    }

    @Test fun `conflict retry waits for active playback to catch newer remote progress`() {
        assertFalse(shouldUploadAudioProgress(remotePosition = 123.0, remoteFinished = false, localPosition = 120.0, duration = 600.0))
        assertTrue(shouldUploadAudioProgress(remotePosition = 123.0, remoteFinished = false, localPosition = 124.0, duration = 600.0))
    }

    @Test fun `finished remote playback remains finished during a stale partial save`() {
        assertFalse(shouldUploadAudioProgress(remotePosition = 600.0, remoteFinished = true, localPosition = 100.0, duration = 600.0))
        assertFalse(shouldUploadAudioProgress(remotePosition = 600.0, remoteFinished = true, localPosition = 590.0, duration = 600.0))
    }

    @Test fun `an explicit seek may intentionally replace newer remote progress`() {
        assertTrue(shouldUploadAudioProgress(240.0, false, 120.0, 600.0, explicitSeek = true))
        assertTrue(shouldUploadAudioProgress(600.0, true, 30.0, 600.0, explicitSeek = true))
        assertFalse(shouldUploadAudioProgress(240.0, false, Double.NaN, 600.0, explicitSeek = true))
    }

    @Test fun `invalid progress never reaches the server`() {
        assertFalse(shouldUploadAudioProgress(Double.NaN, false, 10.0, 60.0))
        assertFalse(shouldUploadAudioProgress(10.0, false, 20.0, 0.0))
    }

    @Test fun `Android audio progress identifies its paired device`() {
        val value = JSONObject(audioProgressRequestBody(
            positionSeconds = 59.0,
            durationSeconds = 60.0,
            revision = 7,
            explicitSeek = false,
            deviceId = "android-tv",
        ))

        assertTrue(value.getBoolean("finished"))
        assertFalse(value.getBoolean("explicit_seek"))
        org.junit.Assert.assertEquals(7L, value.getLong("expected_revision"))
        org.junit.Assert.assertEquals("android-tv", value.getString("device_id"))
    }
}
