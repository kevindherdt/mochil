package net.kdhmedia.mochil.reader.epub

import java.util.Base64
import org.json.JSONObject
import org.readium.r2.navigator.epub.EpubNavigatorFragment

/** Owns validation and document injection for EPUB custom fonts. */
class EpubFontRuntime(private val store: CustomFontStore) {
    fun availableFonts(): List<CustomReaderFont> = store.list()

    fun validated(preferences: ReaderPreferences): ReaderPreferences =
        if (preferences.font == ReaderFont.Custom && store.find(preferences.customFontId) == null) {
            preferences.copy(font = ReaderFont.Publisher, customFontId = null)
        } else {
            preferences
        }

    fun family(preferences: ReaderPreferences): String? =
        store.find(preferences.customFontId)?.let { customFontFamily(it.id) }

    suspend fun apply(navigator: EpubNavigatorFragment?, preferences: ReaderPreferences) {
        if (preferences.font != ReaderFont.Custom) return
        val font = store.find(preferences.customFontId) ?: return
        val bytes = store.bytes(font) ?: return
        runCatching { navigator?.evaluateJavascript(customFontInjectionScript(font, bytes)) }
    }
}

internal fun customFontFamily(id: String): String = "MochilCustom_$id"

internal fun customFontInjectionScript(font: CustomReaderFont, bytes: ByteArray): String {
    val family = customFontFamily(font.id)
    val isOpenType = font.extension.equals("otf", ignoreCase = true)
    val mime = if (isOpenType) "font/otf" else "font/ttf"
    val format = if (isOpenType) "opentype" else "truetype"
    val encoded = Base64.getEncoder().encodeToString(bytes)
    val css = "@font-face{font-family:'$family';src:url(data:$mime;base64,$encoded) format('$format');font-style:normal;font-weight:100 900;font-display:swap;}"
    return """(() => { const id='mochil-custom-font'; let s=document.getElementById(id); if(!s){s=document.createElement('style');s.id=id;document.head.appendChild(s);} s.textContent=${JSONObject.quote(css)}; })();"""
}
