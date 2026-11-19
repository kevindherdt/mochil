package net.kdhmedia.mochil.auth

import android.content.Context

data class DeviceCredential(val serverUrl: String, val deviceId: String, val token: String)

class DeviceCredentialStore(context: Context) {
    private val encrypted = EncryptedPayloadStore(context, "device_credentials", KEY_ALIAS)

    fun save(credential: DeviceCredential) {
        val plaintext = listOf(credential.serverUrl, credential.deviceId, credential.token).joinToString("\u0000").encodeToByteArray()
        encrypted.save(plaintext)
    }

    fun load(): DeviceCredential? = runCatching {
        val parts = encrypted.load()?.decodeToString()?.split('\u0000') ?: return@runCatching null
        if (parts.size != 3) null else DeviceCredential(parts[0], parts[1], parts[2])
    }.getOrNull()

    fun clear() = encrypted.clear()

    private companion object {
        const val KEY_ALIAS = "mochil_device_token"
    }
}
