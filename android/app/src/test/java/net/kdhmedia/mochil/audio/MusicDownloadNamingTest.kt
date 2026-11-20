package net.kdhmedia.mochil.audio

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class MusicDownloadNamingTest {
    @Test fun `legacy download keys normalize through embedded track identity`() {
        val track = AudioTrack("b:c", "a", "Track", "Artist", "Album", 60.0)
        assertEquals(track.identityKey, offlineAudioRecoveryKey("a:b:c", track))
    }
    @Test fun confinesTrackKeysToOneFileName() {
        val name = MusicDownloadNaming.safeName("music:../../private/track?token=secret")
        assertTrue(name.matches(Regex("[a-f0-9]{64}")))
        assertFalse(name.contains('/'))
    }

    @Test fun `punctuation differences cannot alias another download`() {
        assertNotEquals(MusicDownloadNaming.safeName("music:a/b"), MusicDownloadNaming.safeName("music:a_b"))
        assertNotEquals(MusicDownloadNaming.safeName("music:a:b"), MusicDownloadNaming.safeName("music:a?b"))
    }

    @Test fun `long identities retain distinguishing suffixes`() {
        val prefix = "x".repeat(500)
        assertNotEquals(MusicDownloadNaming.safeName(prefix + "1"), MusicDownloadNaming.safeName(prefix + "2"))
    }

    @Test fun `unicode identities produce bounded deterministic ascii filenames`() {
        val identity = "音楽".repeat(200)
        val name = MusicDownloadNaming.safeName(identity)
        assertEquals(64, name.toByteArray(Charsets.UTF_8).size)
        assertEquals(name, MusicDownloadNaming.safeName(identity))
    }

    @Test fun preservesKnownContainerExtensions() {
        assertEquals(".flac", MusicDownloadNaming.extensionFor("audio/FLAC"))
        assertEquals(".m4a", MusicDownloadNaming.extensionFor("audio/mp4"))
        assertEquals(".mp3", MusicDownloadNaming.extensionFor("application/octet-stream"))
    }
}
