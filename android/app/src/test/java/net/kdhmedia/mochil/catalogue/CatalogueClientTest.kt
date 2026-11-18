package net.kdhmedia.mochil.catalogue

import com.sun.net.httpserver.HttpServer
import java.net.InetSocketAddress
import net.kdhmedia.mochil.auth.DeviceCredential
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class CatalogueClientTest {
    @Test fun `library directory follows safe Android API redirect`() {
        val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
        server.createContext("/api/android/v1/libraries") { exchange ->
            exchange.responseHeaders.add("Location", "/api/android/v1/canonical-libraries")
            exchange.sendResponseHeaders(303, -1); exchange.close()
        }
        server.createContext("/api/android/v1/canonical-libraries") { exchange ->
            val body = """[{"id":"books","name":"Books","kind":"books","count":2}]""".encodeToByteArray()
            exchange.sendResponseHeaders(200, body.size.toLong()); exchange.responseBody.use { it.write(body) }
        }
        server.start()
        try {
            val credential = DeviceCredential("http://127.0.0.1:${server.address.port}", "device", "token")
            assertEquals(listOf("Books"), CatalogueClient().libraries(credential).map(LibrarySummary::name))
        } finally { server.stop(0) }
    }

    @Test fun `recent book request preserves bounded filters and cursor`() {
        val query = booksRequestQuery("light novels", null, "next/token", "new story", "N", "recent")
        assertEquals("?library=light+novels&limit=60&q=new+story&bucket=N&sort=recent&cursor=next%2Ftoken", query)
    }

    @Test fun `series request distinguishes an explicit standalone group`() {
        val query = booksRequestQuery("books", "", null, "", "", "")
        assertTrue(query.contains("&series="))
    }
}
