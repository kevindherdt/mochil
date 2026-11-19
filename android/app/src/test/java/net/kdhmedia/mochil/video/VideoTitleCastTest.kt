package net.kdhmedia.mochil.video

import com.sun.net.httpserver.HttpServer
import java.net.InetSocketAddress
import net.kdhmedia.mochil.auth.DeviceCredential
import org.junit.Assert.assertEquals

import org.junit.Test

class VideoTitleCastTest {
    @Test fun detailsReadsCastFromServer() {
        val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
        server.createContext("/api/android/v1/video-details/tv") { exchange ->
            val body = """{"cast":[{"name":"Actor One","character":"Hero","profile_path":"/abc.jpg"},{"name":"Actor Two"}]}""".encodeToByteArray()
            exchange.sendResponseHeaders(200, body.size.toLong()); exchange.responseBody.use { it.write(body) }
        }
        server.start()
        try {
            val details = VideoClient().details(DeviceCredential("http://127.0.0.1:${server.address.port}", "device", "token"), "tv", "Show", null)
            assertEquals(listOf(VideoCastMember("Actor One", "Hero", "/api/android/v1/tmdb-portrait?path=%2Fabc.jpg"), VideoCastMember("Actor Two")), details.cast)
        } finally { server.stop(0) }
    }
}
