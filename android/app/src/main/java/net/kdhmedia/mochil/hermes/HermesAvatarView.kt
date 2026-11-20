package net.kdhmedia.mochil.hermes

import android.annotation.SuppressLint
import android.graphics.Color
import android.webkit.JavascriptInterface
import android.webkit.WebResourceRequest
import android.webkit.WebResourceResponse
import android.webkit.WebView
import android.webkit.WebViewClient
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.viewinterop.AndroidView
import java.io.ByteArrayInputStream
import java.net.HttpURLConnection
import java.net.URI
import kotlinx.coroutines.delay
import net.kdhmedia.mochil.auth.DeviceCredential
import org.json.JSONObject

/**
 * Her avatar: the website's own Live2D page, shown in a WebView. The page and its models come from your server. The WebView
 * never talks to the network itself: every request is answered by [AvatarWebClient], which adds the device token to the
 * avatar files and refuses everything else, so the token never reaches a page and no other site can be loaded.
 */
@SuppressLint("SetJavaScriptEnabled")
@Composable
fun HermesAvatarView(
    credential: DeviceCredential,
    info: String,
    look: String,
    passTouches: Boolean = false, // standing in front of the chat: taps and scrolls go to what is behind her
    level: () -> Float,
    reply: String,
    speaking: Boolean,
    modifier: Modifier = Modifier,
    onError: (String) -> Unit = {},
    onCrash: () -> Unit = {},
) {
    var webView by remember { mutableStateOf<WebView?>(null) }
    var ready by remember { mutableStateOf(false) }

    AndroidView(
        modifier = modifier,
        factory = { context ->
            object : WebView(context) {
                override fun dispatchTouchEvent(event: android.view.MotionEvent): Boolean = if (passTouches) false else super.dispatchTouchEvent(event)
            }.apply {
                setBackgroundColor(Color.TRANSPARENT)
                settings.javaScriptEnabled = true
                settings.allowFileAccess = false
                settings.allowContentAccess = false
                settings.setSupportMultipleWindows(false)
                val origin = avatarOrigin(credential.serverUrl)
                val page = context.assets.open("hermes_avatar.html").bufferedReader().use { it.readText() }
                webViewClient = AvatarWebClient(origin, credential.token, page.toByteArray()) { post { ready = false; webView = null; onCrash() } }
                addJavascriptInterface(object {
                    @JavascriptInterface fun ready() { post { ready = true } }
                    @JavascriptInterface fun error(message: String) { post { onError(message) } }
                }, "Android")
                loadUrl("$origin$AVATAR_PAGE_PATH")
                webView = this
            }
        },
        onRelease = { view -> view.removeJavascriptInterface("Android"); view.destroy(); if (webView === view) webView = null },
    )

    LaunchedEffect(ready, info) { if (ready) webView?.evaluateJavascript("hermesAvatar.show($info)", null) }
    // The background and where she sits: set when the page is ready, and again whenever either changes.
    LaunchedEffect(ready, look) { if (ready) webView?.evaluateJavascript("hermesAvatar.look($look)", null) }
    LaunchedEffect(ready, reply) { if (ready && reply.isNotEmpty()) webView?.evaluateJavascript("hermesAvatar.react(${JSONObject.quote(reply)})", null) }
    LaunchedEffect(ready, speaking) { if (ready && !speaking) webView?.evaluateJavascript("hermesAvatar.calm()", null) }
    // The mouth follows what is playing right now, about thirty times a second, and rests when nothing plays.
    LaunchedEffect(ready) {
        var last = 0f
        while (ready) {
            val now = level()
            if (now > 0f || last > 0f) webView?.evaluateJavascript("hermesAvatar.level(${"%.3f".format(java.util.Locale.ROOT, now)})", null)
            last = now
            delay(33)
        }
    }
}

internal fun avatarOrigin(serverUrl: String): String {
    val uri = URI(serverUrl.trimEnd('/'))
    return "${uri.scheme}://${uri.authority}"
}

/** Where a request from the avatar page really goes, or null when it must be refused. Pure so it can be tested. */
internal fun avatarTarget(origin: String, requestOrigin: String, path: String): String? {
    if (requestOrigin != origin) return null
    return when {
        path.startsWith("/api/avatar/") && canonicalAssetPath(path)?.startsWith("/api/avatar/") == true ->
            "$origin/api/android/v1/avatar/" + path.removePrefix("/api/avatar/")
        path.startsWith("/assets/") && canonicalAssetPath(path)?.startsWith("/assets/") == true ->
            origin + path
        else -> null
    }
}

/**
 * A literal `!path.contains("..")` check only catches a traversal segment written out in full; a
 * percent-encoded one (`%2e%2e`, or a doubly-encoded `%252e%252e`) slips straight past it because the
 * substring never appears in the raw path. This decodes the path and resolves every `.`/`..` segment
 * against a bare root, returning the canonical result (or null once a segment climbs above the root)
 * so callers can check it never leaves the root it claims to be under — the actual request still uses
 * the original, still-encoded path, so a legitimately encoded filename is never altered.
 */
internal fun canonicalAssetPath(path: String): String? {
    var decoded = path
    // Decode repeatedly (bounded) so a doubly-encoded segment cannot hide behind one decode pass.
    repeat(3) {
        val next = runCatching { java.net.URLDecoder.decode(decoded, "UTF-8") }.getOrNull() ?: return@repeat
        if (next == decoded) return@repeat
        decoded = next
    }
    if (!decoded.startsWith("/")) return null
    val stack = ArrayDeque<String>()
    for (segment in decoded.split("/")) {
        when (segment) {
            "", "." -> Unit
            ".." -> { if (stack.isEmpty()) return null; stack.removeLast() }
            else -> stack.addLast(segment)
        }
    }
    return "/" + stack.joinToString("/")
}

/** The page is part of the app; it is served under the server's own address so its scripts and files are same-origin. */
internal const val AVATAR_PAGE_PATH = "/hermes-avatar-app"

private class AvatarWebClient(private val origin: String, private val token: String, private val page: ByteArray, private val onGone: () -> Unit) : WebViewClient() {
    // When the phone kills the page's renderer (usually for memory), Android closes the whole app unless this says it was handled.
    override fun onRenderProcessGone(view: WebView, detail: android.webkit.RenderProcessGoneDetail): Boolean { onGone(); return true }

    private fun refused() = WebResourceResponse("text/plain", "utf-8", 403, "Refused", emptyMap(), ByteArrayInputStream(ByteArray(0)))

    override fun shouldInterceptRequest(view: WebView, request: WebResourceRequest): WebResourceResponse? {
        val url = request.url
        if (request.method != "GET") return refused()
        val path = url.encodedPath ?: return refused()
        if (path == AVATAR_PAGE_PATH && "${url.scheme}://${url.authority}" == origin) {
            return WebResourceResponse("text/html", "utf-8", 200, "OK", mapOf("Cache-Control" to "no-store"), ByteArrayInputStream(page))
        }
        val target = avatarTarget(origin, "${url.scheme}://${url.authority}", path) ?: return refused()
        return try {
            val connection = URI(target + (url.encodedQuery?.let { "?$it" } ?: "")).toURL().openConnection() as HttpURLConnection
            connection.connectTimeout = 10_000
            connection.readTimeout = 30_000
            connection.instanceFollowRedirects = false
            if (path.startsWith("/api/")) connection.setRequestProperty("Authorization", "Bearer $token")
            val status = connection.responseCode
            if (status !in 200..299) return WebResourceResponse("text/plain", "utf-8", if (status in 400..599) status else 502, "Failed", emptyMap(), ByteArrayInputStream(ByteArray(0)))
            val type = connection.contentType?.substringBefore(';')?.trim().orEmpty().ifEmpty { guessType(path) }
            WebResourceResponse(type, if (type.startsWith("text/") || type.endsWith("javascript") || type.endsWith("json")) "utf-8" else null, 200, "OK", emptyMap(), connection.inputStream)
        } catch (error: Exception) {
            WebResourceResponse("text/plain", "utf-8", 502, "Failed", emptyMap(), ByteArrayInputStream(ByteArray(0)))
        }
    }

    // Pages cannot navigate away: the only page is the one the app loaded.
    override fun shouldOverrideUrlLoading(view: WebView, request: WebResourceRequest) = true

    private fun guessType(path: String) = when {
        path.endsWith(".js") -> "text/javascript"
        path.endsWith(".json") -> "application/json"
        path.endsWith(".png") -> "image/png"
        path.endsWith(".jpg") || path.endsWith(".jpeg") -> "image/jpeg"
        else -> "application/octet-stream"
    }
}
