package net.kdhmedia.mochil.hermes

import java.io.BufferedReader
import java.io.IOException
import java.io.InputStreamReader
import java.net.HttpURLConnection
import java.net.URI
import java.net.URLEncoder
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.currentCoroutineContext
import kotlinx.coroutines.Job
import kotlinx.coroutines.ensureActive
import kotlinx.coroutines.withContext
import net.kdhmedia.mochil.auth.DeviceCredential
import net.kdhmedia.mochil.network.MAX_ERROR_RESPONSE_BYTES
import net.kdhmedia.mochil.network.readBoundedResponseText
import org.json.JSONArray
import org.json.JSONObject

/** The server refused or failed a chat request; the message is safe to show. */
class HermesException(message: String, val status: Int = 0) : IOException(message)

/**
 * Talks to the chat under /api/android/v1/hermes/. Only an administrator's paired device is allowed in; for anyone else
 * [status] returns null and the app simply does not offer the chat. The Hermes key never reaches the phone.
 */
class HermesClient {
    /** Null when the chat is not available to this device (module off, or not an administrator). */
    suspend fun status(credential: DeviceCredential): HermesStatus? = withContext(Dispatchers.IO) {
        val connection = open(credential, "GET", "status")
        try {
            when (connection.responseCode) {
                403, 404 -> null
                in 200..299 -> parseHermesStatus(JSONObject(connection.inputStream.use { it.readBoundedResponseText() }))
                else -> throw failure(connection)
            }
        } finally { connection.disconnect() }
    }

    suspend fun chats(credential: DeviceCredential): List<HermesChat> = withContext(Dispatchers.IO) {
        parseHermesChats(getJson(credential, "chats"))
    }

    suspend fun messages(credential: DeviceCredential, chatId: String): List<HermesMessage> = withContext(Dispatchers.IO) {
        parseHermesMessages(getJson(credential, "chats/${segment(chatId)}"))
    }

    suspend fun rename(credential: DeviceCredential, chatId: String, title: String) = withContext(Dispatchers.IO) {
        send(credential, "PATCH", "chats/${segment(chatId)}", JSONObject().put("title", title))
    }

    suspend fun delete(credential: DeviceCredential, chatId: String) = withContext(Dispatchers.IO) {
        send(credential, "DELETE", "chats/${segment(chatId)}", null)
    }

    /** What she remembers, or null when memory is not set up on the server. */
    suspend fun memories(credential: DeviceCredential): List<HermesMemory>? = withContext(Dispatchers.IO) {
        val root = getJson(credential, "memories")
        if (!root.optBoolean("enabled")) null else parseHermesMemories(root)
    }

    suspend fun remember(credential: DeviceCredential, text: String) = withContext(Dispatchers.IO) {
        send(credential, "POST", "remember", JSONObject().put("text", text))
    }

    /**
     * Sends one message and calls [onUpdate] for each piece of the answer as it arrives. Returns when the answer is
     * complete. Cancelling the coroutine closes the connection; the server still lets Hermes finish and save the reply.
     */
    suspend fun chat(
        credential: DeviceCredential,
        chatId: String,
        message: String,
        spoken: Boolean,
        interrupted: Boolean,
        onUpdate: (HermesStreamUpdate) -> Unit,
    ) = withContext(Dispatchers.IO) {
        val body = JSONObject().put("session", chatId).put("message", message).put("spoken", spoken).put("interrupted", interrupted)
        val connection = open(credential, "POST", "chat", readTimeoutMillis = 90_000, accept = "text/event-stream")
        try {
            connection.doOutput = true
            connection.outputStream.use { it.write(body.toString().encodeToByteArray()) }
            if (connection.responseCode !in 200..299) throw failure(connection)
            val reader = HermesStreamReader()
            BufferedReader(InputStreamReader(connection.inputStream, Charsets.UTF_8)).use { lines ->
                while (true) {
                    currentCoroutineContext().ensureActive()
                    val line = lines.readLine() ?: break
                    reader.feed(line)?.let(onUpdate)
                }
            }
        } finally { connection.disconnect() }
    }

    /** The voices on offer and whether the server can also hear you; null when speech is not set up there. */
    suspend fun voiceStatus(credential: DeviceCredential): HermesVoiceStatus? = withContext(Dispatchers.IO) {
        val connection = openRaw(credential, "GET", "narration/status")
        try {
            when (connection.responseCode) {
                403, 404, 503 -> null
                in 200..299 -> parseHermesVoiceStatus(JSONObject(connection.inputStream.use { it.readBoundedResponseText() }))
                else -> throw failure(connection)
            }
        } finally { connection.disconnect() }
    }

    /** A list from the library, for quick-action context and the book picker. `path` is relative to /api/android/v1/. */
    suspend fun library(credential: DeviceCredential, path: String): List<JSONObject> = withContext(Dispatchers.IO) {
        val connection = openRaw(credential, "GET", path)
        try {
            if (connection.responseCode !in 200..299) throw failure(connection)
            jsonObjects(JSONArray(connection.inputStream.use { it.readBoundedResponseText() }))
        } finally { connection.disconnect() }
    }

    /** The avatar the server has chosen; null when the avatar module is off, or there is no model or no Live2D core. */
    suspend fun avatar(credential: DeviceCredential): HermesAvatarInfo? = withContext(Dispatchers.IO) {
        val connection = openRaw(credential, "GET", "avatar/models")
        try {
            when (connection.responseCode) {
                403, 404, 503 -> null
                in 200..299 -> parseHermesAvatarInfo(JSONObject(connection.inputStream.use { it.readBoundedResponseText() }))
                else -> throw failure(connection)
            }
        } finally { connection.disconnect() }
    }

    /** Makes this model the one the chat shows, with the voice it speaks with. */
    suspend fun selectAvatar(credential: DeviceCredential, model: String, voice: String) = withContext(Dispatchers.IO) {
        val connection = openRaw(credential, "PUT", "avatar/select")
        try {
            connection.doOutput = true
            connection.outputStream.use { it.write(JSONObject().put("model", model).put("voice", voice).toString().encodeToByteArray()) }
            if (connection.responseCode !in 200..299) throw failure(connection)
        } finally { connection.disconnect() }
    }

    /** Saves the background and/or where one model sits; the website shows the same choice. */
    suspend fun saveAvatarLook(credential: DeviceCredential, body: JSONObject) = withContext(Dispatchers.IO) {
        val connection = openRaw(credential, "PUT", "avatar/look")
        try {
            connection.doOutput = true
            connection.outputStream.use { it.write(body.toString().encodeToByteArray()) }
            if (connection.responseCode !in 200..299) throw failure(connection)
        } finally { connection.disconnect() }
    }

    /** Loads the speech model ahead of the first sentence. Never fails the caller. */
    suspend fun warm(credential: DeviceCredential, voice: String?) = withContext(Dispatchers.IO) {
        runCatching {
            val connection = openRaw(credential, "POST", "narration/warm")
            try {
                connection.doOutput = true
                connection.outputStream.use { it.write(JSONObject().put("voice", voice ?: "").toString().encodeToByteArray()) }
                connection.responseCode
            } finally { connection.disconnect() }
        }
        Unit
    }

    /** Turns one recording (a 16 kHz mono WAV) into text. Empty when nothing was said. */
    suspend fun transcribe(credential: DeviceCredential, wav: ByteArray): String = withContext(Dispatchers.IO) {
        val connection = openRaw(credential, "POST", "narration/transcribe", readTimeoutMillis = 90_000, contentType = "audio/wav")
        try {
            connection.doOutput = true
            connection.setFixedLengthStreamingMode(wav.size)
            connection.outputStream.use { it.write(wav) }
            if (connection.responseCode !in 200..299) throw failure(connection)
            JSONObject(connection.inputStream.use { it.readBoundedResponseText() }).optString("text").trim()
        } finally { connection.disconnect() }
    }

    /**
     * Speaks one sentence: calls [onRate] once with the sample rate, then [onChunk] with whole 16-bit mono samples as the
     * server makes them. Cancelling the coroutine closes the connection.
     */
    suspend fun speak(credential: DeviceCredential, text: String, voice: String?, onRate: (Int) -> Unit, onChunk: (ByteArray) -> Unit) = withContext(Dispatchers.IO) {
        val connection = openRaw(credential, "POST", "narration/stream", readTimeoutMillis = 60_000)
        val closer = currentCoroutineContext()[Job]?.invokeOnCompletion { connection.disconnect() }
        try {
            connection.doOutput = true
            val body = JSONObject().put("text", text).apply { if (!voice.isNullOrBlank()) put("voice", voice) }
            connection.outputStream.use { it.write(body.toString().encodeToByteArray()) }
            if (connection.responseCode !in 200..299) throw failure(connection)
            onRate(connection.getHeaderField("X-Sample-Rate")?.toIntOrNull() ?: 24000)
            val whole = SampleSplitter()
            val buffer = ByteArray(4096)
            connection.inputStream.use { input ->
                while (true) {
                    currentCoroutineContext().ensureActive()
                    val count = input.read(buffer)
                    if (count < 0) break
                    val bytes = whole.feed(buffer, count)
                    if (bytes.isNotEmpty()) onChunk(bytes)
                }
            }
        } finally { closer?.dispose(); connection.disconnect() }
    }

    private fun getJson(credential: DeviceCredential, path: String): JSONObject {
        val connection = open(credential, "GET", path)
        return try {
            if (connection.responseCode !in 200..299) throw failure(connection)
            JSONObject(connection.inputStream.use { it.readBoundedResponseText() })
        } finally { connection.disconnect() }
    }

    private fun send(credential: DeviceCredential, method: String, path: String, body: JSONObject?) {
        val connection = open(credential, method, path)
        try {
            if (body != null) {
                connection.doOutput = true
                connection.outputStream.use { it.write(body.toString().encodeToByteArray()) }
            }
            if (connection.responseCode !in 200..299) throw failure(connection)
        } finally { connection.disconnect() }
    }

    private fun open(
        credential: DeviceCredential,
        method: String,
        path: String,
        readTimeoutMillis: Int = 20_000,
        accept: String = "application/json",
    ): HttpURLConnection = (URI(credential.serverUrl.trimEnd('/') + "/api/android/v1/hermes/" + path).toURL().openConnection() as HttpURLConnection).apply {
        requestMethod = method
        connectTimeout = 10_000
        readTimeout = readTimeoutMillis
        instanceFollowRedirects = false
        setRequestProperty("Authorization", "Bearer ${credential.token}")
        setRequestProperty("Accept", accept)
        if (method != "GET" && method != "DELETE") setRequestProperty("Content-Type", "application/json")
    }

    /** Like [open] but for the narration endpoints, which live beside the chat under /api/android/v1/. */
    private fun openRaw(
        credential: DeviceCredential,
        method: String,
        path: String,
        readTimeoutMillis: Int = 20_000,
        contentType: String = "application/json",
    ): HttpURLConnection = (URI(credential.serverUrl.trimEnd('/') + "/api/android/v1/" + path).toURL().openConnection() as HttpURLConnection).apply {
        requestMethod = method
        connectTimeout = 10_000
        readTimeout = readTimeoutMillis
        instanceFollowRedirects = false
        setRequestProperty("Authorization", "Bearer ${credential.token}")
        setRequestProperty("Accept", "*/*")
        if (method != "GET") setRequestProperty("Content-Type", contentType)
    }

    private fun failure(connection: HttpURLConnection): HermesException {
        val status = connection.responseCode
        val detail = runCatching {
            JSONObject(connection.errorStream?.use { it.readBoundedResponseText(MAX_ERROR_RESPONSE_BYTES) }.orEmpty()).optString("error")
        }.getOrDefault("")
        return HermesException(detail.ifBlank { "The server answered $status" }, status)
    }

    private fun segment(value: String) = URLEncoder.encode(value, "UTF-8").replace("+", "%20")
}
