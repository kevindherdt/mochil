package net.kdhmedia.mochil.ui

import androidx.compose.foundation.layout.Arrangement
import androidx.activity.compose.BackHandler
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.background
import androidx.compose.ui.draw.clip
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.ScrollState
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.FilterChip
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.ui.Alignment
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.semantics.LiveRegionMode
import androidx.compose.ui.semantics.liveRegion
import androidx.compose.ui.semantics.heading
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import net.kdhmedia.mochil.BuildConfig
import net.kdhmedia.mochil.catalogue.DownloadLocation
import net.kdhmedia.mochil.catalogue.ClientDeviceClass
import net.kdhmedia.mochil.catalogue.LibrarySummary
import net.kdhmedia.mochil.diagnostics.CrashReportState
import net.kdhmedia.mochil.diagnostics.PerformanceDiagnosticsState
import net.kdhmedia.mochil.playback.ReplayGainMode
import net.kdhmedia.mochil.reader.epub.AnnotationPrivacyState
import net.kdhmedia.mochil.reader.epub.AnnotationPrivacyViewModel
import net.kdhmedia.mochil.reader.ReaderHardwareNavigationPreferences
import net.kdhmedia.mochil.update.AppUpdateState
import net.kdhmedia.mochil.ui.kit.LucideIcons
import net.kdhmedia.mochil.ui.kit.MochilCard
import net.kdhmedia.mochil.ui.kit.PageHeader
import net.kdhmedia.mochil.ui.kit.SettingsEntry
import net.kdhmedia.mochil.ui.theme.AppThemeSelection
import net.kdhmedia.mochil.ui.theme.MochilChrome
import net.kdhmedia.mochil.ui.theme.MochilSpace

@Composable
fun DeviceSettingsScreen(
    serverUrl: String,
    serverAddressChanging: Boolean,
    serverAddressMessage: String?,
    onChangeServerAddress: (String) -> Unit,
    themeSelection: AppThemeSelection,
    onThemeSelection: (AppThemeSelection) -> Unit,
    updateState: AppUpdateState,
    onCheckForUpdates: () -> Unit,
    onInstallUpdate: () -> Unit,
    onDisconnectServer: () -> Unit,
    downloadLocation: DownloadLocation,
    onChooseDownloadFolder: () -> Unit,
    onUseAppStorage: () -> Unit,
    replayGainMode: ReplayGainMode,
    onReplayGainMode: (ReplayGainMode) -> Unit,
    crashState: CrashReportState,
    onSendCrashReport: () -> Unit,
    onDiscardCrashReport: () -> Unit,
    diagnosticsState: PerformanceDiagnosticsState,
    onDiagnosticsEnabled: (Boolean) -> Unit,
    onSendDiagnostics: () -> Unit,
    onClearDiagnostics: () -> Unit,
    annotationPrivacyState: AnnotationPrivacyState,
    onExportAnnotations: () -> Unit,
    onDeleteServerAnnotations: (String) -> Unit,
    libraries: List<LibrarySummary>,
    visibleLibraryIds: Set<String>,
    deviceClass: ClientDeviceClass,
    onLibraryVisibility: (LibrarySummary, Boolean) -> Unit,
    showCollections: Boolean,
    showSources: Boolean,
    onDestinationVisibility: (OptionalDestination, Boolean) -> Unit,
) {
    var showDeletion by rememberSaveable { mutableStateOf(false) }
    var showDisconnect by rememberSaveable { mutableStateOf(false) }
    var showServerAddress by rememberSaveable { mutableStateOf(false) }
    var confirmation by rememberSaveable { mutableStateOf("") }
    // Which page is open, by name (null is the hub): a renamed page then reads back as the hub rather than crashing a restore.
    var pageName by rememberSaveable { mutableStateOf<String?>(null) }
    val page = SettingsCategory.fromName(pageName)
    val context = LocalContext.current
    val hardwarePreferences = androidx.compose.runtime.remember(context) { ReaderHardwareNavigationPreferences(context) }
    var volumePageTurn by rememberSaveable { mutableStateOf(hardwarePreferences.volumeButtonsEnabled()) }
    if (showDeletion) AnnotationDeletionDialog(
        confirmation = confirmation,
        deleting = annotationPrivacyState.deleting,
        onConfirmation = { confirmation = it.take(AnnotationPrivacyViewModel.DELETE_CONFIRMATION.length) },
        onDismiss = { showDeletion = false; confirmation = "" },
        onDelete = { onDeleteServerAnnotations(confirmation); showDeletion = false; confirmation = "" },
    )
    if (showDisconnect) AlertDialog(
        onDismissRequest = { showDisconnect = false },
        title = { Text("Disconnect server?") },
        text = { Text("This removes the server connection from this device. Local folders and imported fonts stay available.") },
        confirmButton = { TextButton(onClick = { showDisconnect = false; onDisconnectServer() }, colors = ButtonDefaults.textButtonColors(contentColor = MaterialTheme.colorScheme.error)) { Text("Disconnect") } },
        dismissButton = { TextButton(onClick = { showDisconnect = false }) { Text("Cancel") } },
    )
    if (showServerAddress) ChangeServerAddressDialog(serverUrl, serverAddressChanging, { showServerAddress = false }) { value ->
        onChangeServerAddress(value)
        showServerAddress = false
    }
    // System Back closes a page before the shell sees it; from the hub it falls through to the shell (Library).
    BackHandler(enabled = page != null) { pageName = null }
    if (page == SettingsCategory.Sources) {
        SourceHubScreen(onBack = { pageName = null })
        return
    }
    // One scroll position per page: the hub and the pages share this column, and a page must open at its top, not where the hub was.
    val scroll = rememberSaveable(pageName, saver = ScrollState.Saver) { ScrollState(0) }
    Column(Modifier.fillMaxSize().verticalScroll(scroll).padding(horizontal = MOBILE_CONTENT_GUTTER, vertical = MochilSpace.s8)) {
        if (page == null) {
            Text("Settings", Modifier.padding(vertical = MochilSpace.s12).semantics { heading() }, style = MaterialTheme.typography.headlineSmall, color = MaterialTheme.colorScheme.onSurface)
            Column(verticalArrangement = Arrangement.spacedBy(MochilSpace.s12)) {
                // The mock-up opens on the server: where you are connected and to what. "Paired", not "Connected": nothing here has checked.
                MochilCard(onClick = { pageName = SettingsCategory.Server.name }) {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Column(Modifier.weight(1f).padding(end = MochilSpace.s8)) {
                            Text("Server", style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurface)
                            Text(serverUrl, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant, maxLines = 1, overflow = androidx.compose.ui.text.style.TextOverflow.Ellipsis)
                        }
                        StatusPill("Paired", positive = false)
                        Icon(LucideIcons.ChevronRight, contentDescription = null, Modifier.padding(start = MochilSpace.s8).size(MochilChrome.iconSmall), tint = MaterialTheme.colorScheme.onSurfaceVariant)
                    }
                }
                SettingsCategory.entries.filterNot { it == SettingsCategory.Server }.forEach { entry ->
                    val icon = androidx.compose.runtime.remember(entry) { LucideIcons.withStroke(entry.iconId, 2f) }
                    SettingsEntry(icon, entry.label, entry.description, onClick = { pageName = entry.name })
                }
            }
            Text("Mochil ${BuildConfig.VERSION_NAME}", Modifier.padding(vertical = MochilSpace.s16), style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
        } else {
            PageHeader(page.label, onBack = { pageName = null })
            SettingsSurface(Modifier.padding(top = MochilSpace.s8)) { when (page) {
                SettingsCategory.Server -> {
                    SettingsCard("Connection") {
                        Row(Modifier.fillMaxWidth().padding(horizontal = 8.dp, vertical = 6.dp), verticalAlignment = Alignment.CenterVertically) {
                            Text(serverUrl, Modifier.weight(1f).padding(end = 8.dp), style = MaterialTheme.typography.bodyLarge, maxLines = 2, overflow = androidx.compose.ui.text.style.TextOverflow.Ellipsis)
                            StatusPill("Paired", positive = false)
                        }
                        serverAddressMessage?.let { Text(it, modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp), style = MaterialTheme.typography.bodySmall, color = if (it == "Server address updated.") MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.error) }
                        Row(Modifier.padding(start = 8.dp, top = 4.dp), verticalAlignment = Alignment.CenterVertically) {
                            OutlinedButton(onClick = { showServerAddress = true }, enabled = !serverAddressChanging) { Text("Change address") }
                            TextButton(onClick = { showDisconnect = true }, colors = ButtonDefaults.textButtonColors(contentColor = MaterialTheme.colorScheme.error)) { Text("Disconnect") }
                        }
                    }
                    if (BuildConfig.SELF_UPDATE_ENABLED) UpdateSettings(updateState, onCheckForUpdates, onInstallUpdate)
                }
                SettingsCategory.Appearance -> AppearanceSettings(themeSelection, onThemeSelection)
                SettingsCategory.Libraries -> SettingsCard("Libraries on this device") {
                    LibraryVisibilitySettings(libraries, visibleLibraryIds, deviceClass, onLibraryVisibility, showCollections = showCollections, showSources = showSources, onDestinationVisibility = onDestinationVisibility)
                }
                SettingsCategory.Sources -> Unit // drawn full screen above: the source hub has its own header
                SettingsCategory.About -> AboutContent()
                SettingsCategory.Downloads -> DownloadSettings(downloadLocation, onChooseDownloadFolder, onUseAppStorage)
                SettingsCategory.Reading -> {
                    ReaderDefaultSettings()
                    ReaderHardwareSettings(volumePageTurn) { enabled -> volumePageTurn = enabled; hardwarePreferences.setVolumeButtonsEnabled(enabled) }
                    ReadAloudSettings()
                }
                SettingsCategory.Playback -> {
                    SettingsCard(null) { PlaybackQualitySettings() }
                    ReplayGainSettings(replayGainMode, onReplayGainMode)
                    SettingsCard("Video subtitles") { Column(Modifier.fillMaxWidth().padding(horizontal = 8.dp)) { SubtitleAppearanceSettings() } }
                }
                SettingsCategory.Privacy -> {
                    AnnotationSettings(annotationPrivacyState, onExportAnnotations) { confirmation = ""; showDeletion = true }
                    PerformanceSettings(diagnosticsState, onDiagnosticsEnabled, onSendDiagnostics, onClearDiagnostics)
                    CrashSettings(crashState, onSendCrashReport, onDiscardCrashReport)
                }
            } }
        }
    }
}

@Composable
private fun ChangeServerAddressDialog(current: String, changing: Boolean, onDismiss: () -> Unit, onSave: (String) -> Unit) {
    var value by rememberSaveable(current) { mutableStateOf(current) }
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("Change server address") },
        text = { Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
            Text("Use this when the same Mochil server moves to a new HTTPS address. Your existing device token is verified before anything changes.", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
            OutlinedTextField(value, { value = it.take(2048) }, label = { Text("Mochil address") }, placeholder = { Text("https://media.example.com") }, singleLine = true)
        } },
        confirmButton = { TextButton(onClick = { onSave(value) }, enabled = !changing && value.trim().trimEnd('/') != current.trimEnd('/') && value.isNotBlank()) { Text(if (changing) "Checking…" else "Verify and change") } },
        dismissButton = { TextButton(onClick = onDismiss) { Text("Cancel") } },
    )
}

@Composable
private fun UpdateSettings(state: AppUpdateState, onCheck: () -> Unit, onInstall: () -> Unit) {
    SettingsCard("App updates") {
    Text("Installed version ${BuildConfig.VERSION_NAME}", modifier = Modifier.padding(start = 8.dp, top = 4.dp), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
    if (state.checking || state.downloading) LinearProgressIndicator(Modifier.fillMaxWidth().padding(horizontal = 8.dp, vertical = 12.dp))
    state.message?.let { Text(it, modifier = Modifier.padding(horizontal = 8.dp, vertical = 8.dp), style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant) }
    state.releaseNotes?.let { notes -> Column(Modifier.padding(horizontal = 8.dp, vertical = 6.dp)) {
        ReleaseNoteSection("New", notes.newFeatures); ReleaseNoteSection("Fixed", notes.fixes); ReleaseNoteSection("Performance", notes.performance); ReleaseNoteSection("Changes", notes.changes)
    } }
    Button(onClick = if (state.downloadPath != null) onInstall else onCheck, enabled = !state.checking && !state.downloading, modifier = Modifier.padding(start = 8.dp, top = 8.dp)) {
        Text(if (state.downloadPath != null) "Download update" else "Check for updates")
    }
    }
}

@Composable
private fun DownloadSettings(location: DownloadLocation, onChoose: () -> Unit, onPrivate: () -> Unit) {
    SettingsCard("Download location") {
    Text(location.label, modifier = Modifier.padding(start = 8.dp, top = 4.dp), style = MaterialTheme.typography.bodyLarge)
    Text(
        if (location.treeUri == null) {
            "Downloads are kept in private app storage for fast offline opening. Removing the app removes these files."
        } else {
            "Downloads are kept in app storage and copied here for direct access. Choose this same folder after reinstalling to recover indexed books, music, podcasts, and audiobooks."
        },
        modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant,
    )
    location.message?.let { Text(
        it,
        Modifier.padding(horizontal = 8.dp, vertical = 4.dp).semantics { liveRegion = LiveRegionMode.Polite },
        style = MaterialTheme.typography.bodySmall,
        color = if (location.messageIsError) MaterialTheme.colorScheme.error else MaterialTheme.colorScheme.onSurfaceVariant,
    ) }
    Row(Modifier.padding(start = 8.dp, top = 4.dp)) {
        Button(onClick = onChoose) { Text(if (location.treeUri == null) "Choose folder" else "Change folder") }
        if (location.treeUri != null) TextButton(onClick = onPrivate) { Text("Use app storage") }
    }
    }
}

@Composable
private fun AnnotationSettings(state: AnnotationPrivacyState, onExport: () -> Unit, onDelete: () -> Unit) {
    SettingsCard("Annotations") {
    Text("Export a private JSON backup of synchronized highlights and notes, or remove the server copy. Highlights stored in reading apps remain on those devices and may synchronize again.", modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
    state.message?.let { Text(it, Modifier.padding(horizontal = 8.dp, vertical = 4.dp).semantics { liveRegion = LiveRegionMode.Polite }, style = MaterialTheme.typography.bodySmall, color = if (state.isError) MaterialTheme.colorScheme.error else MaterialTheme.colorScheme.onSurfaceVariant) }
    Row(Modifier.padding(start = 8.dp, top = 4.dp)) {
        Button(onClick = onExport, enabled = !state.exporting && !state.deleting) { Text(if (state.exporting) "Exporting…" else "Export backup") }
        TextButton(onClick = onDelete, enabled = !state.exporting && !state.deleting, colors = ButtonDefaults.textButtonColors(contentColor = MaterialTheme.colorScheme.error)) { Text("Delete server copy") }
    }
    }
}

@Composable
private fun ReplayGainSettings(mode: ReplayGainMode, onMode: (ReplayGainMode) -> Unit) {
    SettingsCard("ReplayGain") {
    Text("Normalize music volume using embedded or analyzed loudness metadata. Peak values prevent clipping when available.", modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
    Row(Modifier.fillMaxWidth().padding(horizontal = 8.dp, vertical = 4.dp), horizontalArrangement = Arrangement.spacedBy(6.dp)) {
        ReplayGainMode.entries.forEach { choice -> FilterChip(selected = mode == choice, onClick = { onMode(choice) }, label = { Text(choice.name) }) }
    }
    }
}

@Composable
private fun PerformanceSettings(state: PerformanceDiagnosticsState, onEnabled: (Boolean) -> Unit, onSend: () -> Unit, onClear: () -> Unit) {
    SettingsCard("Performance diagnostics") {
    Row(Modifier.fillMaxWidth().padding(horizontal = 8.dp, vertical = 6.dp)) {
        Column(Modifier.weight(1f).padding(end = 12.dp)) {
            Text("Help improve Mochil", style = MaterialTheme.typography.bodyLarge)
            Text("When enabled, the app sends a small batch every 10 minutes while open. It includes timings, event names, outcomes, app/device versions, and no credentials, addresses, titles, searches, paths, or error messages.", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
        }
        Switch(checked = state.enabled, onCheckedChange = onEnabled)
    }
    Text("${state.pending} ${if (state.pending == 1) "event" else "events"} waiting on this device", modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
    state.message?.let { Text(it, Modifier.padding(horizontal = 8.dp, vertical = 4.dp), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant) }
    Row(Modifier.padding(start = 8.dp, top = 4.dp)) {
        Button(onClick = onSend, enabled = state.pending > 0 && !state.sending) { Text(if (state.sending) "Sending…" else "Send now") }
        TextButton(onClick = onClear, enabled = state.pending > 0 && !state.sending) { Text("Clear") }
    }
    }
}

@Composable
private fun CrashSettings(state: CrashReportState, onSend: () -> Unit, onDiscard: () -> Unit) {
    if (!state.pending && state.message == null) return
    SettingsCard("Diagnostics") {
    Text(state.message ?: "A bounded crash report is saved on this device. It contains app/device versions and stack frames, but no credentials, server address, book titles, or media paths.", modifier = Modifier.padding(horizontal = 8.dp, vertical = 6.dp), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
    if (state.pending) Row(Modifier.padding(start = 8.dp, top = 4.dp)) {
        Button(onClick = onSend, enabled = !state.sending) { Text(if (state.sending) "Sending…" else "Send crash report") }
        TextButton(onClick = onDiscard, enabled = !state.sending) { Text("Delete") }
    }
    }
}

@Composable
private fun AnnotationDeletionDialog(confirmation: String, deleting: Boolean, onConfirmation: (String) -> Unit, onDismiss: () -> Unit, onDelete: () -> Unit) {
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("Delete server annotations?") },
        text = { Column {
            Text("This permanently removes every synchronized annotation from this Mochil account. Reading apps keep their local copies and can upload them again.")
            Text("Type ${AnnotationPrivacyViewModel.DELETE_CONFIRMATION} to continue.", modifier = Modifier.padding(top = 12.dp), style = MaterialTheme.typography.bodySmall)
            OutlinedTextField(value = confirmation, onValueChange = onConfirmation, label = { Text("Confirmation") }, singleLine = true, modifier = Modifier.fillMaxWidth().padding(top = 8.dp))
        } },
        dismissButton = { TextButton(onClick = onDismiss) { Text("Cancel") } },
        confirmButton = { TextButton(onClick = onDelete, enabled = confirmation == AnnotationPrivacyViewModel.DELETE_CONFIRMATION && !deleting, colors = ButtonDefaults.textButtonColors(contentColor = MaterialTheme.colorScheme.error)) { Text(if (deleting) "Deleting…" else "Delete server copy") } },
    )
}

@Composable
private fun ReleaseNoteSection(title: String, items: List<String>) {
    if (items.isEmpty()) return
    Text(title, modifier = Modifier.padding(top = 8.dp, bottom = 2.dp), style = MaterialTheme.typography.labelLarge, fontWeight = FontWeight.SemiBold)
    items.forEach { note -> Row(Modifier.fillMaxWidth().padding(vertical = 2.dp)) {
        Text("•", modifier = Modifier.padding(end = 8.dp), color = MaterialTheme.colorScheme.primary)
        Text(note, modifier = Modifier.weight(1f), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
    } }
}
