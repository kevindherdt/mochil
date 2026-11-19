package net.kdhmedia.mochil.hermes

import org.json.JSONObject

/** Whether the chat is set up on the server and answering. Only an administrator's device gets this far. */
data class HermesStatus(val configured: Boolean, val reachable: Boolean, val name: String, val memory: Boolean, val voice: Boolean = true, val quick: Boolean = true)

data class HermesChat(val id: String, val title: String, val updated: Double, val count: Int)

/** Something from the library she offers to play, shown as a card with a Play button under her answer. */
data class HermesCard(val id: String, val library: String, val kind: String, val title: String, val subtitle: String)

/** [id] is stable identity for LazyColumn keying: it is generated once per message and
 *  survives `.copy()` (streaming text/cards updates), so the chat list never rekeys the
 *  bubble that is actively growing. */
data class HermesMessage(val role: String, val text: String, val cards: List<HermesCard> = emptyList(), val id: String = java.util.UUID.randomUUID().toString()) {
    val fromUser: Boolean get() = role == "user"
}

/** What speech the server offers: whether it can speak, whether it can hear you, the default voice and the voices. */
data class HermesVoiceStatus(val speak: Boolean, val listen: Boolean, val defaultVoice: String, val voices: List<HermesVoice>)

data class HermesVoice(val id: String, val name: String)

/** The avatar the server shows: the chosen model as the page needs it (JSON text), and the voice kept for it, if any. */
data class HermesAvatarChoice(val id: String, val label: String)

/** How the avatar sits and what is behind her: zoom 1 is the default, x and y move her by up to half the frame. */
data class HermesAvatarFrame(val zoom: Float = 1f, val x: Float = 0f, val y: Float = 0f)

data class HermesAvatarInfo(
    val model: String, val voice: String, val id: String = "", val choices: List<HermesAvatarChoice> = emptyList(), val voices: Map<String, String> = emptyMap(),
    val scene: String = "", val frame: HermesAvatarFrame = HermesAvatarFrame(), val image: Long = 0,
) {
    /** What the avatar page needs to draw the background and place her. */
    fun lookJson(): String = JSONObject().put("scene", scene).put("image", image)
        .put("frame", JSONObject().put("zoom", frame.zoom.toDouble()).put("x", frame.x.toDouble()).put("y", frame.y.toDouble())).toString()
}

/** The backgrounds the website offers, in the same order. "image" is only offered when a picture has been uploaded there. */
val HermesAvatarScenes = listOf("" to "Soft (default)", "clear" to "Transparent (she lives in the chat)", "room" to "Cosy room", "night" to "Night window", "sunset" to "Sunset", "studio" to "Studio", "mint" to "Mint", "rain" to "Rainy window", "desk" to "Desk at night", "loft" to "Bookshop loft", "balcony" to "Balcony dusk")

data class HermesMemory(val text: String, val date: String)

enum class HermesStepStatus { Running, Done, Failed }

/** One thing she is doing before she answers (searching, reading, remembering) or the "waiting" note while she is slow. */
data class HermesStep(val id: String, val tool: String, val detail: String, val status: HermesStepStatus)

internal fun parseHermesStatus(root: JSONObject) = HermesStatus(
    configured = root.optBoolean("configured"),
    reachable = root.optBoolean("reachable"),
    name = root.optString("name").ifBlank { "Hermes" },
    memory = root.optBoolean("memory"),
    voice = root.optJSONObject("features")?.optBoolean("voice", true) ?: true,
    quick = root.optJSONObject("features")?.optBoolean("quick", true) ?: true,
)

internal fun parseHermesChats(root: JSONObject): List<HermesChat> {
    val items = root.optJSONArray("chats") ?: return emptyList()
    return (0 until items.length()).mapNotNull { i ->
        items.optJSONObject(i)?.let { item ->
            val id = item.optString("id")
            if (id.isEmpty()) null else HermesChat(id, item.optString("title").ifBlank { "New chat" }, item.optDouble("updated", 0.0), item.optInt("count"))
        }
    }
}

internal fun parseHermesCards(items: org.json.JSONArray?): List<HermesCard> = (0 until (items?.length() ?: 0)).mapNotNull { i ->
    items?.optJSONObject(i)?.takeIf { it.optString("id").isNotEmpty() && it.optString("library").isNotEmpty() }?.let {
        HermesCard(it.optString("id"), it.optString("library"), it.optString("kind"), it.optString("title").ifBlank { "Untitled" }, it.optString("subtitle"))
    }
}

internal fun parseHermesMessages(root: JSONObject): List<HermesMessage> {
    val items = root.optJSONArray("messages") ?: return emptyList()
    return (0 until items.length()).mapNotNull { i ->
        items.optJSONObject(i)?.let { item ->
            val role = item.optString("role")
            val text = item.optString("text")
            if ((role == "user" || role == "assistant") && text.isNotBlank()) HermesMessage(role, text, parseHermesCards(item.optJSONArray("cards"))) else null
        }
    }
}

internal fun parseHermesMemories(root: JSONObject): List<HermesMemory> {
    val items = root.optJSONArray("items") ?: return emptyList()
    return (0 until items.length()).mapNotNull { i ->
        items.optJSONObject(i)?.let { item ->
            val text = item.optString("text").trim()
            if (text.isEmpty()) null else HermesMemory(text, item.optString("date"))
        }
    }
}

/** A readable name for a tool: known ones in plain words, others from their own name. */
fun hermesToolWords(tool: String): String = when (tool) {
    "web_search" -> "Searching the web"
    "web_extract" -> "Reading a page"
    "memory" -> "Remembering"
    "session_search" -> "Looking back through our chats"
    "ha_list_entities" -> "Looking at the home's devices"
    "ha_get_state" -> "Checking a device"
    "ha_list_services" -> "Checking what a device can do"
    "ha_call_service" -> "Controlling the home"
    "tool_search" -> "Finding a tool"
    "tool_describe" -> "Reading about a tool"
    "skill_view" -> "Checking a skill"
    "skills_list" -> "Checking her skills"
    "todo" -> "Making a list"
    "vision_analyze" -> "Looking at an image"
    "waiting" -> "Waiting"
    else -> tool.ifBlank { "working" }.replace(Regex("[_-]+"), " ").replaceFirstChar { it.uppercase() }
}

internal fun parseHermesVoiceStatus(root: JSONObject): HermesVoiceStatus {
    val items = root.optJSONArray("voices")
    val voices = if (items == null) emptyList() else (0 until items.length()).mapNotNull { i ->
        items.optJSONObject(i)?.let { item -> item.optString("id").takeIf { it.isNotEmpty() }?.let { HermesVoice(it, item.optString("name", it)) } }
    }
    return HermesVoiceStatus(root.optBoolean("available"), root.optBoolean("listen"), root.optString("voice"), voices)
}

/** The chosen avatar from the server's model list, or null when it has none, or the Live2D core is missing. */
internal fun parseHermesAvatarInfo(root: JSONObject): HermesAvatarInfo? {
    if (!root.optBoolean("core")) return null
    val models = root.optJSONArray("models") ?: return null
    val selected = root.optString("selected")
    val chosen = (0 until models.length()).mapNotNull { models.optJSONObject(it) }.let { list -> list.firstOrNull { it.optString("id") == selected } ?: list.firstOrNull() } ?: return null
    val all = (0 until models.length()).mapNotNull { models.optJSONObject(it) }
    val saved = root.optJSONObject("voices")
    val voices = saved?.keys()?.asSequence()?.associateWith { saved.optString(it) }.orEmpty()
    val choices = all.map { HermesAvatarChoice(it.optString("id"), if (it.optString("id") == ".") it.optString("name") else it.optString("id")) }
    val frameJson = root.optJSONObject("frames")?.optJSONObject(chosen.optString("id"))
    val frame = frameJson?.let { HermesAvatarFrame(it.optDouble("zoom", 1.0).toFloat(), it.optDouble("x", 0.0).toFloat(), it.optDouble("y", 0.0).toFloat()) } ?: HermesAvatarFrame()
    return HermesAvatarInfo(chosen.toString(), voices[chosen.optString("id")].orEmpty(), chosen.optString("id"), choices, voices, root.optString("scene"), frame, root.optLong("image"))
}
