package net.kdhmedia.mochil.reader.epub

import org.json.JSONTokener
import org.json.JSONObject

/**
 * The structural subset of CREngine XPointers emitted by KOReader for EPUBs.
 *
 * CREngine numbers DocFragments from one and element siblings by tag name from
 * one. Readium can consume the equivalent CSS selector in a Locator. Text-node
 * offsets are retained for fixture validation and a future range adapter, while
 * the shared text anchor supplies exact character restoration today.
 */
internal data class KoReaderXPointer(
    val spineIndex: Int,
    val cssSelector: String,
    val textNodeIndex: Int? = null,
    val textOffset: Int? = null,
) {
    companion object {
        private const val MAX_XPOINTER_LENGTH = 8_192
        private const val MAX_PATH_STEPS = 128
        private val root = Regex("^/body/DocFragment\\[(\\d+)](?:\\.\\d+)?(?:/body(?:/(.*))?)?$")
        private val element = Regex("^([A-Za-z][A-Za-z0-9_-]*)(?:\\[(\\d+)])?$")
        private val text = Regex("^text\\(\\)(?:\\[(\\d+)])?(?:\\.(\\d+))?$")

        fun parse(value: String?): KoReaderXPointer? {
            if (value.isNullOrBlank() || value.length > MAX_XPOINTER_LENGTH) return null
            val match = root.matchEntire(value) ?: return null
            val fragment = match.groupValues[1].toIntOrNull()?.takeIf { it > 0 } ?: return null
            val rawSteps = match.groupValues[2].split('/').filter(String::isNotEmpty)
            if (rawSteps.size > MAX_PATH_STEPS) return null

            val selector = mutableListOf("body")
            var textIndex: Int? = null
            var offset: Int? = null
            rawSteps.forEachIndexed { index, step ->
                text.matchEntire(step)?.let { textMatch ->
                    if (index != rawSteps.lastIndex || textIndex != null) return null
                    textIndex = textMatch.groupValues[1].ifBlank { "1" }.toIntOrNull()?.takeIf { it > 0 } ?: return null
                    offset = textMatch.groupValues[2].ifBlank { "0" }.toIntOrNull()?.takeIf { it >= 0 } ?: return null
                    return@forEachIndexed
                }
                val elementMatch = element.matchEntire(step) ?: return null
                val sibling = elementMatch.groupValues[2].ifBlank { "1" }.toIntOrNull()?.takeIf { it > 0 } ?: return null
                selector += "${elementMatch.groupValues[1].lowercase()}:nth-of-type($sibling)"
            }
            return KoReaderXPointer(fragment - 1, selector.joinToString(" > "), textIndex, offset)
        }

        fun captureScript(spineIndex: Int, cssSelector: String): String? {
            if (spineIndex < 0 || cssSelector.isBlank() || cssSelector.length > 4_096) return null
            return """(() => {
                const selected = document.querySelector(${JSONObject.quote(cssSelector)});
                if (!selected || !document.body.contains(selected)) return null;
                const steps = [];
                let node = selected;
                while (node && node !== document.body) {
                    const name = node.localName;
                    if (!name || !/^[A-Za-z][A-Za-z0-9_-]*$/.test(name)) return null;
                    let sibling = 1;
                    for (let before = node.previousElementSibling; before; before = before.previousElementSibling) {
                        if (before.localName === name) sibling++;
                    }
                    steps.unshift('/' + name + '[' + sibling + ']');
                    node = node.parentElement;
                }
                if (node !== document.body) return null;
                return '/body/DocFragment[${spineIndex + 1}]/body' + steps.join('');
            })()""".trimIndent()
        }

        fun decodeJavascriptResult(value: String?): String? {
            if (value.isNullOrBlank() || value == "null") return null
            val decoded = runCatching { JSONTokener(value).nextValue() as? String }.getOrNull() ?: value.takeIf { it.startsWith('/') }
            return decoded?.takeIf { parse(it) != null }
        }
    }
}
