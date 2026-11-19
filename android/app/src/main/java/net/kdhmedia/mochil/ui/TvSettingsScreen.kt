package net.kdhmedia.mochil.ui

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.border
import androidx.compose.foundation.focusGroup
import androidx.compose.foundation.focusable
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.Switch
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.focus.onFocusChanged
import androidx.compose.ui.focus.focusRestorer
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import net.kdhmedia.mochil.BuildConfig
import net.kdhmedia.mochil.update.AppUpdateState
import net.kdhmedia.mochil.update.ReleaseNotes
import net.kdhmedia.mochil.ui.theme.AppPalette
import net.kdhmedia.mochil.ui.theme.AppThemeMode
import net.kdhmedia.mochil.ui.theme.AppThemeSelection
import net.kdhmedia.mochil.ui.theme.TvSettingsTypography
import net.kdhmedia.mochil.diagnostics.CrashReportState
import net.kdhmedia.mochil.diagnostics.PerformanceDiagnosticsState
import net.kdhmedia.mochil.catalogue.ClientDeviceClass
import net.kdhmedia.mochil.catalogue.LibrarySummary

@Composable
fun TvSettingsScreen(
    serverUrl: String,
    update: AppUpdateState,
    themeSelection: AppThemeSelection,
    onThemeSelection: (AppThemeSelection) -> Unit,
    crashState: CrashReportState,
    diagnosticsState: PerformanceDiagnosticsState,
    onDiagnosticsEnabled: (Boolean) -> Unit,
    onSendDiagnostics: () -> Unit,
    onClearDiagnostics: () -> Unit,
    onSendCrashReport: () -> Unit,
    onDiscardCrashReport: () -> Unit,
    onBack: () -> Unit,
    onCheckUpdate: () -> Unit,
    onInstallUpdate: () -> Unit,
    onDisconnect: () -> Unit,
    libraries: List<LibrarySummary>,
    visibleLibraryIds: Set<String>,
    deviceClass: ClientDeviceClass,
    onLibraryVisibility: (LibrarySummary, Boolean) -> Unit,
) {
    var confirmDisconnect by remember { mutableStateOf(false) }
    var category by remember { mutableStateOf(TvSettingsCategory.Appearance) }
    val readiness = rememberTvReadinessState()
    val contentScroll = rememberScrollState()
    LaunchedEffect(category) { contentScroll.scrollTo(0) }
    // A settings list is read up close, not from the sofa: the denser scale keeps the category column, the group headings
    // and a full changelog from reading as magnified. The confirmation sits inside it too, so it matches the screen behind.
    MaterialTheme(typography = TvSettingsTypography) {
        Column(
            Modifier.fillMaxSize().observeTvReadiness(readiness).padding(start = 28.dp, end = 32.dp, top = 16.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Column(Modifier.padding(start = 10.dp)) {
                    Text("Settings", style = MaterialTheme.typography.headlineSmall, fontWeight = FontWeight.SemiBold)
                }
            }
            Row(Modifier.fillMaxWidth().weight(1f), horizontalArrangement = Arrangement.spacedBy(20.dp)) {
                Column(Modifier.width(210.dp)) { TvSettingsCategoryList(category, BuildConfig.SELF_UPDATE_ENABLED) { category = it } }
                Column(
                    Modifier.weight(1f).verticalScroll(contentScroll).focusRestorer().focusGroup().padding(end = 12.dp, bottom = 32.dp),
                    verticalArrangement = Arrangement.spacedBy(16.dp),
                ) {
                    if (category == TvSettingsCategory.Appearance) TvSettingsGroup("Appearance") {
                        Text("Theme family", color = MaterialTheme.colorScheme.onSurfaceVariant)
                        TvSettingsChoiceRail(
                            choices = AppPalette.entries,
                            selected = { themeSelection.palette == it },
                            label = AppPalette::label,
                            key = AppPalette::name,
                            onSelect = { onThemeSelection(themeSelection.copy(palette = it)) },
                            modifier = Modifier.fillMaxWidth(),
                        )
                        Text("Brightness", color = MaterialTheme.colorScheme.onSurfaceVariant)
                        TvSettingsChoiceRail(
                            choices = AppThemeMode.entries,
                            selected = { themeSelection.mode == it },
                            label = AppThemeMode::label,
                            key = AppThemeMode::name,
                            onSelect = { onThemeSelection(themeSelection.copy(mode = it)) },
                            modifier = Modifier.fillMaxWidth(),
                        )
                        SubtitleAppearanceSettings(television = true)
                    }
                    if (category == TvSettingsCategory.Libraries) TvSettingsGroup("Libraries") {
                        val order = LocalTvLibraryOrder.current
                        LibraryVisibilitySettings(libraries.inTvOrder(), visibleLibraryIds, deviceClass, onLibraryVisibility, television = true, onMove = order?.let { it::move }?.let { move -> { library, delta -> move(library.id, delta, libraries) } })
                    }
                    if (category == TvSettingsCategory.Playback) TvSettingsGroup("Playback") {
                        PlaybackQualitySettings(television = true)
                    }
                    if (BuildConfig.SELF_UPDATE_ENABLED && category == TvSettingsCategory.Updates) TvSettingsGroup("App update") {
                        Text("Installed version ${BuildConfig.VERSION_NAME}", style = MaterialTheme.typography.titleMedium)
                        update.message?.let { Text(it, color = MaterialTheme.colorScheme.onSurfaceVariant) }
                        update.releaseNotes?.let { notes -> TvReleaseNotes(notes) }
                        Row(horizontalArrangement = Arrangement.spacedBy(16.dp), verticalAlignment = Alignment.CenterVertically) {
                            // A disabled button cannot hold focus, which threw the remote back to the menu mid-check; stay focusable and ignore presses instead.
                            val installFocus = remember { androidx.compose.ui.focus.FocusRequester() }
                            TvButton(if (update.checking) "Checking…" else "Check for updates", { if (!update.checking && !update.downloading) onCheckUpdate() })
                            if (update.availableVersion != null) {
                                LaunchedEffect(update.availableVersion) { runCatching { installFocus.requestFocus() } }
                                TvButton(if (update.downloading) "Downloading…" else stringResource(net.kdhmedia.mochil.R.string.tv_update_install_action, update.availableVersion), { if (!update.downloading) onInstallUpdate() }, Modifier.focusRequester(installFocus), primary = true)
                            }
                            if (update.checking || update.downloading) CircularProgressIndicator()
                        }
                        Text(stringResource(net.kdhmedia.mochil.R.string.tv_update_install_help), color = MaterialTheme.colorScheme.onSurfaceVariant)
                    }
                    if (category == TvSettingsCategory.Connection) TvSettingsGroup("Connection") {
                        Text(serverUrl, style = MaterialTheme.typography.titleMedium)
                        Text("The server address and account stay encrypted on this device.", color = MaterialTheme.colorScheme.onSurfaceVariant)
                        TvButton("Disconnect this TV", { confirmDisconnect = true })
                    }
                    if (category == TvSettingsCategory.Connection) TvSettingsGroup("TV readiness") {
                        TvReadinessCheck(readiness)
                    }
                    if (category == TvSettingsCategory.Diagnostics) TvSettingsGroup("Diagnostics") {
                        Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                            Column(Modifier.weight(1f)) {
                                Text("Help improve TV performance", style = MaterialTheme.typography.titleMedium)
                                Text("Collects bounded timings, outcomes, app/device versions, and no credentials, addresses, titles, searches, or media paths.", color = MaterialTheme.colorScheme.onSurfaceVariant)
                            }
                            Switch(checked = diagnosticsState.enabled, onCheckedChange = onDiagnosticsEnabled, modifier = Modifier.tvFocusBorder())
                        }
                        Text("${diagnosticsState.pending} ${if (diagnosticsState.pending == 1) "event" else "events"} waiting on this TV", color = MaterialTheme.colorScheme.onSurfaceVariant)
                        diagnosticsState.message?.let { Text(it, color = MaterialTheme.colorScheme.onSurfaceVariant) }
                        Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                            TvButton(if (diagnosticsState.sending) "Sending…" else "Send performance logs", onSendDiagnostics, enabled = diagnosticsState.pending > 0 && !diagnosticsState.sending)
                            TvButton("Clear", onClearDiagnostics, enabled = diagnosticsState.pending > 0 && !diagnosticsState.sending)
                        }
                        if (crashState.pending || crashState.message != null) {
                            Text(crashState.message ?: "A bounded crash report is saved on this TV.", color = MaterialTheme.colorScheme.onSurfaceVariant)
                            if (crashState.pending) Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                                TvButton(if (crashState.sending) "Sending…" else "Send crash report", onSendCrashReport, enabled = !crashState.sending)
                                TvButton("Delete", onDiscardCrashReport, enabled = !crashState.sending)
                            }
                        }
                    }
                }
            }
        }
        if (confirmDisconnect) AlertDialog(
            onDismissRequest = { confirmDisconnect = false },
            title = { Text("Disconnect this TV?") },
            text = { Text("This removes the encrypted server connection from this TV. You will need the server address, username, and password to connect again.") },
            confirmButton = {
                Button(onClick = { confirmDisconnect = false; onDisconnect() }, modifier = Modifier.tvFocusBorder()) { Text("Disconnect TV") }
            },
            dismissButton = {
                OutlinedButton(onClick = { confirmDisconnect = false }, modifier = Modifier.tvFocusBorder()) { Text("Keep connected") }
            },
        )
    }
}

/**
 * The changelog as its own bounded panel instead of a column of bullets in the page flow: a long release-note list used
 * to fill the screen and push "Check for updates" below it. It is one focus stop, and the remote scrolls it.
 */
@Composable
internal fun TvReleaseNotes(notes: ReleaseNotes) {
    val sections = listOf(
        "New" to notes.newFeatures,
        "Fixed" to notes.fixes,
        "Performance" to notes.performance,
        "Changes" to notes.changes,
    ).filter { it.second.isNotEmpty() }
    if (sections.isEmpty()) return
    val scroll = rememberScrollState()
    var focused by remember { mutableStateOf(false) }
    Surface(
        shape = RoundedCornerShape(14.dp),
        color = MaterialTheme.colorScheme.surfaceVariant,
        modifier = Modifier.fillMaxWidth().heightIn(max = 232.dp)
            .onFocusChanged { focused = it.isFocused }
            .testTag("tv-release-notes")
            .tvFocusVisual(focused, RoundedCornerShape(14.dp), focusedScale = 1f)
            .focusable()
            .tvScrollKeys(scroll),
    ) {
        Column(
            Modifier.fillMaxWidth().verticalScroll(scroll).padding(horizontal = 16.dp, vertical = 12.dp),
            verticalArrangement = Arrangement.spacedBy(3.dp),
        ) {
            sections.forEach { (heading, items) ->
                Text(
                    heading,
                    style = MaterialTheme.typography.labelLarge,
                    fontWeight = FontWeight.SemiBold,
                    color = MaterialTheme.colorScheme.primary,
                    modifier = Modifier.padding(top = 4.dp),
                )
                items.forEach { note ->
                    Row(Modifier.fillMaxWidth()) {
                        Text("•", modifier = Modifier.padding(end = 8.dp), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.primary)
                        Text(note, modifier = Modifier.weight(1f), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                    }
                }
            }
        }
    }
}

@Composable
private fun TvSettingsGroup(title: String, content: @Composable ColumnScope.() -> Unit) {
    Column(Modifier.fillMaxWidth(), verticalArrangement = Arrangement.spacedBy(10.dp)) {
        Text(title, style = MaterialTheme.typography.titleLarge, fontWeight = FontWeight.SemiBold)
        content()
    }
}
