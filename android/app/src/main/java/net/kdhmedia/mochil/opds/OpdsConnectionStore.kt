package net.kdhmedia.mochil.opds

import android.content.Context
import java.security.MessageDigest
import java.util.UUID
import net.kdhmedia.mochil.auth.EncryptedPayloadStore
import net.kdhmedia.mochil.auth.connectionsWith
import net.kdhmedia.mochil.auth.requireSaved
import org.json.JSONArray
import org.json.JSONObject

class OpdsConnectionStore(context: Context) {
    private val encrypted = EncryptedPayloadStore(context, "opds_connections", "mochil_opds_connections")

    @Synchronized
    fun list(): List<OpdsConnection> = decode().sortedBy { it.name.lowercase() }

    @Synchronized
    fun put(name: String, catalogueUrl: String, username: String, password: String): OpdsConnection {
        val normalized = validatedHttps(catalogueUrl.trim()).toString()
        require(name.trim().length in 1..120) { "Enter a catalogue name." }
        require(username.length <= 320 && password.length <= 2048) { "The OPDS credentials are too long." }
        val id = stableConnectionId(normalized, username)
        val connection = OpdsConnection(id, name.trim(), normalized, username, password, UUID.randomUUID().toString())
        save(connectionsWith(decode(strict = true), connection, MAX_CONNECTIONS) { it.id })
        return connection
    }

    @Synchronized
    fun update(id: String, name: String, catalogueUrl: String, username: String, password: String): OpdsConnection {
        val values = decode(strict = true)
        val existing = values.firstOrNull { it.id == id } ?: error("This OPDS catalogue no longer exists.")
        val connection = updatedOpdsConnection(existing, name, catalogueUrl, username, password, values)
        save(values.map { if (it.id == id) connection else it })
        return connection
    }

    @Synchronized
    fun remove(id: String) = save(decode(strict = true).filterNot { it.id == id })

    // Mutations read strictly: unreadable storage must stop the change, not be replaced by an empty list.
    private fun decode(strict: Boolean = false): List<OpdsConnection> = runCatching {
        val values = JSONArray((if (strict) encrypted.loadStrict() else encrypted.load())?.decodeToString() ?: "[]")
        List(minOf(values.length(), MAX_CONNECTIONS)) { index -> values.getJSONObject(index).run {
            val id = getString("id")
            OpdsConnection(id, getString("name").take(120), validatedHttps(getString("url")).toString(), optString("username").take(320), optString("password").take(2048), optString("cache_generation", id).take(64))
        } }
    }.let { if (strict) it.getOrThrow() else it.getOrDefault(emptyList()) }

    private fun save(connections: List<OpdsConnection>) {
        val values = JSONArray().apply { connections.take(MAX_CONNECTIONS).forEach { connection -> put(JSONObject().apply {
            put("id", connection.id); put("name", connection.name); put("url", connection.catalogueUrl)
            put("username", connection.username); put("password", connection.password); put("cache_generation", connection.cacheGeneration)
        }) } }
        val bytes = values.toString().encodeToByteArray()
        require(bytes.size <= MAX_PAYLOAD_BYTES) { "Too many OPDS connections are configured." }
        requireSaved(encrypted.save(bytes))
    }

    companion object {
        const val MAX_CONNECTIONS = 50
        const val MAX_PAYLOAD_BYTES = 256 * 1024
    }
}

internal fun stableConnectionId(url: String, username: String): String = MessageDigest.getInstance("SHA-256")
    .digest("$url\u0000$username".encodeToByteArray()).take(12).joinToString("") { "%02x".format(it) }

internal fun updatedOpdsConnection(existing: OpdsConnection, name: String, catalogueUrl: String, username: String, password: String, all: List<OpdsConnection> = listOf(existing)): OpdsConnection {
    val normalized = validatedHttps(catalogueUrl.trim()).toString()
    require(name.trim().length in 1..120) { "Enter a catalogue name." }
    require(username.length <= 320 && password.length <= 2048) { "The OPDS credentials are too long." }
    require(all.none { it.id != existing.id && it.catalogueUrl == normalized && it.username == username }) { "This OPDS catalogue already exists." }
    val credentialsChanged = normalized != existing.catalogueUrl || username != existing.username || password != existing.password
    return OpdsConnection(existing.id, name.trim(), normalized, username, password, if (credentialsChanged) UUID.randomUUID().toString() else existing.cacheGeneration)
}
