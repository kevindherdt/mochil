package net.kdhmedia.mochil.ui

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.Storage
import androidx.compose.material.icons.filled.Delete
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material3.Button
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.IconButton
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Surface
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import net.kdhmedia.mochil.auth.LoginState
import net.kdhmedia.mochil.local.LocalSourceViewModel
import net.kdhmedia.mochil.local.LocalMediaKind
import net.kdhmedia.mochil.local.LocalMediaItem
import net.kdhmedia.mochil.audio.AudioTrack
import net.kdhmedia.mochil.playback.PlaybackViewModel

@Composable
fun LocalFirstApp(
    state: LoginState,
    onServerChange: (String) -> Unit,
    onUsernameChange: (String) -> Unit,
    onPasswordChange: (String) -> Unit,
    onLogin: () -> Unit,
    localSources: LocalSourceViewModel = viewModel(),
    playbackViewModel: PlaybackViewModel = viewModel(),
) {
    val context = androidx.compose.ui.platform.LocalContext.current
    // A crash while signed out leaves the person on this screen, so the report has to be readable
    // here: the settings screen that normally shows it is behind a sign-in that just crashed.
    var pendingCrash by remember { mutableStateOf(runCatching { net.kdhmedia.mochil.diagnostics.CrashReportStore(context).pending() }.getOrNull()) }
    var connectingServer by rememberSaveable { mutableStateOf(false) }
    var opdsCatalogues by rememberSaveable { mutableStateOf(false) }
    var webDavLibraries by rememberSaveable { mutableStateOf(false) }

    var selectedKindName by rememberSaveable { mutableStateOf(LocalMediaKind.Publications.name) }
    val selectedKind = runCatching { LocalMediaKind.valueOf(selectedKindName) }.getOrDefault(LocalMediaKind.Publications)
    val sourceState by localSources.state.collectAsStateWithLifecycle()
    val folderPicker = rememberLauncherForActivityResult(ActivityResultContracts.OpenDocumentTree(), localSources::add)
    if (connectingServer) {
        LoginScreen(
            state = state,
            onServerChange = onServerChange,
            onUsernameChange = onUsernameChange,
            onPasswordChange = onPasswordChange,
            onLogin = onLogin,
            onBack = { connectingServer = false },
            crashReport = pendingCrash,
            onDismissCrashReport = {
                net.kdhmedia.mochil.diagnostics.CrashReportStore(context).clear()
                pendingCrash = null
            },
        )
        return
    }
    if (opdsCatalogues) {
        OpdsScreen(onBack = { opdsCatalogues = false })
        return
    }
    if (webDavLibraries) {
        WebDavScreen(onBack = { webDavLibraries = false })
        return
    }
    Scaffold(bottomBar = { MobileMiniPlayerHost(playbackViewModel) {} }) { scaffoldPadding ->
        Column(
            Modifier.fillMaxSize().padding(scaffoldPadding).statusBarsPadding().navigationBarsPadding().padding(horizontal = 24.dp, vertical = 24.dp),
            verticalArrangement = Arrangement.spacedBy(24.dp),
        ) {
            Column(verticalArrangement = Arrangement.spacedBy(5.dp)) {
                Text("Mochil", style = MaterialTheme.typography.headlineLarge, fontWeight = FontWeight.SemiBold)
                Text("Your media, on this device", style = MaterialTheme.typography.bodyLarge, color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
            LocalMediaLibrary(
                state = sourceState,
                selectedKind = selectedKind,
                onKind = { selectedKindName = it.name },
                onOpen = { item -> openLocalMedia(item, sourceState.items, localSources, playbackViewModel) },
                onRefresh = localSources::refresh,
                onRemove = localSources::remove,
                modifier = Modifier.weight(1f),
            )
            Column(Modifier.fillMaxWidth(), verticalArrangement = Arrangement.spacedBy(10.dp)) {
                LocalSourceActions(
                    onServer = { connectingServer = true },
                    onFolder = { folderPicker.launch(null) },
                    onOpds = { opdsCatalogues = true },
                    onWebDav = { webDavLibraries = true },
                )
                state.error?.let { Text(it, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall) }
            }
        }
    }
}

@Composable
internal fun LocalSourceActions(onServer: () -> Unit, onFolder: () -> Unit, onOpds: () -> Unit, onWebDav: () -> Unit) {
    var sourceActionsExpanded by rememberSaveable { mutableStateOf(false) }
    Box(Modifier.fillMaxWidth()) {
        OutlinedButton(onClick = { sourceActionsExpanded = true }, modifier = Modifier.fillMaxWidth()) {
            Icon(Icons.Default.Add, contentDescription = null)
            Text("Manage sources", Modifier.padding(start = 8.dp))
        }
        if (sourceActionsExpanded) {
            DropdownMenu(expanded = true, onDismissRequest = { sourceActionsExpanded = false }) {
                DropdownMenuItem(text = { Text("Connect Mochil server") }, onClick = { sourceActionsExpanded = false; onServer() })
                DropdownMenuItem(text = { Text("Add local media folder") }, onClick = { sourceActionsExpanded = false; onFolder() })
                DropdownMenuItem(text = { Text("OPDS catalogues") }, onClick = { sourceActionsExpanded = false; onOpds() })
                DropdownMenuItem(text = { Text("WebDAV libraries") }, onClick = { sourceActionsExpanded = false; onWebDav() })
            }
        }
    }
}

private fun openLocalMedia(item: LocalMediaItem, allItems: List<LocalMediaItem>, sources: LocalSourceViewModel, playback: PlaybackViewModel) {
    when (item.kind) {
        LocalMediaKind.Publications -> sources.openPublication(item)
        LocalMediaKind.Video -> { playback.pause(); sources.openVideo(item) }
        LocalMediaKind.Audio -> {
            val queue = allItems.filter { it.kind == LocalMediaKind.Audio }.map(LocalMediaItem::asAudioTrack)
            val selected = queue.firstOrNull { it.id == item.documentUri } ?: return
            playback.play(queue, selected, { it.id })
        }
    }
}

private fun LocalMediaItem.asAudioTrack() = AudioTrack(
    id = documentUri, libraryId = "local", title = title, artist = artist, album = album.ifBlank { "On this device" }, durationSeconds = durationSeconds,
    mediaType = mimeType.ifBlank { "audio/*" }, libraryKind = "music",
)
