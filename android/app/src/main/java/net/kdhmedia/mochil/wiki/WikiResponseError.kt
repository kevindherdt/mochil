package net.kdhmedia.mochil.wiki

import org.json.JSONObject

internal class WikiException(
    val status: Int,
    message: String,
    val mutationCommitted: Boolean = false,
) : Exception(message)

/** A generic 500, HTML response, or string boolean cannot acknowledge a write. */
internal fun decodeWikiResponseError(status: Int, text: String): WikiException {
    val payload = runCatching { JSONObject(text) }.getOrNull()
    val committed = status == 500 &&
        payload?.opt("code") == "vault_index_refresh_failed" &&
        payload.opt("committed") == true && payload.opt("index_refreshed") == false
    val message = (payload?.opt("error") as? String)?.takeIf { it.isNotBlank() }
        ?: if (status in 300..399) {
            "The server redirected this Wiki request outside its Android API. Reconnect this device using the current server address."
        } else "The server returned $status."
    return WikiException(status, message, committed)
}
