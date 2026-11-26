package net.kdhmedia.mochil.hermes

import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.compositionLocalOf
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.platform.LocalContext
import net.kdhmedia.mochil.auth.DeviceCredentialStore

/** A menu entry that opens the chat. Null when the server does not offer the chat to this device. */
data class HermesChatEntry(val name: String, val open: () -> Unit)

val LocalHermesChatEntry = compositionLocalOf<HermesChatEntry?> { null }

/**
 * Asks the server once whether this device may use the chat (an administrator's device, with the module on and Hermes
 * set up) and, if so, offers a menu entry named after the assistant to everything inside [content].
 */
@Composable
fun ProvideHermesChatEntry(content: @Composable () -> Unit) {
    val context = LocalContext.current
    var status by remember { mutableStateOf<HermesStatus?>(null) }
    LaunchedEffect(Unit) {
        val credential = DeviceCredentialStore(context).load() ?: return@LaunchedEffect
        status = runCatching { HermesClient().status(credential) }.getOrNull()
    }
    val entry = status?.takeIf { it.configured }?.let { HermesChatEntry(it.name) { context.startActivity(HermesChatActivity.intent(context).addFlags(android.content.Intent.FLAG_ACTIVITY_SINGLE_TOP)) } }
    CompositionLocalProvider(LocalHermesChatEntry provides entry, content = content)
}
