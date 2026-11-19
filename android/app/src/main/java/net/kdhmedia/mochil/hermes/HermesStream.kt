package net.kdhmedia.mochil.hermes

import org.json.JSONObject

/** What one line of the chat stream changed: a piece of the answer, the list of steps, or a failure. */
data class HermesStreamUpdate(val text: String? = null, val steps: List<HermesStep>? = null, val error: String? = null, val cards: List<HermesCard>? = null)

private const val WAIT_ID = "wait"

/**
 * Reads the chat stream one line at a time, the same way the website does. The stream is server-sent events:
 * "event:" names the next "data:" line, a blank line ends an event, and lines starting with ":" are ignored.
 * Between the text chunks come tool progress, a "waiting" note while she is slow, and an error if the relay failed.
 */
class HermesStreamReader {
    private var event = ""
    private val steps = mutableListOf<HermesStep>()

    fun steps(): List<HermesStep> = steps.toList()

    fun feed(raw: String): HermesStreamUpdate? {
        val line = raw.trim()
        if (line.isEmpty()) { event = ""; return null }
        if (line.startsWith("event:")) { event = line.removePrefix("event:").trim(); return null }
        if (!line.startsWith("data:")) return null
        val data = line.removePrefix("data:").trim()
        if (data.isEmpty() || data == "[DONE]") return null
        val body = runCatching { JSONObject(data) }.getOrNull() ?: return null
        return when (event) {
            "hermes.waiting" -> {
                setStep(HermesStep(WAIT_ID, "waiting", "${body.optInt("seconds")} s", HermesStepStatus.Running))
                HermesStreamUpdate(steps = steps())
            }
            "hermes.cards" -> {
                HermesStreamUpdate(cards = parseHermesCards(body.optJSONArray("cards")))
            }
            "hermes.error" -> HermesStreamUpdate(error = body.optString("error").ifBlank { "The answer was interrupted" })
            "hermes.tool.progress" -> {
                val tool = body.optString("tool")
                val id = body.optString("toolCallId").ifEmpty { "$tool-${steps.size}" }
                val old = steps.firstOrNull { it.id == id }
                val detail = body.optString("label").ifEmpty { old?.detail.orEmpty() }
                val status = when (body.optString("status")) {
                    "completed" -> HermesStepStatus.Done
                    "failed", "error" -> HermesStepStatus.Failed
                    else -> HermesStepStatus.Running
                }
                setStep(HermesStep(id, old?.tool ?: tool, detail, status))
                HermesStreamUpdate(steps = steps())
            }
            else -> {
                val delta = body.optJSONArray("choices")?.optJSONObject(0)?.optJSONObject("delta")?.optString("content").orEmpty()
                if (delta.isEmpty()) return null
                val waiting = steps.indexOfFirst { it.id == WAIT_ID }
                if (waiting < 0) HermesStreamUpdate(text = delta)
                else { steps.removeAt(waiting); HermesStreamUpdate(text = delta, steps = steps()) } // the answer has started
            }
        }
    }

    private fun setStep(step: HermesStep) {
        val at = steps.indexOfFirst { it.id == step.id }
        if (at < 0) steps.add(step) else steps[at] = step
    }
}
