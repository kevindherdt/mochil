package net.kdhmedia.mochil.wiki

import java.net.HttpURLConnection
import java.net.URI
import java.net.URLEncoder
import java.nio.charset.StandardCharsets
import net.kdhmedia.mochil.auth.DeviceCredential
import net.kdhmedia.mochil.diagnostics.PerformanceDiagnostics
import net.kdhmedia.mochil.network.MAX_ERROR_RESPONSE_BYTES
import net.kdhmedia.mochil.network.openAndroidApiGet
import net.kdhmedia.mochil.network.readBoundedResponseText
import org.json.JSONArray
import org.json.JSONObject

internal interface WikiRemote {
    fun tree(credential: DeviceCredential, libraryId: String): List<WikiEntry>
    fun search(credential: DeviceCredential, libraryId: String, query: String): List<WikiEntry>
    fun backlinks(credential: DeviceCredential, libraryId: String, id: String): List<WikiEntry>
    fun page(credential: DeviceCredential, libraryId: String, id: String): WikiPage
    fun save(credential: DeviceCredential, libraryId: String, page: WikiPage, content: String)
    fun createFolder(credential: DeviceCredential, libraryId: String, path: String)
    fun createPage(credential: DeviceCredential, libraryId: String, path: String)
    fun movePage(credential: DeviceCredential, libraryId: String, page: WikiPage, path: String)
    fun trashPage(credential: DeviceCredential, libraryId: String, page: WikiPage)
}

internal class WikiClient : WikiRemote {
    fun assetUrl(credential: DeviceCredential, libraryId: String, entry: WikiEntry): String =
        "${credential.serverUrl}/api/android/v1/document-assets/${segment(libraryId)}/${segment(entry.id)}/${segment(entry.path.substringAfterLast('/'))}"

    fun asset(credential: DeviceCredential, libraryId: String, entry: WikiEntry): ByteArray {
        val path = "/api/android/v1/document-assets/${segment(libraryId)}/${segment(entry.id)}/${segment(entry.path.substringAfterLast('/'))}"
        val connection = openAndroidApiGet(credential, path, readTimeoutMillis = 30_000)
        return try {
            val status = connection.responseCode
            net.kdhmedia.mochil.network.reportIfExpiredDeviceSession(status)
            if (status !in 200..299) throw WikiException(status, "The server returned $status while saving a Wiki image.")
            val contentType = connection.contentType.orEmpty().substringBefore(';').lowercase()
            if (!contentType.startsWith("image/")) throw WikiException(status, "This Wiki attachment is not a supported image.")
            val announced = connection.contentLengthLong
            if (announced > MAX_WIKI_ASSET_BYTES) throw WikiException(413, "This Wiki image is too large to save offline.")
            connection.inputStream.use { input ->
                val output = java.io.ByteArrayOutputStream()
                val buffer = ByteArray(32 * 1024)
                var total = 0
                while (true) {
                    val read = input.read(buffer)
                    if (read < 0) break
                    total += read
                    if (total > MAX_WIKI_ASSET_BYTES) throw WikiException(413, "This Wiki image is too large to save offline.")
                    output.write(buffer, 0, read)
                }
                output.toByteArray()
            }
        } finally { connection.disconnect() }
    }

    override fun tree(credential: DeviceCredential, libraryId: String): List<WikiEntry> {
        val response = PerformanceDiagnostics.measure("network", "load_wiki_tree") { request(credential, "GET", "/api/android/v1/documents?library=${encode(libraryId)}&tree=1") }
        val values = JSONArray(response)
        if (values.length() > MAX_WIKI_ENTRIES) throw WikiException(0, "The Wiki tree is larger than this app can safely open.")
        return List(values.length()) { index -> values.getJSONObject(index).wikiEntry() }
    }

    override fun search(credential: DeviceCredential, libraryId: String, query: String): List<WikiEntry> {
        val response = PerformanceDiagnostics.measure("network", "search_wiki") {
            request(credential, "GET", "/api/android/v1/documents?library=${encode(libraryId)}&q=${encode(query)}&limit=120")
        }
        val values = JSONArray(response)
        if (values.length() > MAX_WIKI_SEARCH_RESULTS) throw WikiException(0, "The Wiki search returned too many results.")
        return List(values.length()) { index -> values.getJSONObject(index).wikiEntry() }
            .filter { it.format.equals("md", ignoreCase = true) }
    }

    override fun backlinks(credential: DeviceCredential, libraryId: String, id: String): List<WikiEntry> {
        val values = JSONArray(PerformanceDiagnostics.measure("network", "load_wiki_backlinks") {
            request(credential, "GET", "/api/android/v1/documents/${segment(libraryId)}/${segment(id)}/backlinks")
        })
        if (values.length() > MAX_WIKI_ENTRIES) throw WikiException(0, "The Wiki backlink list is larger than this app can safely open.")
        return List(values.length()) { index -> values.getJSONObject(index).wikiEntry() }
    }

    override fun page(credential: DeviceCredential, libraryId: String, id: String): WikiPage {
        val value = JSONObject(PerformanceDiagnostics.measure("network", "load_wiki_page") { request(credential, "GET", "/api/android/v1/documents/${segment(libraryId)}/${segment(id)}/text") })
        return decodeWikiPage(value, id)
    }

    override fun save(credential: DeviceCredential, libraryId: String, page: WikiPage, content: String) {
        PerformanceDiagnostics.measure("network", "save_wiki_page") { request(credential, "PUT", "/api/android/v1/vaults/${segment(libraryId)}/pages/${segment(page.id)}", JSONObject().put("revision", page.revision).put("content", content).toString()) }
    }

    override fun createFolder(credential: DeviceCredential, libraryId: String, path: String) {
        PerformanceDiagnostics.measure("network", "create_wiki_entry") { request(credential, "POST", "/api/android/v1/vaults/${segment(libraryId)}/folders", JSONObject().put("path", path).toString()) }
    }

    override fun createPage(credential: DeviceCredential, libraryId: String, path: String) {
        PerformanceDiagnostics.measure("network", "create_wiki_entry") { request(credential, "POST", "/api/android/v1/vaults/${segment(libraryId)}/pages", JSONObject().put("path", path).put("content", "# ${path.substringAfterLast('/').substringBeforeLast('.')}\n").toString()) }
    }

    override fun movePage(credential: DeviceCredential, libraryId: String, page: WikiPage, path: String) {
        PerformanceDiagnostics.measure("network", "move_wiki_page") {
            request(
                credential,
                "PATCH",
                "/api/android/v1/vaults/${segment(libraryId)}/pages/${segment(page.id)}",
                JSONObject().put("revision", page.revision).put("path", path).toString(),
            )
        }
    }

    override fun trashPage(credential: DeviceCredential, libraryId: String, page: WikiPage) {
        PerformanceDiagnostics.measure("network", "trash_wiki_page") {
            request(
                credential,
                "DELETE",
                "/api/android/v1/vaults/${segment(libraryId)}/pages/${segment(page.id)}",
                JSONObject().put("revision", page.revision).toString(),
            )
        }
    }

    private fun request(credential: DeviceCredential, method: String, path: String, body: String? = null): String {
        val connection = if (method == "GET") {
            openAndroidApiGet(credential, path)
        } else {
            mutationConnection(credential, method, path, body)
        }
        return readResponse(connection)
    }

    private fun mutationConnection(credential: DeviceCredential, method: String, path: String, body: String?): HttpURLConnection {
        val connection = URI(credential.serverUrl + path).toURL().openConnection() as HttpURLConnection
        return prepareWikiMutationConnection(connection, method, credential.token, body)
    }

    private fun readResponse(connection: HttpURLConnection): String {
        return try {
            val status = connection.responseCode
            net.kdhmedia.mochil.network.reportIfExpiredDeviceSession(status)
            val maximum = if (status in 200..299) MAX_WIKI_RESPONSE_BYTES else MAX_ERROR_RESPONSE_BYTES
            val text = (if (status in 200..299) connection.inputStream else connection.errorStream)
                ?.use { it.readBoundedResponseText(maximum) }.orEmpty()
            if (status !in 200..299) throw decodeWikiResponseError(status, text)
            text
        } finally { connection.disconnect() }
    }

    private fun encode(value: String) = URLEncoder.encode(value, StandardCharsets.UTF_8.toString())
    private fun segment(value: String) = encode(value).replace("+", "%20")
    private fun JSONObject.wikiEntry() = WikiEntry(
        optString("kind", "page"), optString("id"), getString("path"), optString("title"), optString("format"), optString("match_snippet"), optString("revision"), optString("modified"),
    )
}

private const val MAX_WIKI_RESPONSE_BYTES = 16 * 1024 * 1024
private const val MAX_WIKI_ENTRIES = 5_000
private const val MAX_WIKI_SEARCH_RESULTS = 120
private const val MAX_WIKI_ASSET_BYTES = 3 * 1024 * 1024
