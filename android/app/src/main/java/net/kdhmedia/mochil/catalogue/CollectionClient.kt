package net.kdhmedia.mochil.catalogue

import java.net.HttpURLConnection
import java.net.URI
import java.net.URLEncoder
import net.kdhmedia.mochil.auth.DeviceCredential
import net.kdhmedia.mochil.network.MAX_ERROR_RESPONSE_BYTES
import net.kdhmedia.mochil.network.readBoundedResponseText
import net.kdhmedia.mochil.network.requireAtMost
import org.json.JSONArray
import org.json.JSONObject

internal class CollectionClient {
    fun list(credential: DeviceCredential, libraryId: String): List<PublicationCollection> {
        val response = request(credential, "/api/android/v1/collections?library=${encode(libraryId)}", "GET")
        val values = JSONArray(response).requireAtMost(MAX_COLLECTIONS, "Collection list")
        return List(values.length()) { index -> values.getJSONObject(index).collection() }
    }

    /** Every collection the account has, across all libraries. */
    fun listAll(credential: DeviceCredential): List<PublicationCollection> {
        val values = JSONArray(request(credential, "/api/android/v1/collections", "GET")).requireAtMost(MAX_COLLECTIONS, "Collection list")
        return List(values.length()) { index -> values.getJSONObject(index).collection() }
    }

    fun books(credential: DeviceCredential, collectionId: String): List<BookSummary> {
        val values = JSONArray(request(credential, "/api/android/v1/collections/${encode(collectionId)}/books", "GET")).requireAtMost(MAX_COLLECTION_ITEMS, "Collection")
        return List(values.length()) { index -> values.getJSONObject(index).toBookSummary() }
    }

    fun setMember(credential: DeviceCredential, collectionId: String, member: CollectionMember, add: Boolean) {
        request(credential, "/api/android/v1/collections/${encode(collectionId)}/members", if (add) "POST" else "DELETE", member.json())
    }

    fun delete(credential: DeviceCredential, collectionId: String) {
        request(credential, "/api/android/v1/collections/${encode(collectionId)}", "DELETE")
    }

    fun rename(credential: DeviceCredential, collectionId: String, name: String) {
        request(credential, "/api/android/v1/collections/${encode(collectionId)}", "PATCH", JSONObject().put("name", name))
    }

    fun create(credential: DeviceCredential, name: String, target: CollectionTarget): PublicationCollection {
        val created = JSONObject(request(credential, "/api/android/v1/collections", "POST", JSONObject()
            .put("name", name).put("library_id", target.member.libraryId)))
        val collection = created.collection()
        setMember(credential, collection.id, target.member, true)
        return collection.copy(count = 1, members = listOf(target.member))
    }

    private fun request(credential: DeviceCredential, path: String, method: String, body: JSONObject? = null): String {
        val connection = URI(credential.serverUrl + path).toURL().openConnection() as HttpURLConnection
        connection.requestMethod = method
        connection.connectTimeout = 10_000
        connection.readTimeout = 20_000
        connection.instanceFollowRedirects = false
        connection.setRequestProperty("Authorization", "Bearer ${credential.token}")
        connection.setRequestProperty("Accept", "application/json")
        if (body != null) {
            connection.doOutput = true
            connection.setRequestProperty("Content-Type", "application/json")
            connection.outputStream.use { it.write(body.toString().toByteArray()) }
        }
        return try {
            if (connection.responseCode !in 200..299) {
                val detail = connection.errorStream?.use { it.readBoundedResponseText(MAX_ERROR_RESPONSE_BYTES) }?.trim().orEmpty()
                net.kdhmedia.mochil.network.reportIfExpiredDeviceSession(connection.responseCode)
                error(detail.take(180).ifBlank { "The server returned ${connection.responseCode}." })
            }
            connection.inputStream?.use { it.readBoundedResponseText() }.orEmpty()
        } finally { connection.disconnect() }
    }

    private fun JSONObject.collection() = PublicationCollection(
        id = getString("id"), name = optString("name", "Collection"), libraryId = getString("library_id"), count = optInt("count"),
        members = optJSONArray("members")?.requireAtMost(MAX_COLLECTION_ITEMS, "Collection members")?.let { values -> List(values.length()) { values.getJSONObject(it).member() } }.orEmpty(),
        smart = has("rule") && !isNull("rule"),
        covers = optJSONArray("covers")?.let { values -> List(minOf(values.length(), 3)) { values.optString(it) }.filter(String::isNotBlank) }.orEmpty(),
    )

    private fun JSONObject.member() = CollectionMember(optString("type"), optString("library_id"), optString("book_id"), optString("series"))
    private fun CollectionMember.json() = JSONObject().put("type", type).put("library_id", libraryId).apply {
        if (bookId.isNotBlank()) put("book_id", bookId)
        if (series.isNotBlank()) put("series", series)
    }
    private fun encode(value: String) = URLEncoder.encode(value, Charsets.UTF_8.name()).replace("+", "%20")
}

private const val MAX_COLLECTIONS = 1_000
private const val MAX_COLLECTION_ITEMS = 10_000
