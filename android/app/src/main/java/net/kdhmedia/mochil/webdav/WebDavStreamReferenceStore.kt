package net.kdhmedia.mochil.webdav

import android.content.Context
import android.net.Uri
import java.security.MessageDigest
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonArray
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.put
import net.kdhmedia.mochil.auth.EncryptedPayloadStore

internal data class WebDavStreamReference(val token: String, val connectionId: String, val credentialKey: String, val href: String, val usedAt: Long)

internal class WebDavStreamReferenceStore(context: Context) {
    private val encrypted = EncryptedPayloadStore(context, "webdav_stream_references", "mochil_webdav_stream_references")

    @Synchronized fun put(connection: WebDavConnection, href: String): Uri {
        val identity = webDavCredentialKey(connection)
        val token = webDavStreamToken(identity, href)
        val values = (decodeWebDavStreamReferences(encrypted.load()).filterNot { it.token == token } + WebDavStreamReference(token, connection.id, identity, href, System.currentTimeMillis()))
            .sortedByDescending(WebDavStreamReference::usedAt).take(MAX_STREAM_REFERENCES)
        encrypted.save(encodeWebDavStreamReferences(values))
        return webDavStreamUri(token)
    }

    @Synchronized fun resolve(token: String): WebDavStreamReference? = decodeWebDavStreamReferences(encrypted.load()).firstOrNull { it.token == token }
    @Synchronized fun removeConnection(connectionId: String) {
        val values = decodeWebDavStreamReferences(encrypted.load()).filterNot { it.connectionId == connectionId }
        if (values.isEmpty()) encrypted.clear() else encrypted.save(encodeWebDavStreamReferences(values))
    }
}

internal fun webDavStreamUri(token: String): Uri = Uri.Builder().scheme(WEB_DAV_STREAM_SCHEME).authority("stream").appendPath(token).build()
internal fun webDavStreamToken(credentialKey: String, href: String): String = MessageDigest.getInstance("SHA-256")
    .digest("$credentialKey\u0000$href".encodeToByteArray()).take(18).joinToString("") { "%02x".format(it) }

internal fun encodeWebDavStreamReferences(values: List<WebDavStreamReference>): ByteArray = buildJsonArray {
    values.take(MAX_STREAM_REFERENCES).forEach { value -> add(buildJsonObject {
        put("token", value.token); put("connection", value.connectionId); put("credential", value.credentialKey); put("href", value.href); put("used", value.usedAt)
    }) }
}.toString().encodeToByteArray()

internal fun decodeWebDavStreamReferences(payload: ByteArray?): List<WebDavStreamReference> = runCatching {
    if (payload == null || payload.size > MAX_STREAM_REFERENCE_BYTES) return emptyList()
    val root = Json.parseToJsonElement(payload.decodeToString()) as? JsonArray ?: return emptyList()
    root.take(MAX_STREAM_REFERENCES).mapNotNull { (it as? JsonObject)?.run {
        val token = string("token"); val connection = string("connection"); val credential = string("credential"); val href = string("href")
        if (!token.matches(Regex("^[a-f0-9]{36}$")) || connection.isBlank() || href.isBlank()) return@mapNotNull null
        WebDavStreamReference(token, connection.take(64), credential.take(64), href.take(4_096), long("used"))
    } }
}.getOrDefault(emptyList())

private fun JsonObject.string(name: String) = (this[name] as? JsonPrimitive)?.contentOrNull.orEmpty()
private fun JsonObject.long(name: String) = string(name).toLongOrNull()?.coerceAtLeast(0) ?: 0
internal const val WEB_DAV_STREAM_SCHEME = "mochil-webdav"
internal const val MAX_STREAM_REFERENCES = 200
internal const val MAX_STREAM_REFERENCE_BYTES = 1024 * 1024
