package net.kdhmedia.mochil.audio

import com.sun.net.httpserver.HttpServer
import java.net.InetSocketAddress
import net.kdhmedia.mochil.auth.DeviceCredential
import org.json.JSONObject
import org.junit.Assert.*
import org.junit.Test

class MusicPlaylistOverviewTest {
    @Test fun `summary contains count and bounded artwork without track objects`() {
        val result = parsePlaylistSummaries(JSONObject("""{"version":1,"playlists":[{"name":"Saved","item_count":200,"artwork_urls":["/cover/one"]}]}"""))
        assertEquals(listOf(MusicPlaylistOverview("Saved", 200, listOf("/cover/one"))), result)
    }

    @Test fun `invalid summaries are rejected rather than displayed as empty playlists`() {
        for (body in listOf(
            """{"version":2,"playlists":[]}""",
            """{"version":1,"playlists":null}""",
            """{"version":1,"playlists":[{"name":"Saved","item_count":-1,"artwork_urls":[]}]}""",
            """{"version":1,"playlists":[{"name":"Saved","item_count":10001,"artwork_urls":[]}]}""",
            """{"version":1,"playlists":[{"name":"Saved","item_count":2,"artwork_urls":["1","2","3","4","5"]}]}""",
        )) assertTrue(runCatching { parsePlaylistSummaries(JSONObject(body)) }.isFailure)
        assertTrue(parsePlaylistSummaries(JSONObject("""{"version":1,"playlists":[]}""")).isEmpty())
    }

    @Test fun `overview negotiates summaries and accepts legacy arrays without another request`() {
        val legacy = """[{"name":"Saved","items":[{"artwork_url":"/cover/one"},{"artwork_url":"/cover/one"}]}]"""
        val summary = """{"version":1,"playlists":[{"name":"Saved","item_count":2,"artwork_urls":["/cover/one"]}]}"""
        for (body in listOf(legacy, summary)) {
            val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
            val queries = java.util.Collections.synchronizedList(mutableListOf<String>())
            server.createContext("/api/android/v1/audio-playlists") { exchange ->
                queries.add(exchange.requestURI.rawQuery)
                val bytes = body.encodeToByteArray()
                exchange.sendResponseHeaders(200, bytes.size.toLong())
                exchange.responseBody.use { it.write(bytes) }
            }
            server.start()
            try {
                val credential = DeviceCredential("http://127.0.0.1:${server.address.port}", "device", "token")
                assertEquals(listOf(MusicPlaylistOverview("Saved", 2, listOf("/cover/one"))), MusicPlaylistClient().overview(credential))
                assertEquals(listOf("view=summary"), queries)
            } finally { server.stop(0) }
        }
    }

    @Test fun `summary numbers cannot be coerced truncated or wrapped`() {
        for (invalid in listOf("1.5", "4294967297", "1e100", "\"1\"", "true", "null")) {
            assertTrue("version=$invalid", runCatching {
                parsePlaylistSummaries(JSONObject("""{"version":$invalid,"playlists":[]}"""))
            }.isFailure)
            assertTrue("count=$invalid", runCatching {
                parsePlaylistSummaries(JSONObject("""{"version":1,"playlists":[{"name":"Saved","item_count":$invalid,"artwork_urls":[]}]}"""))
            }.isFailure)
        }
        for (valid in listOf("0", "10000", "2.0")) {
            val result = parsePlaylistSummaries(JSONObject("""{"version":1,"playlists":[{"name":"Saved","item_count":$valid,"artwork_urls":[]}]}"""))
            assertEquals(valid.toDouble().toInt(), result.single().trackCount)
        }
    }
}
