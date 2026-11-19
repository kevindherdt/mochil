package net.kdhmedia.mochil.auth

/** Missing storage is empty; a present but unreadable envelope must not be overwritten. */
internal fun requireReadableEncryptedPayload(encoded: String?, encodedIv: String?, read: () -> ByteArray?): ByteArray? {
    if (encoded == null && encodedIv == null) return null
    return checkNotNull(read()) { "Encrypted storage could not be read" }
}

/** Bounds allocations before decoding/decrypting our AES-GCM preference envelope. */
internal fun decodeBoundedEncryptedPayload(
    encoded: String?,
    encodedIv: String?,
    maxPlaintextBytes: Int,
    decode: (String) -> ByteArray,
    decrypt: (ByteArray, ByteArray) -> ByteArray,
): ByteArray? = runCatching {
    require(maxPlaintextBytes >= 0)
    if (encoded == null || encodedIv == null) return null
    val ciphertextLimit = maxPlaintextBytes.toLong() + 16 // 128-bit GCM authentication tag.
    val encodedLimit = ((ciphertextLimit + 2) / 3) * 4
    if (encoded.length > encodedLimit || encodedIv.length > 24) return null
    val ciphertext = decode(encoded)
    if (ciphertext.size < 16 || ciphertext.size > ciphertextLimit) return null
    val iv = decode(encodedIv)
    if (iv.size != 12) return null // AndroidKeyStore's generated 96-bit GCM IV.
    decrypt(ciphertext, iv).takeIf { it.size <= maxPlaintextBytes }
}.getOrNull()
