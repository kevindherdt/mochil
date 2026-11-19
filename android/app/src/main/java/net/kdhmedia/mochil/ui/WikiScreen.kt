package net.kdhmedia.mochil.ui

import androidx.activity.compose.BackHandler
import net.kdhmedia.mochil.ui.theme.LocalReduceMotion
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.ChevronRight
import androidx.compose.material.icons.filled.ExpandMore
import androidx.compose.material.icons.filled.Folder
import androidx.compose.material.icons.outlined.Description
import androidx.compose.material.icons.filled.MoreVert
import androidx.compose.material.icons.filled.Close
import androidx.compose.material.icons.filled.Search
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.VerticalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.produceState
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import net.kdhmedia.mochil.wiki.WikiEntry
import net.kdhmedia.mochil.wiki.WikiMarkdownDocument
import net.kdhmedia.mochil.wiki.WikiState
import net.kdhmedia.mochil.wiki.parseWikiMarkdownDocument
import net.kdhmedia.mochil.wiki.WikiTreeIndex
import net.kdhmedia.mochil.wiki.recentWikiPages
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.currentCoroutineContext
import kotlinx.coroutines.ensureActive
import kotlinx.coroutines.withContext
import net.kdhmedia.mochil.diagnostics.PerformanceDiagnostics

data class WikiBindings(
    val state: WikiState,
    val open: (String) -> Unit,
    val refresh: () -> Unit,
    val search: (String) -> Unit,
    val toggle: (String) -> Unit,
    val select: (WikiEntry) -> Unit,
    val edit: (Boolean) -> Unit,
    val draft: (String) -> Unit,
    val save: () -> Unit,
    val toggleTask: (Int, Boolean) -> Unit,
    val closePage: () -> Unit,
    val create: (String, Boolean) -> Unit,
    val move: (String) -> Unit,
    val openLink: (String) -> Unit,
    val dismissMissingLink: () -> Unit,
    val createMissingLink: () -> Unit,
    val consumeNavigationHeading: () -> Unit,
    val trash: () -> Unit,
    val saveOffline: () -> Unit,
    val assetUrl: (WikiEntry) -> String,
    val authorizationHeader: String?,
    val cancelAttachment: () -> Unit = {},
)

@Composable
internal fun WikiScreen(bindings: WikiBindings) {
    WikiAttachmentLifecycle(bindings.cancelAttachment)
    val state = bindings.state
    BoxWithConstraints(Modifier.fillMaxSize()) {
        val split = maxWidth >= 720.dp
        if (split) Row(Modifier.fillMaxSize()) {
            WikiTree(state, bindings, Modifier.width(300.dp))
            VerticalDivider()
            WikiPagePane(state, bindings, showBack = false, modifier = Modifier.weight(1f))
        } else if (state.page == null) WikiTree(state, bindings, Modifier.fillMaxSize())
        else WikiPagePane(state, bindings, showBack = true, modifier = Modifier.fillMaxSize())
    }
}

@Composable
private fun WikiTree(state: WikiState, bindings: WikiBindings, modifier: Modifier) {
    var creating by remember { mutableStateOf<Boolean?>(null) }
    var pendingSelection by remember { mutableStateOf<WikiEntry?>(null) }
    var searching by rememberSaveable(state.libraryId) { mutableStateOf(false) }
    var query by rememberSaveable(state.libraryId) { mutableStateOf("") }
    val treeIndex by produceState<WikiTreeIndex?>(null, state.entries) {
        value = withContext(Dispatchers.Default) {
            PerformanceDiagnostics.measure("reader", "index_wiki_tree") { WikiTreeIndex(state.entries) }
        }
    }
    val visibleEntries by produceState(
        initialValue = emptyList(),
        treeIndex,
        state.expanded,
        state.searchQuery,
        state.searchResults,
        state.searching,
        query,
    ) {
        value = withContext(Dispatchers.Default) {
            PerformanceDiagnostics.measure("reader", "format_wiki_tree") {
                if (query.isNotBlank() && state.searchQuery == query.trim() && !state.searching) state.searchResults
                else treeIndex?.search(query, state.expanded).orEmpty()
            }
        }
    }
    // Sorting every page on each recomposition would be wasted work in a large vault.
    val recentPages = remember(state.entries) { recentWikiPages(state.entries) }
    val recent = if (query.isBlank()) recentPages else emptyList()
    Column(modifier) {
        Row(Modifier.fillMaxWidth().padding(horizontal = 12.dp, vertical = 4.dp), verticalAlignment = Alignment.CenterVertically) {
            if (searching) {
                OutlinedTextField(query, { query = it.take(200); bindings.search(query) }, Modifier.weight(1f), placeholder = { Text("Search pages and contents") }, singleLine = true)
                IconButton(onClick = { query = ""; bindings.search(""); searching = false }) { Icon(Icons.Default.Close, "Close search") }
            } else {
                Text("Wiki", Modifier.weight(1f), style = MaterialTheme.typography.titleLarge, fontWeight = FontWeight.SemiBold)
                IconButton(onClick = bindings.refresh, enabled = !state.loading && !state.saving) { Icon(Icons.Default.Refresh, "Refresh vault") }
                IconButton(onClick = { searching = true }) { Icon(Icons.Default.Search, "Search pages") }
                TextButton(onClick = { creating = false }) { Text("New page") }
                TextButton(onClick = { creating = true }) { Text("Folder") }
            }
        }
        when {
            state.loading && state.entries.isEmpty() -> Box(Modifier.fillMaxSize()) {
                if (LocalReduceMotion.current) {
                    CircularProgressIndicator(progress = { 1f }, modifier = Modifier.align(Alignment.Center))
                } else {
                    CircularProgressIndicator(Modifier.align(Alignment.Center))
                }
            }
            state.error != null && state.entries.isEmpty() -> Column(Modifier.padding(20.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
                Text(state.error, color = MaterialTheme.colorScheme.error)
                Button(onClick = { bindings.open(state.libraryId) }) { Text("Try again") }
            }
            state.entries.isEmpty() -> Text("This vault has no pages yet.", color = MaterialTheme.colorScheme.onSurfaceVariant, modifier = Modifier.padding(20.dp))
            else -> LazyColumn(Modifier.fillMaxSize()) {
                if (recent.isNotEmpty()) {
                    item(key = "recent-title", contentType = "title") {
                        Text("Recently edited", Modifier.padding(start = 12.dp, end = 12.dp, top = 8.dp, bottom = 2.dp), style = MaterialTheme.typography.titleSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                    }
                    items(recent, key = { "recent:${it.path}" }, contentType = { "recent" }) { entry ->
                        WikiTreeRow(entry, false, state.page?.id == entry.id, bindings.toggle, flat = true) { selected ->
                            if (state.editing && state.draft != state.page?.content.orEmpty()) pendingSelection = selected
                            else bindings.select(selected)
                        }
                    }
                    item(key = "all-title", contentType = "title") {
                        Text("All pages", Modifier.padding(start = 12.dp, end = 12.dp, top = 14.dp, bottom = 2.dp), style = MaterialTheme.typography.titleSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                    }
                }
                items(visibleEntries, key = { "${it.kind}:${it.path}" }, contentType = { it.kind }) { entry ->
                    WikiTreeRow(entry, entry.path in state.expanded, state.page?.id == entry.id, bindings.toggle) { selected ->
                        if (state.editing && state.draft != state.page?.content.orEmpty()) pendingSelection = selected
                        else bindings.select(selected)
                    }
                }
            }
        }
    }
    creating?.let { folder -> WikiCreateDialog(folder, state.saving, { creating = null }) { path -> bindings.create(path, folder); creating = null } }
    pendingSelection?.let { entry ->
        WikiDiscardDialog({ pendingSelection = null }) {
            pendingSelection = null
            bindings.select(entry)
        }
    }
}

@Composable
private fun WikiTreeRow(entry: WikiEntry, expanded: Boolean, selected: Boolean, onToggle: (String) -> Unit, flat: Boolean = false, onSelect: (WikiEntry) -> Unit) {
    Row(
        Modifier.fillMaxWidth().clickable { if (entry.isFolder) onToggle(entry.path) else onSelect(entry) }
            .padding(start = (12 + (if (flat) 0 else entry.depth) * 16).dp, end = 12.dp, top = 10.dp, bottom = 10.dp),
        horizontalArrangement = Arrangement.spacedBy(10.dp), verticalAlignment = Alignment.CenterVertically,
    ) {
        if (entry.isFolder) Icon(if (expanded) Icons.Default.ExpandMore else Icons.Default.ChevronRight, contentDescription = if (expanded) "Collapse folder" else "Expand folder")
        Icon(if (entry.isFolder) Icons.Default.Folder else Icons.Outlined.Description, contentDescription = null, tint = if (selected) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurfaceVariant)
        Column(Modifier.weight(1f)) {
            Text(entry.label, maxLines = 1, overflow = TextOverflow.Ellipsis, fontWeight = if (selected) FontWeight.SemiBold else FontWeight.Normal)
            if (flat && entry.path.contains('/')) Text(
                entry.path.substringBeforeLast('/').replace("/", " / "),
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            if (entry.matchSnippet.isNotBlank()) Text(
                entry.matchSnippet,
                maxLines = 2,
                overflow = TextOverflow.Ellipsis,
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }
    }
}

@Composable
private fun WikiPagePane(state: WikiState, bindings: WikiBindings, showBack: Boolean, modifier: Modifier) {
    val page = state.page
    var moving by remember { mutableStateOf(false) }
    var removing by remember { mutableStateOf(false) }
    var menuExpanded by remember { mutableStateOf(false) }
    var confirmClose by remember { mutableStateOf(false) }
    var outlineOpen by remember(page?.id) { mutableStateOf(false) }
    var outlineTarget by remember(page?.id) { mutableStateOf<Int?>(null) }
    val document by produceState<WikiMarkdownDocument?>(null, page?.id, page?.content, state.editing) {
        value = if (page == null || state.editing) null else withContext(Dispatchers.Default) {
            val context = currentCoroutineContext()
            PerformanceDiagnostics.measure("reader", "format_wiki_page") {
                parseWikiMarkdownDocument(page.content) { context.ensureActive() }
            }
        }
    }
    val outline = document?.outline.orEmpty()
    LaunchedEffect(page?.id, state.loading, state.navigationHeading, document) {
        if (state.loading || page == null || document == null) return@LaunchedEffect
        val heading = state.navigationHeading.takeIf(String::isNotBlank) ?: return@LaunchedEffect
        outline.firstOrNull { it.title.equals(heading, ignoreCase = true) }?.let { outlineTarget = it.blockIndex }
        bindings.consumeNavigationHeading()
    }
    val closePage = {
        if (state.editing && state.draft != page?.content.orEmpty()) confirmClose = true else bindings.closePage()
    }
    BackHandler(enabled = page != null, onBack = closePage)
    Column(modifier.padding(horizontal = 20.dp)) {
        Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
            if (showBack) IconButton(onClick = closePage) { Icon(Icons.AutoMirrored.Filled.ArrowBack, "Back to pages") }
            Text(page?.title ?: "Choose a page", Modifier.weight(1f), style = MaterialTheme.typography.titleLarge, maxLines = 1, overflow = TextOverflow.Ellipsis)
            if (page?.writable == true) {
                if (state.editing) TextButton(onClick = { bindings.edit(false) }, enabled = !state.saving) { Text("Cancel") }
                TextButton(onClick = if (state.editing) bindings.save else ({ bindings.edit(true) }), enabled = !state.saving) { Text(if (state.saving) "Saving…" else if (state.editing) "Save" else "Edit") }
            }
            if (page != null) {
                Box {
                    IconButton(onClick = { menuExpanded = true }, enabled = !state.saving) { Icon(Icons.Default.MoreVert, "Page options") }
                    DropdownMenu(expanded = menuExpanded, onDismissRequest = { menuExpanded = false }) {
                        if (outline.isNotEmpty()) DropdownMenuItem(text = { Text("Outline") }, onClick = { menuExpanded = false; outlineOpen = true })
                        DropdownMenuItem(
                            text = { Text(if (state.offlineSaving) "Saving offline…" else "Save vault pages offline") },
                            enabled = !state.offlineSaving,
                            onClick = { menuExpanded = false; bindings.saveOffline() },
                        )
                        if (page.writable) {
                            DropdownMenuItem(text = { Text("Rename or move") }, onClick = { menuExpanded = false; moving = true })
                            DropdownMenuItem(text = { Text("Move to vault trash") }, onClick = { menuExpanded = false; removing = true })
                        }
                    }
                }
            }
        }
        state.error?.let { Text(it, color = MaterialTheme.colorScheme.error, modifier = Modifier.padding(vertical = 8.dp)) }
        state.attachmentStatus?.let { Text(it, color = MaterialTheme.colorScheme.onSurfaceVariant, style = MaterialTheme.typography.bodySmall, modifier = Modifier.padding(vertical = 6.dp)) }
        if (state.offlineSaving) Text(
            if (state.offlineTotal > 0) "Saving offline pages ${state.offlineSaved} of ${state.offlineTotal}…" else "Preparing offline pages…",
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            style = MaterialTheme.typography.bodySmall,
            modifier = Modifier.padding(vertical = 6.dp),
        ) else state.offlineStatus?.let { status -> Text(
            status,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            style = MaterialTheme.typography.bodySmall,
            modifier = Modifier.padding(vertical = 6.dp),
        ) }
        if (state.offlineCopy) Text(
            "Saved copy · reconnect to edit or refresh",
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            style = MaterialTheme.typography.bodySmall,
            modifier = Modifier.padding(vertical = 6.dp),
        )
        if (state.draftRecovered) Text(
            "Recovered unsaved changes from this device.",
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            style = MaterialTheme.typography.bodySmall,
            modifier = Modifier.padding(vertical = 6.dp),
        )
        when {
            state.loading && page == null -> if (LocalReduceMotion.current) {
                CircularProgressIndicator(progress = { 1f }, modifier = Modifier.align(Alignment.CenterHorizontally).padding(32.dp))
            } else {
                CircularProgressIndicator(Modifier.align(Alignment.CenterHorizontally).padding(32.dp))
            }
            page == null -> Text("Select a page from the vault.", color = MaterialTheme.colorScheme.onSurfaceVariant, modifier = Modifier.padding(vertical = 24.dp))
            state.editing -> WikiEditor(page.id, state.draft, bindings.draft)
            document == null -> AccessibleLoadingIndicator("Formatting Wiki page", Modifier.fillMaxSize())
            else -> WikiMarkdownView(
                blocks = document!!.blocks,
                pageId = page.id,
                modifier = Modifier.fillMaxSize(),
                onWikiLink = bindings.openLink,
                entries = state.entries,
                pagePath = state.selectedPath,
                assetUrl = bindings.assetUrl,
                authorizationHeader = bindings.authorizationHeader,
                backlinks = state.backlinks,
                onBacklink = bindings.select,
                onToggleTask = if (page.writable && !state.saving) bindings.toggleTask else null,
                scrollToBlock = outlineTarget,
                onScrollComplete = { outlineTarget = null },
            )
        }
    }
    if (moving) WikiMoveDialog(state.selectedPath, state.saving, { moving = false }) { bindings.move(it); moving = false }
    if (removing) WikiTrashDialog(page?.title.orEmpty(), state.saving, { removing = false }) { bindings.trash(); removing = false }
    if (confirmClose) WikiDiscardDialog({ confirmClose = false }) { confirmClose = false; bindings.closePage() }
    if (outlineOpen) WikiOutlineDialog(outline, { outlineOpen = false }) { entry ->
        outlineOpen = false
        outlineTarget = entry.blockIndex
    }
    state.missingLink?.let { link ->
        WikiMissingLinkDialog(link.label, link.path, state.saving, bindings.dismissMissingLink, bindings.createMissingLink)
    }
}
