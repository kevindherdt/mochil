package net.kdhmedia.mochil.playback

import androidx.media3.common.PlaybackParameters
import androidx.media3.common.Player
import java.lang.reflect.Proxy
import net.kdhmedia.mochil.audio.AudioTrack
import net.kdhmedia.mochil.auth.DeviceCredential
import org.json.JSONArray
import org.json.JSONObject
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.RuntimeEnvironment
import org.robolectric.annotation.Config

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35])
class PlaybackSnapshotCompatibilityTest {
    private val context = RuntimeEnvironment.getApplication()
    private val credential = DeviceCredential("https://example.test", "device", "token")
    private val preferences get() = context.getSharedPreferences("playback_snapshot", 0)

    @Test fun `version one snapshot without duration still restores`() {
        writeSnapshot(JSONObject().put("id", "old").put("library_id", "library"))
        val item = requireNotNull(PlaybackSnapshotStore(context).restore(credential)).items.single()
        assertEquals(0.0, item.toPlaybackTrack().durationSeconds, 0.0)
        assertEquals(42.0, item.toPlaybackTrack(42_000).durationSeconds, 0.0)
    }

    @Test fun `invalid snapshot durations do not prevent queue restoration`() {
        for (invalid in listOf(JSONObject.NULL, -2, 0, "NaN", "Infinity", "bad")) {
            writeSnapshot(JSONObject().put("id", "old").put("library_id", "library").put("duration_seconds", invalid))
            val item = requireNotNull(PlaybackSnapshotStore(context).restore(credential)).items.single()
            assertEquals(42.0, item.toPlaybackTrack(42_000).durationSeconds, 0.0)
            assertFalse(item.mediaMetadata.extras!!.containsKey("duration_seconds"))
        }
    }

    @Test fun `snapshot retains distinct durations throughout queue`() {
        val items = listOf(12.5, 98.75, 0.0).mapIndexed { index, duration ->
            AudioTrack("id$index", "library", "title", "artist", "album", duration)
                .toPlaybackMediaItem("https://example.test/audio")
        }
        val player = Proxy.newProxyInstance(Player::class.java.classLoader, arrayOf(Player::class.java)) { _, method, args ->
            when (method.name) {
                "getMediaItemCount" -> items.size
                "getCurrentMediaItemIndex" -> 1
                "getMediaItemAt" -> items[args!![0] as Int]
                "getCurrentPosition" -> 123L
                "getPlaybackParameters" -> PlaybackParameters.DEFAULT
                "getRepeatMode" -> Player.REPEAT_MODE_OFF
                "getShuffleModeEnabled" -> false
                else -> error("Unexpected player call: ${method.name}")
            }
        } as Player
        val store = PlaybackSnapshotStore(context)
        store.save(player, durable = true)
        val payload = JSONObject(preferences.getString("active", null)!!)
        assertEquals(1, payload.getInt("version"))
        assertEquals(12.5, payload.getJSONArray("items").getJSONObject(0).getDouble("duration_seconds"), 0.0)
        assertFalse(payload.getJSONArray("items").getJSONObject(2).has("duration_seconds"))
        val snapshot = store.restore(credential)
        assertNotNull(snapshot)
        assertEquals(listOf(12.5, 98.75, 0.0), snapshot!!.items.map { it.toPlaybackTrack().durationSeconds })
    }

    private fun writeSnapshot(item: JSONObject) {
        val root = JSONObject().put("version", 1).put("items", JSONArray().put(item))
        preferences.edit().putString("active", root.toString()).commit()
    }
}
