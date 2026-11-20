package net.kdhmedia.mochil.audio

import java.io.ByteArrayOutputStream
import java.net.HttpURLConnection
import java.net.URL
import net.kdhmedia.mochil.auth.DeviceCredential
import org.junit.Assert.*
import org.junit.Test
import org.json.JSONObject
import org.json.JSONArray

class MusicPlaylistStructureRevisionTest {
    @Test fun `visible edit uses capability and retains explicit replacement behavior`() {
        val connections = mutableListOf<Connection>()
        val client = MusicPlaylistClient { uri -> Connection(uri.toURL(), 204).also(connections::add) }
        val credential = DeviceCredential("https://fixture.invalid", "device", "token")
        val tracks = listOf("one", "two").map { AudioTrack(it, "music", it, "Artist", "Album", 1.0) }
        val original = MusicPlaylist("Saved", tracks, "\"r\"", visibleEdit = true)
        client.editVisible(credential, original, tracks.takeLast(1))
        assertEquals("edit=visible", connections[0].url.query)
        val body = JSONObject(connections[0].body.toString())
        assertEquals(2, body.getJSONArray("visible_entries").length())
        assertEquals("one", body.getJSONArray("visible_entries").getJSONObject(0).getString("item_id"))
        assertEquals("two", body.getJSONArray("visible_entries").getJSONObject(1).getString("item_id"))
        assertEquals(1, body.getJSONArray("entries").length())
        assertEquals("two", body.getJSONArray("entries").getJSONObject(0).getString("item_id"))
        assertEquals(original.revision, connections[0].getRequestProperty("If-Match"))
        client.editVisible(credential, original.copy(visibleEdit = false), emptyList())
        client.put(credential, original.name, emptyList(), original.revision)
        for (connection in connections.drop(1)) {
            assertNull(connection.url.query)
            assertEquals(0, JSONArray(connection.body.toString()).length())
        }
        assertThrows(IllegalStateException::class.java) {
            client.editVisible(credential, original.copy(revision = null), emptyList())
        }
        assertEquals(3, connections.size)
    }

    private class Connection(url: URL, private val status: Int) : HttpURLConnection(url) {
        val body = ByteArrayOutputStream()
        var disconnected = false
        override fun setRequestMethod(value: String) { method = value }
        override fun getOutputStream() = body
        override fun getResponseCode() = status
        override fun disconnect() { disconnected = true }
        override fun usingProxy() = false
        override fun connect() = Unit
    }

    @Test fun `rename and delete transmit conditions and never retry conflicts`() {
        for (method in listOf("PATCH", "DELETE")) for (status in listOf(204, 412)) {
            val connections = mutableListOf<Connection>()
            val client = MusicPlaylistClient { uri -> Connection(uri.toURL(), status).also(connections::add) }
            val credential = DeviceCredential("https://fixture.invalid", "device", "token")
            val action = {
                if (method == "PATCH") client.rename(credential, "A B", "Renamed", "\"seen\"")
                else client.delete(credential, "A B", "\"seen\"")
            }
            if (status == 412) assertThrows(MusicPlaylistConflict::class.java) { action() } else action()
            assertEquals(1, connections.size)
            val connection = connections.single()
            assertEquals(method, connection.requestMethod)
            assertEquals("/api/android/v1/audio-playlists/A%20B", connection.url.path)
            assertEquals("\"seen\"", connection.getRequestProperty("If-Match"))
            assertEquals("Bearer token", connection.getRequestProperty("Authorization"))
            assertFalse(connection.instanceFollowRedirects)
            assertTrue(connection.disconnected)
            if (method == "PATCH") assertTrue(connection.body.toString().contains("Renamed"))
            else assertEquals(0, connection.body.size())
        }
    }
}
