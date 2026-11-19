package net.kdhmedia.mochil.auth

import java.util.Base64
import javax.crypto.Cipher
import javax.crypto.spec.GCMParameterSpec
import javax.crypto.spec.SecretKeySpec
import org.junit.Assert.assertArrayEquals
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class EncryptedPayloadBoundsTest {
    private val key = SecretKeySpec(ByteArray(16) { it.toByte() }, "AES")
    private val iv = ByteArray(12) { (it + 1).toByte() }
    private fun encode(bytes: ByteArray) = Base64.getEncoder().encodeToString(bytes)
    private fun encrypted(bytes: ByteArray) = Cipher.getInstance("AES/GCM/NoPadding").run {
        init(Cipher.ENCRYPT_MODE, key, GCMParameterSpec(128, this@EncryptedPayloadBoundsTest.iv))
        doFinal(bytes)
    }
    private fun decrypt(bytes: ByteArray, nonce: ByteArray) = Cipher.getInstance("AES/GCM/NoPadding").run {
        init(Cipher.DECRYPT_MODE, key, GCMParameterSpec(128, nonce))
        doFinal(bytes)
    }
    private fun load(bytes: ByteArray, limit: Int) = decodeBoundedEncryptedPayload(
        encode(bytes), encode(iv), limit, { Base64.getDecoder().decode(it) }, ::decrypt,
    )

    @Test fun `authenticated payloads at the exact byte limit and empty payloads round trip`() {
        for (size in listOf(0, 1, 2, 3, 1_024)) {
            val plaintext = ByteArray(size) { it.toByte() }
            assertArrayEquals(plaintext, load(encrypted(plaintext), size))
        }
    }

    @Test fun `oversized encoded envelope is rejected before decoding or decrypting`() {
        var calls = 0
        assertNull(decodeBoundedEncryptedPayload("x".repeat(200), encode(iv), 1,
            { calls++; ByteArray(16) }, { _, _ -> calls++; byteArrayOf() }))
        assertEquals(0, calls)
    }

    @Test fun `base64 padding slack cannot admit oversized ciphertext`() {
        var decryptions = 0
        // 17 and 18 bytes both encode to 24 characters; decoded size must also be checked.
        assertNull(decodeBoundedEncryptedPayload(encode(ByteArray(18)), encode(iv), 1,
            { Base64.getDecoder().decode(it) }, { _, _ -> decryptions++; byteArrayOf() }))
        assertEquals(0, decryptions)
    }

    @Test fun `bad tag malformed encoding and invalid IV fail closed`() {
        val bytes = encrypted(byteArrayOf(1, 2, 3))
        bytes[0] = (bytes[0].toInt() xor 1).toByte()
        assertNull(load(bytes, 3))
        assertNull(decodeBoundedEncryptedPayload("???", encode(iv), 3,
            { Base64.getDecoder().decode(it) }, ::decrypt))
        assertNull(decodeBoundedEncryptedPayload(encode(encrypted(byteArrayOf())), encode(ByteArray(16)), 3,
            { Base64.getDecoder().decode(it) }, ::decrypt))
    }

    @Test fun `default capacity arithmetic does not overflow`() {
        assertArrayEquals(byteArrayOf(1), load(encrypted(byteArrayOf(1)), Int.MAX_VALUE))
    }
}
