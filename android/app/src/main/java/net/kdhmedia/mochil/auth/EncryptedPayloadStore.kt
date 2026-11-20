package net.kdhmedia.mochil.auth

import android.content.Context
import android.security.keystore.KeyGenParameterSpec
import android.security.keystore.KeyProperties
import android.util.Base64
import java.security.KeyStore
import javax.crypto.Cipher
import javax.crypto.KeyGenerator
import javax.crypto.SecretKey
import javax.crypto.spec.GCMParameterSpec

internal class EncryptedPayloadStore(
    context: Context,
    preferencesName: String,
    private val keyAlias: String,
    private val maxPlaintextBytes: Int = Int.MAX_VALUE,
) {
    init { require(maxPlaintextBytes >= 0) }
    private val preferences = context.getSharedPreferences(preferencesName, Context.MODE_PRIVATE)
    // Opening the keystore used to happen here, in the field initialiser, so any store that was
    // constructed took the risk: a locked, corrupted or missing keystore threw out of the constructor
    // and whatever was building the object -- a view model created during composition, for example --
    // took the whole app down. Nothing is read until a payload is actually loaded or saved now, and
    // every caller of those already handles failure.
    private val keyStore by lazy { KeyStore.getInstance("AndroidKeyStore").apply { load(null) } }

    fun save(plaintext: ByteArray): Boolean {
        if (plaintext.size > maxPlaintextBytes) return false
        val cipher = Cipher.getInstance(TRANSFORMATION).apply { init(Cipher.ENCRYPT_MODE, secretKey()) }
        return preferences.edit()
            .putString("payload", Base64.encodeToString(cipher.doFinal(plaintext), Base64.NO_WRAP))
            .putString("iv", Base64.encodeToString(cipher.iv, Base64.NO_WRAP))
            .commit()
    }

    fun load(): ByteArray? = runCatching {
        decodeBoundedEncryptedPayload(
            preferences.getString("payload", null), preferences.getString("iv", null), maxPlaintextBytes,
            decode = { Base64.decode(it, Base64.NO_WRAP) },
            decrypt = { payload, iv -> Cipher.getInstance(TRANSFORMATION).run {
                init(Cipher.DECRYPT_MODE, secretKey(), GCMParameterSpec(128, iv))
                doFinal(payload)
            } },
        )
    }.getOrNull()

    // Opt-in strict reads for read/modify/write stores. Existing best-effort
    // consumers retain load(), while journals can distinguish absence/failure.
    fun loadStrict(): ByteArray? = requireReadableEncryptedPayload(
        preferences.getString("payload", null), preferences.getString("iv", null), ::load,
    )

    fun clear() = preferences.edit().clear().commit()

    private fun secretKey(): SecretKey = getOrCreatePersistentKey(
        load = { keyStore.getKey(keyAlias, null) as? SecretKey },
        create = { KeyGenerator.getInstance(KeyProperties.KEY_ALGORITHM_AES, "AndroidKeyStore").run {
            init(
                KeyGenParameterSpec.Builder(keyAlias, KeyProperties.PURPOSE_ENCRYPT or KeyProperties.PURPOSE_DECRYPT)
                    .setBlockModes(KeyProperties.BLOCK_MODE_GCM)
                    .setEncryptionPaddings(KeyProperties.ENCRYPTION_PADDING_NONE)
                    .build(),
            )
            generateKey()
        } },
    )

    private companion object { const val TRANSFORMATION = "AES/GCM/NoPadding" }
}
