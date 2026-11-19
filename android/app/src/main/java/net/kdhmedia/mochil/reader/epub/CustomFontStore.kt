package net.kdhmedia.mochil.reader.epub

import android.app.Application
import android.content.Context
import android.net.Uri
import android.provider.OpenableColumns
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import java.io.File
import java.io.ByteArrayOutputStream
import java.io.InputStream
import java.security.MessageDigest
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import org.json.JSONArray
import org.json.JSONObject

data class CustomReaderFont(val id: String, val name: String, val extension: String)
data class CustomFontState(val fonts: List<CustomReaderFont> = emptyList(), val importing: Boolean = false, val message: String? = null)

class CustomFontViewModel(application: Application) : AndroidViewModel(application) {
    private val store = CustomFontStore(application)
    private val mutableState = MutableStateFlow(CustomFontState(store.list()))
    val state = mutableState.asStateFlow()

    fun import(uris: List<Uri>) {
        if (uris.isEmpty() || mutableState.value.importing) return
        mutableState.value = mutableState.value.copy(importing = true, message = null)
        viewModelScope.launch {
            val results = withContext(Dispatchers.IO) { uris.take(CustomFontStore.MAX_IMPORT_COUNT).map { runCatching { store.import(it) } } }
            val imported = results.count { it.isSuccess }
            val failed = results.size - imported
            mutableState.value = CustomFontState(
                fonts = store.list(),
                message = when {
                    imported > 0 && failed == 0 -> "Imported $imported custom ${if (imported == 1) "font" else "fonts"}."
                    imported > 0 -> "Imported $imported fonts; $failed could not be read."
                    else -> "No valid TTF or OTF fonts were imported."
                },
            )
        }
    }

    fun remove(font: CustomReaderFont) {
        store.remove(font)
        mutableState.value = mutableState.value.copy(fonts = store.list(), message = "Removed ${font.name}.")
    }
}

class CustomFontStore(private val context: Context) {
    private val directory = File(context.filesDir, "reader-fonts").apply { mkdirs() }
    private val preferences = context.getSharedPreferences("custom_reader_fonts", Context.MODE_PRIVATE)

    fun list(): List<CustomReaderFont> = runCatching {
        val values = JSONArray(preferences.getString("fonts", "[]"))
        List(values.length()) { index -> values.getJSONObject(index).run { CustomReaderFont(getString("id"), getString("name"), getString("extension")) } }
            .filter { file(it).isFile }.sortedBy { it.name.lowercase() }
    }.getOrDefault(emptyList())

    fun find(id: String?): CustomReaderFont? = id?.let { value -> list().firstOrNull { it.id == value } }

    fun bytes(font: CustomReaderFont): ByteArray? = runCatching { file(font).takeIf { it.length() in 1..MAX_FONT_BYTES }?.readBytes() }.getOrNull()

    fun import(uri: Uri): CustomReaderFont {
        val displayName = displayName(uri)
        val extension = displayName.substringAfterLast('.', "").lowercase().takeIf { it == "ttf" || it == "otf" }
            ?: error("Unsupported font type")
        val bytes = context.contentResolver.openInputStream(uri)?.use { input -> input.readBounded(MAX_FONT_BYTES.toInt() + 1) } ?: error("Font unavailable")
        require(bytes.size in 4..MAX_FONT_BYTES.toInt() && isValidReaderFont(bytes, extension)) { "Invalid font" }
        val id = MessageDigest.getInstance("SHA-256").digest(bytes).take(12).joinToString("") { "%02x".format(it) }
        val font = CustomReaderFont(id, displayName.substringBeforeLast('.').trim().take(80).ifBlank { "Custom font" }, extension)
        val target = file(font)
        if (!target.exists()) target.outputStream().use { it.write(bytes) }
        save((list().filterNot { it.id == id } + font).sortedBy { it.name.lowercase() })
        return font
    }

    fun remove(font: CustomReaderFont) {
        file(font).delete()
        save(list().filterNot { it.id == font.id })
    }

    private fun save(fonts: List<CustomReaderFont>) {
        val json = JSONArray().apply { fonts.forEach { put(JSONObject().put("id", it.id).put("name", it.name).put("extension", it.extension)) } }
        preferences.edit().putString("fonts", json.toString()).apply()
    }

    private fun file(font: CustomReaderFont) = File(directory, "${font.id}.${font.extension}")

    private fun displayName(uri: Uri): String = context.contentResolver.query(uri, arrayOf(OpenableColumns.DISPLAY_NAME), null, null, null)?.use { cursor ->
        if (cursor.moveToFirst()) cursor.getString(0) else null
    } ?: uri.lastPathSegment.orEmpty().substringAfterLast('/').ifBlank { "font.ttf" }


    fun typeface(font: CustomReaderFont): android.graphics.Typeface? = runCatching { android.graphics.Typeface.createFromFile(file(font)) }.getOrNull()

    companion object {
        const val MAX_FONT_BYTES = 6L * 1024 * 1024
        const val MAX_IMPORT_COUNT = 12
    }
}

private fun InputStream.readBounded(maximumBytes: Int): ByteArray {
    val output = ByteArrayOutputStream(minOf(maximumBytes, 64 * 1024))
    val buffer = ByteArray(16 * 1024)
    while (output.size() < maximumBytes) {
        val count = read(buffer, 0, minOf(buffer.size, maximumBytes - output.size()))
        if (count < 0) break
        output.write(buffer, 0, count)
    }
    return output.toByteArray()
}

internal fun isValidReaderFont(bytes: ByteArray, extension: String): Boolean {
    if (bytes.size < 4) return false
    return when (extension.lowercase()) {
        "otf" -> bytes.copyOfRange(0, 4).decodeToString() == "OTTO"
        "ttf" -> bytes[0] == 0.toByte() && bytes[1] == 1.toByte() && bytes[2] == 0.toByte() && bytes[3] == 0.toByte()
        else -> false
    }
}
