package net.kdhmedia.mochil.webdav

import android.content.Context
import java.net.URI
import java.security.MessageDigest
import java.util.UUID
import net.kdhmedia.mochil.auth.EncryptedPayloadStore
import net.kdhmedia.mochil.auth.connectionsWith
import net.kdhmedia.mochil.auth.requireSaved
import org.json.JSONArray
import org.json.JSONObject

class WebDavConnectionStore(context: Context) {
    private val encrypted = EncryptedPayloadStore(context, "webdav_connections", "mochil_webdav_connections")

    @Synchronized fun list(): List<WebDavConnection> = decode().sortedBy { it.name.lowercase() }

    @Synchronized fun put(name: String, url: String, username: String, password: String): WebDavConnection {
        require(name.trim().length in 1..120) { "Enter a connection name." }
        require(username.length <= 320 && password.length <= 2_048) { "The WebDAV credentials are too long." }
        require((username.isBlank()) == password.isBlank()) { "Enter both the WebDAV username and password, or leave both empty." }
        val normalized = validatedWebDavUrl(url).toString()
        val connection = WebDavConnection(webDavConnectionId(normalized, username), name.trim(), normalized, username, password, UUID.randomUUID().toString())
        save(connectionsWith(decode(strict = true), connection, MAX_CONNECTIONS) { it.id })
        return connection
    }

    @Synchronized fun update(id: String, name: String, url: String, username: String, password: String): WebDavConnection {
        val values = decode(strict = true)
        val existing = values.firstOrNull { it.id == id } ?: error("This WebDAV connection no longer exists.")
        val connection = updatedWebDavConnection(existing, name, url, username, password, values)
        save(values.map { if (it.id == id) connection else it })
        return connection
    }

    @Synchronized fun remove(id: String) = save(decode(strict = true).filterNot { it.id == id })

    // Mutations read strictly: unreadable storage must stop the change, not be replaced by an empty list.
    private fun decode(strict: Boolean = false): List<WebDavConnection> = runCatching {
        val values = JSONArray((if (strict) encrypted.loadStrict() else encrypted.load())?.decodeToString() ?: "[]")
        List(minOf(values.length(), MAX_CONNECTIONS)) { index -> values.getJSONObject(index).run {
            val id = getString("id")
            WebDavConnection(id, getString("name").take(120), validatedWebDavUrl(getString("url")).toString(), optString("username").take(320), optString("password").take(2_048), optString("cache_generation", id).take(64))
        } }
    }.let { if (strict) it.getOrThrow() else it.getOrDefault(emptyList()) }

    private fun save(connections: List<WebDavConnection>) {
        val payload = JSONArray().apply { connections.take(MAX_CONNECTIONS).forEach { value -> put(JSONObject().apply {
            put("id", value.id); put("name", value.name); put("url", value.rootUrl); put("username", value.username); put("password", value.password); put("cache_generation", value.cacheGeneration)
        }) } }.toString().encodeToByteArray()
        require(payload.size <= MAX_PAYLOAD_BYTES) { "Too many WebDAV connections are configured." }
        requireSaved(encrypted.save(payload))
    }

    companion object { const val MAX_CONNECTIONS = 50; const val MAX_PAYLOAD_BYTES = 256 * 1024 }
}

internal fun validatedWebDavUrl(raw: String): URI {
    val uri = URI(raw.trim())
    require(uri.scheme.equals("https", true) && !uri.host.isNullOrBlank() && uri.userInfo == null && uri.fragment == null) { "Use a credential-free HTTPS WebDAV address." }
    return URI(uri.scheme.lowercase(), null, uri.host.lowercase(), uri.port, if (uri.path.isNullOrBlank()) "/" else uri.path, uri.query, null)
}

internal fun webDavConnectionId(url: String, username: String): String = MessageDigest.getInstance("SHA-256")
    .digest("$url\u0000$username".encodeToByteArray()).take(12).joinToString("") { "%02x".format(it) }

internal fun updatedWebDavConnection(existing: WebDavConnection, name: String, url: String, username: String, password: String, all: List<WebDavConnection> = listOf(existing)): WebDavConnection {
    require(name.trim().length in 1..120) { "Enter a connection name." }
    require(username.length <= 320 && password.length <= 2_048) { "The WebDAV credentials are too long." }
    require((username.isBlank()) == password.isBlank()) { "Enter both the WebDAV username and password, or leave both empty." }
    val normalized = validatedWebDavUrl(url).toString()
    require(all.none { it.id != existing.id && it.rootUrl == normalized && it.username == username }) { "This WebDAV connection already exists." }
    val credentialsChanged = normalized != existing.rootUrl || username != existing.username || password != existing.password
    return WebDavConnection(existing.id, name.trim(), normalized, username, password, if (credentialsChanged) UUID.randomUUID().toString() else existing.cacheGeneration)
}
