package net.kdhmedia.mochil.auth

import android.util.Base64
import org.json.JSONObject
import java.net.HttpURLConnection
import java.net.URI
import net.kdhmedia.mochil.network.MAX_ERROR_RESPONSE_BYTES
import net.kdhmedia.mochil.network.readBoundedResponseText

class LoginClient {
    data class PairingRequest(val id: String, val code: String, val verificationUri: String)

    fun login(serverUrl: String, username: String, password: String, deviceName: String): DeviceCredential {
        val base = normalizedHttpsUrl(serverUrl)
        val connection = URI("$base/api/android/v1/login").toURL().openConnection() as HttpURLConnection
        return connection.run {
            requestMethod = "POST"
            instanceFollowRedirects = false
            connectTimeout = 10_000
            readTimeout = 10_000
            doOutput = true
            setRequestProperty("Accept", "application/json")
            setRequestProperty("Content-Type", "application/json")
            val basic = Base64.encodeToString("$username:$password".encodeToByteArray(), Base64.NO_WRAP)
            setRequestProperty("Authorization", "Basic $basic")
            outputStream.use { it.write(JSONObject().put("device_name", deviceName).toString().encodeToByteArray()) }
            val body = (if (responseCode in 200..299) inputStream else errorStream)?.use {
                it.readBoundedResponseText(if (responseCode in 200..299) 1 shl 20 else MAX_ERROR_RESPONSE_BYTES)
            }.orEmpty()
            if (responseCode !in 200..299) throw LoginException(responseCode, loginMessage(responseCode))
            val json = JSONObject(body)
            credentialFromServerResponse(base, json)
        }.also { connection.disconnect() }
    }

    fun disconnect(credential: DeviceCredential) {
        val connection = URI("${credential.serverUrl}/api/android/v1/device").toURL().openConnection() as HttpURLConnection
        try {
            connection.requestMethod = "DELETE"
            connection.connectTimeout = 10_000
            connection.readTimeout = 10_000
            connection.instanceFollowRedirects = false
            connection.setRequestProperty("Authorization", "Bearer ${credential.token}")
            if (connection.responseCode !in 200..299 && connection.responseCode != 401) throw LoginException(connection.responseCode, "The server could not disconnect this device.")
        } finally { connection.disconnect() }
    }

    fun changeServerAddress(credential: DeviceCredential, serverUrl: String): DeviceCredential {
        val base = normalizedHttpsUrl(serverUrl)
        val connection = URI("$base/api/android/v1/app").toURL().openConnection() as HttpURLConnection
        try {
            connection.requestMethod = "GET"
            connection.instanceFollowRedirects = false
            connection.connectTimeout = 10_000
            connection.readTimeout = 10_000
            connection.setRequestProperty("Authorization", "Bearer ${credential.token}")
            connection.setRequestProperty("Accept", "application/json")
            val status = connection.responseCode
            if (status == 401 || status == 403) throw LoginException(status, "This device token is not valid at the new address. Sign in there instead.")
            if (status !in 200..299) throw LoginException(status, "The new Mochil address could not validate this device.")
            if (!connection.contentType.orEmpty().substringBefore(';').trim().equals("application/json", true)) {
                throw LoginException(status, "The new address did not return the Mochil device API.")
            }
            connection.inputStream.close()
            return credential.copy(serverUrl = base)
        } finally { connection.disconnect() }
    }

    fun startPairing(serverUrl: String, deviceName: String): PairingRequest {
        val base = normalizedHttpsUrl(serverUrl)
        val connection = URI("$base/api/android/v1/pairing").toURL().openConnection() as HttpURLConnection
        return connection.run {
            requestMethod = "POST"; instanceFollowRedirects = false; connectTimeout = 10_000; readTimeout = 10_000; doOutput = true
            setRequestProperty("Accept", "application/json"); setRequestProperty("Content-Type", "application/json")
            outputStream.use { it.write(JSONObject().put("device_name", deviceName).toString().encodeToByteArray()) }
            val body = (if (responseCode in 200..299) inputStream else errorStream)?.use {
                it.readBoundedResponseText(if (responseCode in 200..299) 1 shl 20 else MAX_ERROR_RESPONSE_BYTES)
            }.orEmpty()
            if (responseCode !in 200..299) throw LoginException(responseCode, pairingMessage(responseCode))
            val json = JSONObject(body)
            PairingRequest(json.getString("pairing_id"), json.getString("user_code"), json.getString("verification_uri"))
        }.also { connection.disconnect() }
    }

    fun pollPairing(serverUrl: String, id: String): DeviceCredential? {
        val base = normalizedHttpsUrl(serverUrl)
        val connection = URI("$base/api/android/v1/pairing/$id").toURL().openConnection() as HttpURLConnection
        return connection.run {
            requestMethod = "GET"; instanceFollowRedirects = false; connectTimeout = 10_000; readTimeout = 10_000
            setRequestProperty("Accept", "application/json")
            val status = responseCode
            if (status == 202) return@run null
            val body = (if (status in 200..299) inputStream else errorStream)?.use {
                it.readBoundedResponseText(if (status in 200..299) 1 shl 20 else MAX_ERROR_RESPONSE_BYTES)
            }.orEmpty()
            if (status !in 200..299) throw LoginException(status, pairingMessage(status))
            val json = JSONObject(body)
            credentialFromServerResponse(base, json)
        }.also { connection.disconnect() }
    }

    private fun normalizedHttpsUrl(value: String): String = ServerAddress.normalize(value)

    private fun loginMessage(status: Int) = when (status) {
        401 -> "The username or password is not correct."
        429 -> "Too many sign-in attempts. Wait a few minutes and try again."
        in 500..599 -> "Mochil is temporarily unavailable. Try again."
        else -> "Could not connect to this Mochil server."
    }

    private fun pairingMessage(status: Int) = when (status) {
        410 -> "This pairing code expired. Start again."
        429 -> "Too many pairing requests. Wait a moment and try again."
        in 500..599 -> "Mochil is temporarily unavailable. Try again."
        else -> "Could not pair with this Mochil server."
    }
}

class LoginException(val status: Int, override val message: String) : Exception(message)

internal fun credentialFromServerResponse(approvedServerUrl: String, json: JSONObject): DeviceCredential {
    val deviceId = json.optString("device_id")
    val token = json.optString("device_token")
    if (deviceId.isBlank() || deviceId.length > 256 || token.isBlank() || token.length > 8192) {
        throw LoginException(0, "The Mochil server returned an invalid device credential.")
    }
    return DeviceCredential(approvedServerUrl, deviceId, token)
}
