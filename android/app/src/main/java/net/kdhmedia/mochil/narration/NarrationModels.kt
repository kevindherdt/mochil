package net.kdhmedia.mochil.narration

import org.json.JSONObject

data class NarrationVoice(val id: String, val name: String, val language: String, val custom: Boolean)

data class NarrationStatus(val voices: List<NarrationVoice>, val defaultVoice: String)

data class NarrationChapter(val index: Int, val href: String, val title: String, val matter: Boolean)

data class NarrationSentence(val text: String, val progression: Double, val before: String, val after: String)

internal fun parseNarrationStatus(root: JSONObject): NarrationStatus? {
    if (!root.optBoolean("available")) return null
    val voices = root.optJSONArray("voices") ?: return null
    return NarrationStatus(
        voices = (0 until voices.length()).mapNotNull { i ->
            voices.optJSONObject(i)?.let { v ->
                val id = v.optString("id")
                if (id.isEmpty()) null else NarrationVoice(id, v.optString("name", id), v.optString("language", "english"), v.optBoolean("custom"))
            }
        },
        defaultVoice = root.optString("voice", "alba"),
    )
}

internal fun parseNarrationChapters(root: JSONObject): List<NarrationChapter> {
    val chapters = root.optJSONArray("chapters") ?: return emptyList()
    return (0 until chapters.length()).mapNotNull { i ->
        chapters.optJSONObject(i)?.let { c ->
            NarrationChapter(c.optInt("index", i), c.optString("href"), c.optString("title"), c.optBoolean("matter"))
        }
    }
}

internal fun parseNarrationSentences(root: JSONObject): List<NarrationSentence> {
    val sentences = root.optJSONArray("sentences") ?: return emptyList()
    return (0 until sentences.length()).mapNotNull { i ->
        sentences.optJSONObject(i)?.let { s ->
            val text = s.optString("text")
            if (text.isBlank()) null
            else NarrationSentence(text, s.optDouble("progression", 0.0).coerceIn(0.0, 1.0), s.optString("before"), s.optString("after"))
        }
    }
}
