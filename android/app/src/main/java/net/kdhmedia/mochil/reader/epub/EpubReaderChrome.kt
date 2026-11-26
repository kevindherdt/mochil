package net.kdhmedia.mochil.reader.epub

import net.kdhmedia.mochil.ui.MotionVisibility
import androidx.compose.foundation.clickable
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.automirrored.outlined.MenuBook
import androidx.compose.material.icons.filled.Search
import androidx.compose.material.icons.filled.Bookmark
import androidx.compose.material.icons.outlined.BookmarkBorder
import androidx.compose.material.icons.outlined.CloudDone
import androidx.compose.material.icons.outlined.CloudOff
import androidx.compose.material.icons.outlined.CloudSync
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Surface
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.OutlinedTextFieldDefaults
import androidx.compose.material3.PrimaryScrollableTabRow
import androidx.compose.material3.Tab
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.unit.dp
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.ui.platform.LocalFocusManager
import androidx.compose.ui.semantics.LiveRegionMode
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.liveRegion
import androidx.compose.ui.semantics.semantics
import org.readium.r2.navigator.preferences.Theme
import org.readium.r2.shared.publication.Link
import java.text.DateFormat
import java.util.Date

data class EpubChapter(val title: String, val depth: Int, val link: Link)
private enum class ReaderPanel(val label: String) { Contents("Contents"), Bookmarks("Bookmarks"), Highlights("Highlights"), Statistics("Stats") }
/**
 * The web reader's grouping, as far as the phone has settings for it: everything that sets the text (Typography), then
 * everything that sets the page around it (Display & theme). The web's third tab, Narration, has no settings of its own here:
 * the read-aloud controls are the bar above this sheet, always one tap away.
 */
private enum class SettingsSection(val label: String) { Typography("Typography"), Narration("Narration"), Display("Display & theme") }

@Composable
@OptIn(ExperimentalMaterial3Api::class)
internal fun EpubReaderChrome(
    visible: Boolean,
    title: String,
    progression: Double?,
    readingStatistics: ReadingStatistics,
    preferences: ReaderPreferences,
    customFonts: List<CustomReaderFont>,
    onPreferences: ((ReaderPreferences) -> ReaderPreferences) -> Unit,
    chapters: List<EpubChapter>,
    onChapter: (EpubChapter) -> Unit,
    searchState: EpubSearchState,
    onSearch: (String) -> Unit,
    onSearchResult: (org.readium.r2.shared.publication.Locator) -> Unit,
    bookmarks: List<EpubBookmark>,
    annotations: List<SyncedAnnotation>,
    annotationConflicts: List<AnnotationConflict>,
    currentBookmarked: Boolean,
    onToggleBookmark: () -> Unit,
    onBookmark: (EpubBookmark) -> Unit,
    onAnnotation: (SyncedAnnotation) -> Unit,
    onEditAnnotationNote: (SyncedAnnotation, String) -> Unit,
    onDeleteAnnotation: (SyncedAnnotation) -> Unit,
    onKeepLocalAnnotation: (AnnotationConflict) -> Unit,
    onUseRemoteAnnotation: (AnnotationConflict) -> Unit,
    annotationNotesAvailable: Boolean,
    annotationNotesExporting: Boolean,
    onExportAnnotationNotes: () -> Unit,
    readingSyncStatus: ReaderSyncStatus,
    syncMessage: String?,
    syncMessageIsError: Boolean,
    onDismiss: () -> Unit,
    onClose: () -> Unit,
    narrationControls: @Composable () -> Unit = {},
    /** The narration settings for the sheet's Narration tab; shown only when [narrationAvailable]. */
    narrationSettings: @Composable () -> Unit = {},
    narrationAvailable: Boolean = false,
) {
    var settingsExpanded by remember { mutableStateOf(false) }
    var fontsExpanded by remember { mutableStateOf(false) }
    var contentsExpanded by remember { mutableStateOf(false) }
    var readerPanel by remember { mutableStateOf(ReaderPanel.Contents) }
    // Layout is the default tab so the paging mode (Paged/Scroll) stays immediately reachable,
    // matching the long-standing one-tap behaviour the existing settings tests rely on.
    var settingsSection by remember { mutableStateOf(SettingsSection.Typography) }
    var searchExpanded by remember { mutableStateOf(false) }
    var searchDraft by remember { mutableStateOf("") }
    val focusManager = LocalFocusManager.current
    val chromeContent = preferences.theme.chromeContent
    Box(
        Modifier.fillMaxSize().clickable(
            interactionSource = remember { MutableInteractionSource() },
            indication = null,
            onClickLabel = if (visible) "Hide reader controls" else "Show reader controls",
            role = Role.Button,
            onClick = onDismiss,
        ),
    ) {
        MotionVisibility(visible, modifier = Modifier.align(Alignment.TopCenter)) {
            Surface(color = preferences.theme.chromeSurface, contentColor = preferences.theme.chromeContent) {
            Column(Modifier.fillMaxWidth().statusBarsPadding()) {
                Row(Modifier.fillMaxWidth().padding(horizontal = 4.dp, vertical = 4.dp), verticalAlignment = Alignment.CenterVertically) {
                    IconButton(onClick = onClose) { Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Close reader") }
                    Text(title, modifier = Modifier.weight(1f), style = MaterialTheme.typography.titleSmall, maxLines = 1, overflow = TextOverflow.Ellipsis)
                    IconButton(onClick = { searchExpanded = !searchExpanded }) { Icon(Icons.Default.Search, contentDescription = "Search this book") }
                    if (chapters.isNotEmpty() || bookmarks.isNotEmpty() || annotations.isNotEmpty() || annotationConflicts.isNotEmpty()) {
                        IconButton(onClick = { contentsExpanded = true }) { Icon(Icons.AutoMirrored.Outlined.MenuBook, contentDescription = "Reading navigation") }
                    }
                    IconButton(onClick = onToggleBookmark) {
                        Icon(if (currentBookmarked) Icons.Default.Bookmark else Icons.Outlined.BookmarkBorder, contentDescription = if (currentBookmarked) "Remove bookmark" else "Add bookmark")
                    }
                    ReaderChromeTextButton("Aa", chromeContent, Modifier.semantics { contentDescription = "Reader settings" }) { settingsExpanded = !settingsExpanded }
                }
                narrationControls()
                ReaderSyncIndicator(readingSyncStatus, chromeContent)
                MotionVisibility(searchExpanded) {
                    OutlinedTextField(
                        value = searchDraft,
                        onValueChange = { searchDraft = it },
                        modifier = Modifier.fillMaxWidth().padding(horizontal = 12.dp, vertical = 6.dp),
                        singleLine = true,
                        label = { Text("Search this book") },
                        trailingIcon = {
                            IconButton(onClick = { if (searchDraft.isNotBlank()) onSearch(searchDraft) }) {
                                Icon(Icons.Default.Search, contentDescription = "Search")
                            }
                        },
                        keyboardOptions = KeyboardOptions(imeAction = ImeAction.Search),
                        keyboardActions = KeyboardActions(onSearch = {
                            if (searchDraft.isNotBlank()) onSearch(searchDraft)
                            focusManager.clearFocus()
                        }),
                        colors = OutlinedTextFieldDefaults.colors(
                            focusedTextColor = chromeContent,
                            unfocusedTextColor = chromeContent,
                            focusedLabelColor = chromeContent,
                            unfocusedLabelColor = chromeContent.copy(alpha = 0.78f),
                            focusedBorderColor = chromeContent,
                            unfocusedBorderColor = chromeContent.copy(alpha = 0.58f),
                            cursorColor = chromeContent,
                        ),
                    )
                }
                syncMessage?.let {
                    Text(
                        it,
                        modifier = Modifier.padding(horizontal = 16.dp, vertical = 6.dp).semantics { liveRegion = LiveRegionMode.Polite },
                        style = MaterialTheme.typography.bodySmall,
                        color = if (syncMessageIsError) Color(0xFFFFB4AB) else chromeContent.copy(alpha = 0.78f),
                    )
                }
                MotionVisibility(settingsExpanded) {
                    Column(Modifier.fillMaxWidth()) {
                        // Narration only when this book can be read aloud: an empty tab is a dead one.
                        val sections = SettingsSection.entries.filter { it != SettingsSection.Narration || narrationAvailable }
                        PrimaryScrollableTabRow(selectedTabIndex = sections.indexOf(settingsSection).coerceAtLeast(0), edgePadding = 12.dp) {
                            sections.forEach { section ->
                                Tab(selected = settingsSection == section, onClick = { settingsSection = section }, text = { Text(section.label, maxLines = 1) })
                            }
                        }
                        Column(Modifier.fillMaxWidth().padding(horizontal = 12.dp, vertical = 6.dp)) {
                            when (settingsSection) {
                                SettingsSection.Typography -> {
                                    Row(verticalAlignment = Alignment.CenterVertically) {
                                        Text("Font size", modifier = Modifier.weight(1f), color = chromeContent)
                                        ReaderChromeTextButton(
                                            "A−", chromeContent,
                                            Modifier.semantics { contentDescription = "Decrease font size" },
                                        ) { onPreferences { current -> current.copy(fontSize = (current.fontSize - 0.1).coerceAtLeast(0.7)) } }
                                        ReaderChromeTextButton(
                                            "A+", chromeContent,
                                            Modifier.semantics { contentDescription = "Increase font size" },
                                        ) { onPreferences { current -> current.copy(fontSize = (current.fontSize + 0.1).coerceAtMost(2.0)) } }
                                    }
                                    Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                                        Text("Font", modifier = Modifier.weight(1f), color = chromeContent)
                                        ReaderChromeTextButton(
                                            if (preferences.font == ReaderFont.Custom) customFonts.firstOrNull { it.id == preferences.customFontId }?.name ?: "Custom" else preferences.font.shortLabel,
                                            chromeContent,
                                        ) { fontsExpanded = true }
                                    }
                                }
                                SettingsSection.Narration -> narrationSettings()
                            SettingsSection.Display -> {
                                    Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                                        Text("Page turning", modifier = Modifier.weight(1f), color = chromeContent)
                                        ReaderChromeTextButton(if (preferences.scroll) "Scroll" else "Paged", chromeContent) { onPreferences { current -> current.copy(scroll = !current.scroll) } }
                                    }
                                    Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                                        Text("Tap zones", modifier = Modifier.weight(1f), color = chromeContent)
                                        ReaderChromeTextButton("Tap zones: ${preferences.tapZones.label}", chromeContent) {
                                            onPreferences { current -> current.copy(tapZones = current.tapZones.next()) }
                                        }
                                    }
                                    Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                                        Text("Theme", modifier = Modifier.weight(1f), color = chromeContent)
                                        ReaderChromeTextButton(preferences.theme.label, chromeContent) { onPreferences { current -> current.copy(theme = current.theme.next()) } }
                                    }
                                    Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                                        Text("Sharper text", modifier = Modifier.weight(1f), color = chromeContent)
                                        ReaderChromeTextButton("Sharper text: ${preferences.sharpness.label}", chromeContent) {
                                            onPreferences { current -> current.copy(sharpness = current.sharpness.next()) }
                                        }
                                    }
                                    Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                                        Text("Reading progress", modifier = Modifier.weight(1f), color = chromeContent)
                                        ReaderChromeTextButton("Reading progress: ${preferences.progress.label}", chromeContent) {
                                            onPreferences { current -> current.copy(progress = current.progress.next()) }
                                        }
                                    }
                                    Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                                        Text("Status icons", modifier = Modifier.weight(1f), color = chromeContent)
                                        ReaderChromeTextButton(if (preferences.showSystemStatusBar) "Status icons: on" else "Status icons: off", chromeContent) {
                                            onPreferences { current -> current.copy(showSystemStatusBar = !current.showSystemStatusBar) }
                                        }
                                    }
                                }
                            }
                        }
                    }
                }
            }
            }
        }
        MotionVisibility(visible && progression != null && preferences.progress != ReadingProgress.Bar, modifier = Modifier.align(Alignment.BottomCenter)) {
            ReaderProgressLabel(progression ?: 0.0, readingStatistics, preferences.progress == ReadingProgress.Time)
        }
    }
    if (fontsExpanded) {
        ModalBottomSheet(onDismissRequest = { fontsExpanded = false }) {
            Text("Reader font", modifier = Modifier.padding(horizontal = 20.dp, vertical = 8.dp), style = MaterialTheme.typography.titleLarge)
            LazyColumn(Modifier.fillMaxWidth().navigationBarsPadding()) {
                items(ReaderFont.entries.filterNot { it == ReaderFont.Custom }, key = { "built-in:${it.name}" }) { font ->
                    TextButton(
                        onClick = { fontsExpanded = false; onPreferences { current -> current.copy(font = font, customFontId = null) } },
                        modifier = Modifier.fillMaxWidth().padding(horizontal = 8.dp),
                    ) { Text(if (preferences.font == font) "✓  ${font.label}" else font.label, modifier = Modifier.fillMaxWidth()) }
                }
                items(customFonts, key = { "custom:${it.id}" }) { font ->
                    val selected = preferences.font == ReaderFont.Custom && preferences.customFontId == font.id
                    TextButton(
                        onClick = { fontsExpanded = false; onPreferences { current -> current.copy(font = ReaderFont.Custom, customFontId = font.id) } },
                        modifier = Modifier.fillMaxWidth().padding(horizontal = 8.dp),
                    ) { Text(if (selected) "✓  ${font.name}" else font.name, modifier = Modifier.fillMaxWidth()) }
                }
            }
        }
    }
    if (contentsExpanded) {
        ModalBottomSheet(onDismissRequest = { contentsExpanded = false }) {
            Text("Reading navigation", modifier = Modifier.padding(horizontal = 20.dp, vertical = 8.dp), style = MaterialTheme.typography.titleLarge)
            PrimaryScrollableTabRow(selectedTabIndex = readerPanel.ordinal, edgePadding = 0.dp) {
                ReaderPanel.entries.forEach { panel -> Tab(selected = readerPanel == panel, onClick = { readerPanel = panel }, text = { Text(panel.label, maxLines = 1) }) }
            }
            LazyColumn(Modifier.fillMaxWidth().navigationBarsPadding().imePadding()) {
                when (readerPanel) {
                    ReaderPanel.Contents -> {
                        if (chapters.isEmpty()) item { ReaderPanelEmpty("This book has no table of contents.") }
                        items(chapters) { chapter ->
                            TextButton(onClick = { contentsExpanded = false; onChapter(chapter) }, modifier = Modifier.fillMaxWidth().padding(start = (12 * chapter.depth).dp)) {
                                Text(chapter.title, modifier = Modifier.fillMaxWidth(), maxLines = 2, overflow = TextOverflow.Ellipsis)
                            }
                        }
                    }
                    ReaderPanel.Bookmarks -> {
                        if (bookmarks.isEmpty()) item { ReaderPanelEmpty("No bookmarks yet.") }
                        items(bookmarks) { bookmark ->
                        val label = bookmark.locator.text.highlight ?: bookmark.locator.title ?: "Saved position"
                        TextButton(
                            onClick = {
                                contentsExpanded = false
                                onBookmark(bookmark)
                            },
                            modifier = Modifier.fillMaxWidth().padding(horizontal = 8.dp),
                        ) { Text(label, modifier = Modifier.fillMaxWidth(), maxLines = 2, overflow = TextOverflow.Ellipsis) }
                        }
                    }
                    ReaderPanel.Highlights -> {
                        if (annotations.isEmpty() && annotationConflicts.isEmpty()) item { ReaderPanelEmpty("No highlights yet.") }
                        epubAnnotationPanelItems(
                            annotations = annotations,
                            conflicts = annotationConflicts,
                            onOpen = { annotation -> contentsExpanded = false; onAnnotation(annotation) },
                            onEditNote = onEditAnnotationNote,
                            onDelete = onDeleteAnnotation,
                            onKeepLocal = onKeepLocalAnnotation,
                            onUseRemote = onUseRemoteAnnotation,
                            exportAvailable = annotationNotesAvailable,
                            exporting = annotationNotesExporting,
                            onExport = onExportAnnotationNotes,
                        )
                    }
                    ReaderPanel.Statistics -> item {
                        ReadingStatisticsPanel(readingStatistics)
                    }
                }
            }
        }
    }
    if (searchState.query.isNotBlank()) {
        ModalBottomSheet(onDismissRequest = { onSearch("") }) {
            Text(
                if (searchState.searching) "Searching…" else "Results for “${searchState.query}”",
                modifier = Modifier.padding(horizontal = 20.dp, vertical = 8.dp),
                style = MaterialTheme.typography.titleLarge,
            )
            searchState.error?.let { Text(it, modifier = Modifier.padding(20.dp), color = MaterialTheme.colorScheme.error) }
            if (!searchState.searching && searchState.error == null && searchState.results.isEmpty()) {
                Text("No matches found.", modifier = Modifier.padding(20.dp), color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
            LazyColumn(Modifier.fillMaxWidth().navigationBarsPadding()) {
                items(searchState.results) { locator ->
                    val excerpt = locator.text.highlight ?: locator.text.before ?: locator.title ?: "Search result"
                    TextButton(
                        onClick = { onSearchResult(locator) },
                        modifier = Modifier.fillMaxWidth().padding(horizontal = 8.dp),
                    ) { Text(excerpt, modifier = Modifier.fillMaxWidth(), maxLines = 3, overflow = TextOverflow.Ellipsis) }
                }
            }
        }
    }
}

@Composable
private fun ReaderSyncIndicator(status: ReaderSyncStatus, contentColor: Color) {
    val formatter = remember { DateFormat.getTimeInstance(DateFormat.SHORT) }
    val label = readerSyncLabel(status) { formatter.format(Date(it)) }
    val icon = when (status.phase) {
        ReaderSyncPhase.Synced -> Icons.Outlined.CloudDone
        ReaderSyncPhase.Syncing -> Icons.Outlined.CloudSync
        ReaderSyncPhase.DeviceOnly -> Icons.Outlined.CloudOff
    }
    Row(
        modifier = Modifier.fillMaxWidth().padding(horizontal = 12.dp).semantics { liveRegion = LiveRegionMode.Polite },
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(4.dp, Alignment.End),
    ) {
        Icon(icon, contentDescription = null, tint = contentColor.copy(alpha = 0.78f), modifier = Modifier.size(16.dp))
        Text(label, color = contentColor.copy(alpha = 0.78f), style = MaterialTheme.typography.labelSmall, maxLines = 1)
    }
}

@Composable private fun ReaderPanelEmpty(message: String) {
    Text(message, modifier = Modifier.fillMaxWidth().padding(24.dp), color = MaterialTheme.colorScheme.onSurfaceVariant)
}

@Composable
private fun ReaderChromeTextButton(label: String, color: Color, modifier: Modifier = Modifier, onClick: () -> Unit) {
    TextButton(onClick = onClick, modifier = modifier, colors = ButtonDefaults.textButtonColors(contentColor = color)) { Text(label) }
}

private val Theme.label: String get() = when (this) {
    Theme.LIGHT -> "Light"
    Theme.SEPIA -> "Sepia"
    Theme.DARK -> "Dark"
}

private fun Theme.next(): Theme = when (this) {
    Theme.LIGHT -> Theme.SEPIA
    Theme.SEPIA -> Theme.DARK
    Theme.DARK -> Theme.LIGHT
}

private val ReaderFont.shortLabel get() = if (this == ReaderFont.Publisher) "Book font" else label
private val Theme.chromeSurface: Color get() = when (this) {
    Theme.LIGHT -> Color(0xFFFFF8F5)
    Theme.SEPIA -> Color(0xFFF4E6CF)
    Theme.DARK -> Color(0xFF201C1A)
}

private val Theme.chromeContent: Color get() = when (this) {
    Theme.LIGHT -> Color(0xFF241A17)
    Theme.SEPIA -> Color(0xFF34251D)
    Theme.DARK -> Color(0xFFFFF8F4)
}
