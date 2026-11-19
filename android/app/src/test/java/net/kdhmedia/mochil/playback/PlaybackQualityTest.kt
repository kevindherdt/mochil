package net.kdhmedia.mochil.playback

import androidx.media3.common.Player
import net.kdhmedia.mochil.audio.AudioClient
import net.kdhmedia.mochil.audio.AudioTrack
import net.kdhmedia.mochil.auth.DeviceCredential
import net.kdhmedia.mochil.video.VideoClient
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class PlaybackQualityTest {
    @Test fun `quality wire values remain stable`() {
        assertEquals(listOf("maximum", "high", "medium", "low"), PlaybackQuality.entries.map(PlaybackQuality::wireValue))
    }

    @Test fun `quality defaults preserve wifi and bound mobile usage`() {
        assertEquals(PlaybackQuality.Maximum, PlaybackQualityPolicy().wifi)
        assertEquals(PlaybackQuality.High, PlaybackQualityPolicy().mobile)
    }

    @Test fun `lower profiles use quality-specific server streams`() {
        val credential = DeviceCredential("https://media.example", "device", "token")
        val track = AudioTrack("track one", "music shelf", "Title", "Artist", "Album", 120.0)
        assertEquals("https://media.example/api/android/v1/audio-stream/music%20shelf/track%20one/audio", AudioClient().playbackUrl(credential, track, PlaybackQuality.Maximum))
        assertEquals("https://media.example/api/android/v1/audio-hls/music%20shelf/track%20one/low/index.m3u8", AudioClient().playbackUrl(credential, track, PlaybackQuality.Low))
        assertEquals("/api/video-hls/tv/item/medium/index.m3u8", VideoClient().playbackPath("/api/video-hls/tv/item/index.m3u8", PlaybackQuality.Medium))
    }

    @Test fun `a small file is played as it is and a failed converted stream falls back to the original`() {
        val credential = DeviceCredential("https://media.example", "device", "token")
        val small = AudioTrack("t", "music", "Title", "Artist", "Album", 120.0, bitrate = 128_000)
        val large = AudioTrack("t", "music", "Title", "Artist", "Album", 120.0, bitrate = 900_000)
        assertEquals("https://media.example/api/android/v1/audio-stream/music/t/audio", AudioClient().playbackUrl(credential, small, PlaybackQuality.High))
        assertEquals("https://media.example/api/android/v1/audio-hls/music/t/high/index.m3u8", AudioClient().playbackUrl(credential, large, PlaybackQuality.High))
        assertEquals("https://media.example/api/android/v1/audio-stream/music%20shelf/track%20one/audio",
            originalStreamUrl("https://media.example/api/android/v1/audio-hls/music%20shelf/track%20one/low/index.m3u8"))
        assertEquals(null, originalStreamUrl("https://media.example/api/android/v1/audio-stream/music/t/audio"))
    }

    @Test fun `automatic converted-track transitions retain play intent`() {
        assertTrue(shouldContinueAfterAutomaticTransition(Player.MEDIA_ITEM_TRANSITION_REASON_AUTO, false))
        assertFalse(shouldContinueAfterAutomaticTransition(Player.MEDIA_ITEM_TRANSITION_REASON_AUTO, true))
        assertFalse(shouldContinueAfterAutomaticTransition(Player.MEDIA_ITEM_TRANSITION_REASON_SEEK, false))
        assertFalse(shouldContinueAfterAutomaticTransition(Player.MEDIA_ITEM_TRANSITION_REASON_PLAYLIST_CHANGED, false))
    }
}
