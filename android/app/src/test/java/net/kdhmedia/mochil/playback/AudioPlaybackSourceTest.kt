package net.kdhmedia.mochil.playback

import org.junit.Assert.assertEquals
import org.junit.Test

class AudioPlaybackSourceTest {
    @Test fun convertedAudioUrlsExposeTheirEffectiveQuality() {
        assertEquals(
            AudioPlaybackSource("converted", "medium"),
            audioPlaybackSource("https://books.example/api/android/v1/audio-hls/music/track/medium/index.m3u8"),
        )
        assertEquals(
            AudioPlaybackSource("converted", "low"),
            audioPlaybackSource("https://books.example/api/android/v1/audio-hls/music%2Farchive/track%2Fid/low/index.m3u8"),
        )
    }

    @Test fun directAndMalformedUrlsRemainOriginal() {
        assertEquals(AudioPlaybackSource("original", "maximum"), audioPlaybackSource("https://books.example/audio/track.flac"))
        assertEquals(AudioPlaybackSource("original", "maximum"), audioPlaybackSource("not a URI"))
    }
}
