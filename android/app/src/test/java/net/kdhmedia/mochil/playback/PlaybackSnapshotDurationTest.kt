package net.kdhmedia.mochil.playback

import androidx.media3.common.PlaybackParameters
import androidx.media3.common.Player
import java.lang.reflect.Proxy
import net.kdhmedia.mochil.audio.AudioTrack
import net.kdhmedia.mochil.auth.DeviceCredential
import org.junit.Assert.assertEquals
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.RuntimeEnvironment
import org.robolectric.annotation.Config

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35])
class PlaybackSnapshotDurationTest {
    @Test fun `restored queue retains known duration for departing progress`() {
        val store = PlaybackSnapshotStore(RuntimeEnvironment.getApplication())
        store.clear(durable = true)
        val track = AudioTrack("episode", "library", "Title", "Artist", "Album", 1234.5, libraryKind = "podcasts")
        val item = track.toPlaybackMediaItem("https://example.test/audio")
        val player = Proxy.newProxyInstance(Player::class.java.classLoader, arrayOf(Player::class.java)) { _, method, _ ->
            when (method.name) {
                "getMediaItemCount" -> 1
                "getCurrentMediaItemIndex" -> 0
                "getMediaItemAt" -> item
                "getCurrentPosition" -> 123L
                "getPlaybackParameters" -> PlaybackParameters.DEFAULT
                "getRepeatMode" -> Player.REPEAT_MODE_OFF
                "getShuffleModeEnabled" -> false
                else -> error("Unexpected player call: ${method.name}")
            }
        } as Player

        store.save(player, durable = true)
        val restored = requireNotNull(store.restore(DeviceCredential("https://example.test", "device", "token")))
        assertEquals(1234.5, restored.items.single().toPlaybackTrack().durationSeconds, 0.0)
    }
}
