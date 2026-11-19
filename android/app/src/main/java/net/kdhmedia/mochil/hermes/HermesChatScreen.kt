package net.kdhmedia.mochil.hermes

import net.kdhmedia.mochil.ui.PendingShellOpen
import net.kdhmedia.mochil.ui.MobileShellDestination
import net.kdhmedia.mochil.ui.MobileLibraryRail
import android.content.ClipData
import android.content.ClipboardManager
import android.content.Context
import android.Manifest
import android.content.pm.PackageManager
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.ime
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.selection.SelectionContainer
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.Send
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.Close
import androidx.compose.material.icons.filled.ContentCopy
import androidx.compose.material.icons.filled.Delete
import androidx.compose.material.icons.filled.Edit
import androidx.compose.material.icons.filled.Favorite
import androidx.compose.material.icons.filled.FavoriteBorder
import androidx.compose.material.icons.filled.Menu
import androidx.compose.material.icons.filled.Mic
import androidx.compose.material.icons.filled.MicOff
import androidx.compose.material.icons.filled.Settings
import androidx.compose.material.icons.filled.VolumeUp
import androidx.compose.material.icons.filled.Stop
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.DrawerValue
import net.kdhmedia.mochil.ui.theme.LocalReduceMotion
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.ModalDrawerSheet
import androidx.compose.material3.ModalNavigationDrawer
import androidx.compose.material3.NavigationDrawerItem
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Slider
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Surface
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.rememberDrawerState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.core.content.ContextCompat
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleEventObserver
import androidx.lifecycle.compose.LocalLifecycleOwner
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun HermesChatScreen(viewModel: HermesChatViewModel, onClose: () -> Unit, onPlayCard: (HermesCard, List<HermesCard>) -> Unit = { _, _ -> }) {
    val state by viewModel.state.collectAsStateWithLifecycle()
    val drawer = rememberDrawerState(DrawerValue.Closed)
    val scope = rememberCoroutineScope()
    val snackbars = remember { SnackbarHostState() }
    var showMemories by rememberSaveable { mutableStateOf(false) }
    var showVoice by rememberSaveable { mutableStateOf(false) }
    var picking by rememberSaveable { mutableStateOf(false) }
    val context = LocalContext.current
    val microphone = rememberLauncherForActivityResult(ActivityResultContracts.RequestPermission()) { granted ->
        if (granted) viewModel.toggleTalk() else viewModel.say("Allow the microphone to talk with her")
    }
    val toggleTalk = {
        if (ContextCompat.checkSelfPermission(context, Manifest.permission.RECORD_AUDIO) == PackageManager.PERMISSION_GRANTED) viewModel.toggleTalk()
        else microphone.launch(Manifest.permission.RECORD_AUDIO)
    }
    // Nothing may keep listening or talking once the screen is out of sight.
    val lifecycleOwner = LocalLifecycleOwner.current
    DisposableEffect(lifecycleOwner) {
        val observer = LifecycleEventObserver { _, event -> if (event == Lifecycle.Event.ON_STOP) viewModel.leave() }
        lifecycleOwner.lifecycle.addObserver(observer)
        onDispose { lifecycleOwner.lifecycle.removeObserver(observer) }
    }
    var renaming by remember { mutableStateOf<HermesChat?>(null) }
    var deleting by remember { mutableStateOf<HermesChat?>(null) }
    LaunchedEffect(Unit) { viewModel.load() }
    LaunchedEffect(state.notice) { state.notice?.let { snackbars.showSnackbar(it); viewModel.clearNotice() } }

    ModalNavigationDrawer(
        drawerState = drawer,
        drawerContent = {
            ModalDrawerSheet {
                Text(state.name, style = MaterialTheme.typography.titleLarge, modifier = Modifier.padding(horizontal = 24.dp, vertical = 20.dp))
                NavigationDrawerItem(
                    label = { Text("New chat") }, selected = false, icon = { Icon(Icons.Default.Add, null) },
                    onClick = { viewModel.newChat(); scope.launch { drawer.close() } },
                    modifier = Modifier.padding(horizontal = 12.dp),
                )
                HorizontalDivider(Modifier.padding(vertical = 8.dp))
                LazyColumn(Modifier.weight(1f)) {
                    items(state.chats, key = HermesChat::id) { chat ->
                        Row(Modifier.padding(horizontal = 12.dp), verticalAlignment = Alignment.CenterVertically) {
                            NavigationDrawerItem(
                                label = { Text(chat.title, maxLines = 1, overflow = TextOverflow.Ellipsis) },
                                selected = chat.id == state.currentId,
                                onClick = { viewModel.openChat(chat.id); scope.launch { drawer.close() } },
                                modifier = Modifier.weight(1f),
                            )
                            IconButton({ renaming = chat }) { Icon(Icons.Default.Edit, "Rename ${chat.title}", Modifier.size(20.dp)) }
                            IconButton({ deleting = chat }) { Icon(Icons.Default.Delete, "Delete ${chat.title}", Modifier.size(20.dp)) }
                        }
                    }
                }
                if (state.memoryOn) {
                    HorizontalDivider()
                    NavigationDrawerItem(
                        label = { Text("What she remembers") }, selected = false, icon = { Icon(Icons.Default.FavoriteBorder, null) },
                        onClick = { showMemories = true; viewModel.loadMemories(); scope.launch { drawer.close() } },
                        modifier = Modifier.padding(12.dp).navigationBarsPadding(),
                    )
                }
            }
        },
    ) {
        Scaffold(
            topBar = {
                TopAppBar(
                    title = {
                        androidx.compose.foundation.layout.Row(verticalAlignment = Alignment.CenterVertically) {
                            // The same Home / library switcher as the rest of the app; choosing something leaves the chat and the app goes there.
                            if (state.libraries.isNotEmpty()) Box(Modifier.width(84.dp)) {
                                MobileLibraryRail(
                                    state.libraries, null, MobileShellDestination.Home,
                                    { PendingShellOpen.target = PendingShellOpen.HOME; onClose() },
                                    false, false, {}, {},
                                    { PendingShellOpen.target = PendingShellOpen.SETTINGS; onClose() },
                                    { library -> PendingShellOpen.target = PendingShellOpen.library(library.id); onClose() },
                                    chatName = state.name,
                                )
                            }
                            Text(state.name, maxLines = 1, overflow = TextOverflow.Ellipsis, modifier = Modifier.padding(start = 8.dp))
                        }
                    },
                    navigationIcon = { IconButton({ scope.launch { drawer.open() } }) { Icon(Icons.Default.Menu, "Conversations") } },
                    actions = {
                        if (state.voice.canSpeak || state.avatar != null) IconButton({ showVoice = true }) { Icon(Icons.Default.Settings, "Chat settings") }
                        if (state.memoryOn) IconButton({ showMemories = true; viewModel.loadMemories() }) { Icon(Icons.Default.FavoriteBorder, "What she remembers") }
                        IconButton(onClose) { Icon(Icons.Default.Close, "Close chat") }
                    },
                )
            },
            snackbarHost = { SnackbarHost(snackbars) },
        ) { padding ->
            Column(Modifier.fillMaxSize().padding(padding).imePadding()) {
                when {
                    state.loading -> Box(Modifier.weight(1f).fillMaxWidth(), contentAlignment = Alignment.Center) {
                        if (LocalReduceMotion.current) CircularProgressIndicator(progress = { 1f }) else CircularProgressIndicator()
                    }
                    !state.available -> Box(Modifier.weight(1f).fillMaxWidth().padding(32.dp), contentAlignment = Alignment.Center) {
                        Text("The chat is only for administrators, and has to be switched on in the server's modules.", style = MaterialTheme.typography.bodyLarge)
                    }
                    else -> {
                        // The avatar steps aside while the keyboard is up, so the messages keep their room.
                        val keyboardUp = WindowInsets.ime.getBottom(LocalDensity.current) > 0
                        val deviceCredential = remember(state.avatar) { viewModel.deviceCredential() }
                        // "Transparent" scene: she stands in the chat itself, in front of the messages, which keep scrolling behind her.
                        val standing = state.avatar?.scene == "clear"
                        val avatarOn = state.avatar != null && state.showAvatar && deviceCredential != null
                        @Composable
                        fun avatar(modifier: Modifier) = HermesAvatarView(
                            credential = deviceCredential!!, info = state.avatar!!.model, look = state.avatar!!.lookJson(), level = viewModel::voiceLevel,
                            reply = if (state.busy) state.messages.lastOrNull()?.takeIf { it.role == "assistant" }?.text.orEmpty() else "",
                            speaking = state.speaking, modifier = modifier, passTouches = standing,
                            onError = { viewModel.say("Avatar: $it") },
                            onCrash = { viewModel.avatarCrashed() },
                        )
                        if (avatarOn && !standing && !keyboardUp) avatar(Modifier.fillMaxWidth().height(220.dp))
                        Box(Modifier.weight(1f).fillMaxWidth()) {
                            Messages(state, deviceCredential, viewModel::remember, viewModel::readAloud, onPlayCard, { action -> if (action.pick) picking = true else viewModel.runAction(action) }, Modifier.fillMaxSize())
                            if (avatarOn && standing) avatar(Modifier.fillMaxSize())
                        }
                        state.error?.let { Text(it, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodyMedium, modifier = Modifier.padding(horizontal = 16.dp, vertical = 4.dp)) }
                        Composer(state, { viewModel.send(it) }, viewModel::stop, toggleTalk, viewModel::stopSpeaking) { action -> if (action.pick) picking = true else viewModel.runAction(action) }
                    }
                }
            }
        }
    }

    renaming?.let { chat ->
        var title by remember(chat.id) { mutableStateOf(chat.title) }
        AlertDialog(
            onDismissRequest = { renaming = null },
            title = { Text("Rename chat") },
            text = { OutlinedTextField(title, { title = it }, singleLine = true) },
            confirmButton = { TextButton({ viewModel.rename(chat.id, title); renaming = null }, enabled = title.isNotBlank()) { Text("Save") } },
            dismissButton = { TextButton({ renaming = null }) { Text("Cancel") } },
        )
    }
    deleting?.let { chat ->
        AlertDialog(
            onDismissRequest = { deleting = null },
            title = { Text("Delete this chat?") },
            text = { Text("“${chat.title}” will be removed from ${state.name}'s chats. What she remembers stays.") },
            confirmButton = { TextButton({ viewModel.delete(chat.id); deleting = null }) { Text("Delete") } },
            dismissButton = { TextButton({ deleting = null }) { Text("Cancel") } },
        )
    }
    if (showMemories) MemorySheet(state, viewModel::loadMemories) { showMemories = false }
    if (picking) BookPicker(viewModel, onPick = { book -> picking = false; viewModel.runAction(HERMES_ACTIONS.first { it.id == "book" }, book) }) { picking = false }
    if (showVoice) VoiceSheet(state.voice, state.avatar, state.showAvatar, viewModel) { showVoice = false }
}

@Composable
private fun Messages(state: HermesChatState, credential: net.kdhmedia.mochil.auth.DeviceCredential?, onRemember: (String) -> Unit, onRead: (String) -> Unit, onPlayCard: (HermesCard, List<HermesCard>) -> Unit, onAction: (HermesAction) -> Unit, modifier: Modifier) {
    val listState = rememberLazyListState()
    val last = state.messages.lastOrNull()
    LaunchedEffect(state.messages.size, last?.text?.length, state.steps.size) {
        if (state.messages.isNotEmpty()) listState.scrollToItem(state.messages.lastIndex)
    }
    if (state.messages.isEmpty()) {
        Column(modifier.fillMaxWidth().padding(32.dp), verticalArrangement = Arrangement.Center, horizontalAlignment = Alignment.CenterHorizontally) {
            Box(Modifier.size(64.dp).clip(CircleShape).background(MaterialTheme.colorScheme.primary), contentAlignment = Alignment.Center) {
                Text(state.name.take(1).uppercase(), color = MaterialTheme.colorScheme.onPrimary, style = MaterialTheme.typography.headlineMedium)
            }
            Text(state.name, style = MaterialTheme.typography.titleLarge, modifier = Modifier.padding(top = 12.dp))
            Text("Say something to start the conversation.", color = MaterialTheme.colorScheme.onSurfaceVariant, modifier = Modifier.padding(top = 4.dp))
            Column(Modifier.padding(top = 16.dp), horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.spacedBy(8.dp)) {
                HERMES_ACTIONS.filter { it.id == "recent" || it.id == "next" }.forEach { action ->
                    androidx.compose.material3.OutlinedButton({ onAction(action) }, enabled = !state.busy) { Text(action.label) }
                }
            }
        }
        return
    }
    LazyColumn(modifier.fillMaxWidth(), state = listState, contentPadding = androidx.compose.foundation.layout.PaddingValues(horizontal = 12.dp, vertical = 8.dp), verticalArrangement = Arrangement.spacedBy(10.dp)) {
        itemsIndexed(state.messages, key = { _, message -> message.id }) { index, message ->
            val streaming = state.busy && index == state.messages.lastIndex && !message.fromUser
            Bubble(message, streaming, if (streaming) state.steps else emptyList(), state.voice.canSpeak, credential, onRemember, onRead, onPlayCard)
        }
    }
}

@Composable
private fun Bubble(message: HermesMessage, streaming: Boolean, steps: List<HermesStep>, canRead: Boolean, credential: net.kdhmedia.mochil.auth.DeviceCredential?, onRemember: (String) -> Unit, onRead: (String) -> Unit, onPlayCard: (HermesCard, List<HermesCard>) -> Unit) {
    val context = LocalContext.current
    val mine = message.fromUser
    // The device credential is decrypted once at the screen, not per bubble: AndroidKeyStore work must not repeat per visible message.
    // A message sent by a quick action carries library context; show what was asked, with the context folded away.
    val parts = remember(message.text) { if (mine) splitContext(message.text) else SplitContext(message.text, "") }
    var showContext by rememberSaveable(message.text) { mutableStateOf(false) }
    Column(Modifier.fillMaxWidth(), horizontalAlignment = if (mine) Alignment.End else Alignment.Start) {
        Surface(
            shape = RoundedCornerShape(18.dp, 18.dp, if (mine) 4.dp else 18.dp, if (mine) 18.dp else 4.dp),
            color = if (mine) MaterialTheme.colorScheme.primaryContainer else MaterialTheme.colorScheme.surfaceVariant,
            modifier = Modifier.widthIn(max = 520.dp),
        ) {
            Column(Modifier.padding(horizontal = 14.dp, vertical = 10.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
                steps.forEach { step ->
                    Text(
                        "${hermesToolWords(step.tool)}${if (step.detail.isNotBlank()) ": ${step.detail}" else ""}${if (step.status == HermesStepStatus.Running) "…" else ""}",
                        style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
                if (message.text.isEmpty()) {
                    if (streaming && steps.isEmpty()) Text("…", style = MaterialTheme.typography.bodyLarge, modifier = Modifier.semantics { contentDescription = "Typing" })
                } else SelectionContainer { Text(parts.prompt, style = MaterialTheme.typography.bodyLarge) }
                if (parts.context.isNotEmpty()) {
                    Text(
                        if (showContext) "Hide library context" else "Library context sent with this",
                        style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.primary,
                        modifier = Modifier.clickable { showContext = !showContext }.padding(vertical = 4.dp),
                    )
                    if (showContext) Text(parts.context, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                }
            }
        }
        message.cards.forEach { card ->
            Surface(shape = RoundedCornerShape(14.dp), color = MaterialTheme.colorScheme.surface, tonalElevation = 2.dp, modifier = Modifier.padding(top = 6.dp).widthIn(max = 520.dp).fillMaxWidth()) {
                Row(Modifier.padding(start = 8.dp, top = 6.dp, bottom = 6.dp, end = 6.dp), verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                    if (credential != null) net.kdhmedia.mochil.ui.PublicationArtwork(
                        net.kdhmedia.mochil.audio.AudioClient().artworkUrl(credential, net.kdhmedia.mochil.audio.AudioTrack(id = card.id, libraryId = card.library, title = card.title, artist = "", album = "", durationSeconds = 0.0)),
                        "Bearer ${credential.token}", card.title, Modifier.size(48.dp),
                    )
                    Column(Modifier.weight(1f)) {
                        Text(card.title, style = MaterialTheme.typography.titleSmall, maxLines = 1, overflow = androidx.compose.ui.text.style.TextOverflow.Ellipsis)
                        val line = listOf(card.subtitle, card.kind.takeIf { it != "music" }.orEmpty()).filter { it.isNotBlank() }.joinToString(" · ")
                        if (line.isNotEmpty()) Text(line, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant, maxLines = 1, overflow = androidx.compose.ui.text.style.TextOverflow.Ellipsis)
                    }
                    androidx.compose.material3.FilledTonalButton({ onPlayCard(card, message.cards) }) { Text("Play") }
                }
            }
        }
        if (!streaming && message.text.isNotEmpty()) Row {
            IconButton({
                val clipboard = context.getSystemService(Context.CLIPBOARD_SERVICE) as ClipboardManager
                clipboard.setPrimaryClip(ClipData.newPlainText("message", parts.prompt))
            }, Modifier.size(36.dp)) { Icon(Icons.Default.ContentCopy, "Copy", Modifier.size(18.dp), tint = MaterialTheme.colorScheme.onSurfaceVariant) }
            IconButton({ onRemember(parts.prompt) }, Modifier.size(36.dp)) { Icon(Icons.Default.FavoriteBorder, "Remember this", Modifier.size(18.dp), tint = MaterialTheme.colorScheme.onSurfaceVariant) }
            if (canRead && !mine) IconButton({ onRead(message.text) }, Modifier.size(36.dp)) { Icon(Icons.Default.VolumeUp, "Read aloud", Modifier.size(18.dp), tint = MaterialTheme.colorScheme.onSurfaceVariant) }
        }
    }
}

@Composable
private fun Composer(state: HermesChatState, onSend: (String) -> Unit, onStop: () -> Unit, onTalk: () -> Unit, onStopSpeaking: () -> Unit, onAction: (HermesAction) -> Unit) {
    var text by rememberSaveable { mutableStateOf("") }
    var menu by remember { mutableStateOf(false) }
    val talking = state.talk != TalkMode.Off
    Column(Modifier.fillMaxWidth().navigationBarsPadding()) {
        if (talking || state.speaking) Row(Modifier.fillMaxWidth().padding(horizontal = 16.dp), verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.SpaceBetween) {
            Text(
                when {
                    state.talk == TalkMode.Hearing -> "Hearing you…"
                    state.talk == TalkMode.Transcribing -> "Understanding…"
                    state.speaking -> "${state.name} is speaking"
                    state.talk == TalkMode.Waiting -> "Waiting for ${state.name}…"
                    else -> "Listening… say “stop” to end"
                },
                style = MaterialTheme.typography.labelLarge, color = MaterialTheme.colorScheme.primary,
            )
            if (state.speaking) TextButton(onStopSpeaking) { Text("Stop talking") }
        }
        Row(Modifier.fillMaxWidth().padding(horizontal = 12.dp, vertical = 8.dp), verticalAlignment = Alignment.Bottom, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            Box {
                if (state.quickOn) IconButton({ menu = true }, Modifier.size(52.dp), enabled = !state.busy) { Icon(Icons.Default.Add, "Quick actions") }
                DropdownMenu(menu, { menu = false }) {
                    HERMES_ACTIONS.forEach { action ->
                        DropdownMenuItem(
                            text = { Column { Text(action.label); Text(action.hint, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant) } },
                            onClick = { menu = false; if (action.draft.isNotEmpty()) text = action.draft else onAction(action) },
                        )
                    }
                }
            }
            OutlinedTextField(
                value = text, onValueChange = { text = it }, modifier = Modifier.weight(1f), maxLines = 5,
                placeholder = { Text("Message ${state.name}") }, shape = RoundedCornerShape(24.dp),
            )
            if (state.voice.canListen) IconButton(
                onTalk,
                Modifier.size(52.dp).background(if (talking) MaterialTheme.colorScheme.tertiaryContainer else MaterialTheme.colorScheme.surfaceVariant, CircleShape),
            ) { Icon(if (talking) Icons.Default.MicOff else Icons.Default.Mic, if (talking) "Stop listening" else "Talk to ${state.name}", tint = MaterialTheme.colorScheme.onSurface) }
            if (state.busy) IconButton(onStop, Modifier.size(52.dp).background(MaterialTheme.colorScheme.errorContainer, CircleShape)) { Icon(Icons.Default.Stop, "Stop", tint = MaterialTheme.colorScheme.onErrorContainer) }
            else IconButton({ onSend(text); text = "" }, Modifier.size(52.dp).background(MaterialTheme.colorScheme.primary, CircleShape), enabled = text.isNotBlank()) {
                Icon(Icons.AutoMirrored.Filled.Send, "Send", tint = MaterialTheme.colorScheme.onPrimary)
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun VoiceSheet(voice: HermesVoiceState, avatar: HermesAvatarInfo?, showAvatar: Boolean, viewModel: HermesChatViewModel, onDismiss: () -> Unit) {
    var pick by remember { mutableStateOf(false) }
    var pickAvatar by remember { mutableStateOf(false) }
    val hasAvatar = avatar != null
    ModalBottomSheet(onDismissRequest = onDismiss) {
        Column(Modifier.padding(horizontal = 20.dp).padding(bottom = 24.dp).navigationBarsPadding(), verticalArrangement = Arrangement.spacedBy(4.dp)) {
            Text("Chat settings", style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.SemiBold)
            if (avatar != null && avatar.choices.size > 1) Box {
                val shown = avatar.choices.firstOrNull { it.id == avatar.id }?.label ?: "Default"
                Row(Modifier.fillMaxWidth().clickable { pickAvatar = true }.padding(vertical = 10.dp), horizontalArrangement = Arrangement.SpaceBetween, verticalAlignment = Alignment.CenterVertically) {
                    Column(Modifier.weight(1f)) { Text("Character"); Text("Which avatar appears above the chat. Shared with the website.", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant) }
                    Text(shown, color = MaterialTheme.colorScheme.primary, maxLines = 1, overflow = TextOverflow.Ellipsis, modifier = Modifier.widthIn(max = 160.dp))
                }
                DropdownMenu(pickAvatar, { pickAvatar = false }) {
                    avatar.choices.forEach { item -> DropdownMenuItem(text = { Text(item.label) }, onClick = { viewModel.chooseAvatar(item.id); pickAvatar = false }) }
                }
            }
            if (avatar != null && showAvatar) AvatarLook(avatar, viewModel)
            if (hasAvatar) VoiceRow("Show her avatar", "The character above the chat. Her mouth moves with her voice.", showAvatar, viewModel::setShowAvatar)
            if (voice.canSpeak) VoiceRow("Read replies aloud", "Speaks each sentence as it arrives.", voice.speakReplies, viewModel::setSpeakReplies)
            if (voice.canSpeak) Box {
                val current = voice.voices.firstOrNull { it.id == voice.voiceId }?.name ?: "Default voice"
                Row(Modifier.fillMaxWidth().clickable { pick = true }.padding(vertical = 10.dp), horizontalArrangement = Arrangement.SpaceBetween, verticalAlignment = Alignment.CenterVertically) {
                    Column(Modifier.weight(1f)) { Text("Voice"); Text("Applies right away.", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant) }
                    Text(current, color = MaterialTheme.colorScheme.primary, maxLines = 1, overflow = TextOverflow.Ellipsis, modifier = Modifier.widthIn(max = 180.dp))
                }
                DropdownMenu(pick, { pick = false }) {
                    DropdownMenuItem(text = { Text("Default voice") }, onClick = { viewModel.setVoiceId(""); pick = false })
                    voice.voices.forEach { item -> DropdownMenuItem(text = { Text(item.name) }, onClick = { viewModel.setVoiceId(item.id); pick = false }) }
                }
            }
            if (voice.canListen) {
                VoiceRow("Hands-free conversation", "Keeps listening after each answer. Say “stop” to end it.", voice.handsFree, viewModel::setHandsFree)
                VoiceRow("Interrupt when I speak", "She stops the moment you start. Use headphones so she does not hear herself.", voice.bargeIn, viewModel::setBargeIn)
            }
        }
    }
}

@Composable
private fun VoiceRow(title: String, note: String, checked: Boolean, onChange: (Boolean) -> Unit) {
    Row(Modifier.fillMaxWidth().padding(vertical = 6.dp), horizontalArrangement = Arrangement.SpaceBetween, verticalAlignment = Alignment.CenterVertically) {
        Column(Modifier.weight(1f).padding(end = 12.dp)) { Text(title); Text(note, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant) }
        Switch(checked, onChange)
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun MemorySheet(state: HermesChatState, onRefresh: () -> Unit, onDismiss: () -> Unit) {
    ModalBottomSheet(onDismissRequest = onDismiss) {
        Column(Modifier.padding(horizontal = 20.dp).padding(bottom = 24.dp).navigationBarsPadding(), verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween, verticalAlignment = Alignment.CenterVertically) {
                Text("What ${state.name} remembers", style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.SemiBold)
                TextButton(onRefresh) { Text("Refresh") }
            }
            when {
                state.memoriesError != null -> Text(state.memoriesError, color = MaterialTheme.colorScheme.error)
                state.memories == null -> if (LocalReduceMotion.current) {
                    CircularProgressIndicator(progress = { 1f }, modifier = Modifier.padding(16.dp))
                } else {
                    CircularProgressIndicator(Modifier.padding(16.dp))
                }
                state.memories.isEmpty() -> Text("${state.name} has not remembered anything yet. Talk for a while, or use the heart on a message.", color = MaterialTheme.colorScheme.onSurfaceVariant)
                else -> {
                    Text("${state.memories.size} thing${if (state.memories.size == 1) "" else "s"} picked up from your chats. She keeps them herself; this list is only for looking.", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                    LazyColumn(Modifier.fillMaxWidth(), verticalArrangement = Arrangement.spacedBy(2.dp)) {
                        items(state.memories, key = { "${it.date}:${it.text}" }) { memory ->
                            Column(Modifier.padding(vertical = 8.dp)) {
                                Text(memory.text, style = MaterialTheme.typography.bodyLarge)
                                if (memory.date.isNotBlank()) Text(memory.date.take(10), style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                            }
                            HorizontalDivider()
                        }
                    }
                }
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun BookPicker(viewModel: HermesChatViewModel, onPick: (org.json.JSONObject) -> Unit, onDismiss: () -> Unit) {
    var query by rememberSaveable { mutableStateOf("") }
    var results by remember { mutableStateOf<List<org.json.JSONObject>?>(null) }
    LaunchedEffect(query) {
        if (query.trim().length < 2) { results = null; return@LaunchedEffect }
        kotlinx.coroutines.delay(220)
        results = viewModel.searchBooks(query)
    }
    ModalBottomSheet(onDismissRequest = onDismiss) {
        Column(Modifier.padding(horizontal = 20.dp).padding(bottom = 24.dp).navigationBarsPadding().imePadding(), verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Text("Which book?", style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.SemiBold)
            OutlinedTextField(query, { query = it }, Modifier.fillMaxWidth(), singleLine = true, placeholder = { Text("Search your books") })
            val found = results
            when {
                found == null -> Unit
                found.isEmpty() -> Text("No books found.", color = MaterialTheme.colorScheme.onSurfaceVariant)
                else -> found.forEach { book ->
                    Text(bookLine(book).removePrefix("- "), Modifier.fillMaxWidth().clickable { onPick(book) }.padding(vertical = 10.dp), style = MaterialTheme.typography.bodyLarge)
                }
            }
        }
    }
}

/** Background and position of the avatar. The avatar is above the sheet, so every change can be watched as it is made. */
@Composable
private fun AvatarLook(avatar: HermesAvatarInfo, viewModel: HermesChatViewModel) {
    var open by remember { mutableStateOf(false) }
    val scenes = HermesAvatarScenes + if (avatar.image > 0) listOf("image" to "My picture") else emptyList()
    Box {
        Row(Modifier.fillMaxWidth().clickable { open = true }.padding(vertical = 10.dp), horizontalArrangement = Arrangement.SpaceBetween, verticalAlignment = Alignment.CenterVertically) {
            Column(Modifier.weight(1f)) { Text("Background"); Text("What is behind her. Shared with the website.", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant) }
            Text(scenes.firstOrNull { it.first == avatar.scene }?.second ?: "Soft (default)", color = MaterialTheme.colorScheme.primary, maxLines = 1, overflow = TextOverflow.Ellipsis, modifier = Modifier.widthIn(max = 160.dp))
        }
        DropdownMenu(open, { open = false }) {
            scenes.forEach { (id, label) -> DropdownMenuItem(text = { Text(label) }, onClick = { viewModel.setScene(id); open = false }) }
        }
    }
    Text("Position", style = MaterialTheme.typography.bodyMedium, modifier = Modifier.padding(top = 4.dp))
    val frame = avatar.frame
    FrameSlider("Size", frame.zoom, 0.3f..2.2f, { viewModel.previewFrame(frame.copy(zoom = it)) }, viewModel::saveFrame)
    FrameSlider("Left / right", frame.x, -1f..1f, { viewModel.previewFrame(frame.copy(x = it)) }, viewModel::saveFrame)
    FrameSlider("Up / down", frame.y, -1f..1f, { viewModel.previewFrame(frame.copy(y = it)) }, viewModel::saveFrame)
    TextButton({ viewModel.previewFrame(HermesAvatarFrame()); viewModel.saveFrame() }, enabled = frame != HermesAvatarFrame()) { Text("Reset position") }
}

@Composable
private fun FrameSlider(label: String, value: Float, range: ClosedFloatingPointRange<Float>, onChange: (Float) -> Unit, onDone: () -> Unit) {
    Row(verticalAlignment = Alignment.CenterVertically) {
        Text(label, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant, modifier = Modifier.width(88.dp))
        Slider(value.coerceIn(range), onChange, Modifier.weight(1f), valueRange = range, onValueChangeFinished = onDone)
    }
}
