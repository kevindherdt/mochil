package net.kdhmedia.mochil.reader.epub

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.relocation.BringIntoViewRequester
import androidx.compose.foundation.relocation.bringIntoViewRequester
import androidx.compose.foundation.lazy.LazyListScope
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Delete
import androidx.compose.material.icons.outlined.EditNote
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalFocusManager
import androidx.compose.ui.focus.onFocusChanged
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import java.text.DateFormat
import java.util.Date
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

fun LazyListScope.epubAnnotationPanelItems(
    annotations: List<SyncedAnnotation>,
    conflicts: List<AnnotationConflict>,
    onOpen: (SyncedAnnotation) -> Unit,
    onEditNote: (SyncedAnnotation, String) -> Unit,
    onDelete: (SyncedAnnotation) -> Unit,
    onKeepLocal: (AnnotationConflict) -> Unit,
    onUseRemote: (AnnotationConflict) -> Unit,
    exportAvailable: Boolean,
    exporting: Boolean,
    onExport: () -> Unit,
) {
    val conflictingIDs = conflicts.mapTo(mutableSetOf()) { it.submitted.id }
    if (annotations.isNotEmpty()) {
        if (exportAvailable) item(key = "annotation-notes-export") {
            AnnotationNotesAction(exporting, conflicts.isEmpty(), onExport)
        }
        item { SectionHeading("Highlights") }
        items(annotations, key = { "annotation-${it.id}" }) { annotation ->
            AnnotationRow(annotation, annotation.id in conflictingIDs, onOpen, onEditNote, onDelete)
        }
    }
    if (conflicts.isNotEmpty()) {
        item { SectionHeading("Needs review", error = true) }
        items(conflicts, key = { "conflict-${it.submitted.id}" }) { conflict ->
            ConflictRow(conflict, onKeepLocal, onUseRemote)
        }
    }
}

@androidx.compose.runtime.Composable
private fun AnnotationNotesAction(exporting: Boolean, conflictFree: Boolean, onExport: () -> Unit) {
    Row(
        Modifier.fillMaxWidth().padding(start = 20.dp, end = 12.dp, top = 10.dp, bottom = 2.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Column(Modifier.weight(1f)) {
            Text("Mochil Notes", style = MaterialTheme.typography.titleSmall)
            Text(
                if (conflictFree) "Create a Notes page from your synced highlights and notes."
                else "Resolve annotation conflicts before creating a Notes page.",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }
        TextButton(onClick = onExport, enabled = conflictFree && !exporting) {
            Text(if (exporting) "Creating…" else "Create page")
        }
    }
}

@androidx.compose.runtime.Composable
private fun SectionHeading(label: String, error: Boolean = false) {
    Text(
        label,
        modifier = Modifier.padding(horizontal = 20.dp, vertical = 12.dp),
        style = MaterialTheme.typography.titleMedium,
        color = if (error) MaterialTheme.colorScheme.error else MaterialTheme.colorScheme.onSurface,
    )
}

@androidx.compose.runtime.Composable
private fun AnnotationRow(
    annotation: SyncedAnnotation,
    hasConflict: Boolean,
    onOpen: (SyncedAnnotation) -> Unit,
    onEditNote: (SyncedAnnotation, String) -> Unit,
    onDelete: (SyncedAnnotation) -> Unit,
) {
    var editing by remember(annotation.id) { mutableStateOf(false) }
    var draft by remember(annotation.id, annotation.note) { mutableStateOf(annotation.note) }
    val focusManager = LocalFocusManager.current
    val bringIntoViewRequester = remember { BringIntoViewRequester() }
    val scope = rememberCoroutineScope()
    Column(Modifier.fillMaxWidth().padding(horizontal = 8.dp, vertical = 2.dp)) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            TextButton(onClick = { onOpen(annotation) }, modifier = Modifier.weight(1f)) {
                Column(Modifier.fillMaxWidth()) {
                    Text(annotation.quote.ifBlank { "Saved annotation" }, maxLines = 3, overflow = TextOverflow.Ellipsis)
                    annotation.note.takeIf(String::isNotBlank)?.let {
                        Text(it, modifier = Modifier.padding(top = 4.dp), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant, maxLines = 2, overflow = TextOverflow.Ellipsis)
                    }
                }
            }
            IconButton(onClick = { editing = true }, enabled = !hasConflict) {
                Icon(Icons.Outlined.EditNote, contentDescription = if (annotation.note.isBlank()) "Add note to highlight" else "Edit note on highlight")
            }
            IconButton(onClick = { onDelete(annotation) }, enabled = !hasConflict) {
                Icon(Icons.Outlined.Delete, contentDescription = "Delete highlight", tint = MaterialTheme.colorScheme.error)
            }
        }
        if (editing) {
            Column(Modifier.fillMaxWidth().bringIntoViewRequester(bringIntoViewRequester)) {
                OutlinedTextField(
                    value = draft,
                    onValueChange = { draft = it },
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(horizontal = 12.dp)
                        .onFocusChanged { state ->
                            if (state.isFocused) scope.launch {
                                delay(250)
                                bringIntoViewRequester.bringIntoView()
                            }
                        },
                    label = { Text("Note") },
                    supportingText = { Text("Saved with this highlight") },
                    minLines = 2,
                    maxLines = 5,
                    keyboardOptions = KeyboardOptions(imeAction = ImeAction.Done),
                    keyboardActions = KeyboardActions(onDone = { focusManager.clearFocus() }),
                )
                Row(Modifier.align(Alignment.End).padding(end = 4.dp)) {
                    TextButton(onClick = { draft = annotation.note; editing = false }) { Text("Cancel") }
                    TextButton(onClick = { onEditNote(annotation, draft); editing = false; focusManager.clearFocus() }) { Text("Save note") }
                }
            }
        }
    }
}

@androidx.compose.runtime.Composable
private fun ConflictRow(conflict: AnnotationConflict, onKeepLocal: (AnnotationConflict) -> Unit, onUseRemote: (AnnotationConflict) -> Unit) {
    Column(Modifier.fillMaxWidth().padding(horizontal = 20.dp, vertical = 8.dp)) {
        Text("On this device", style = MaterialTheme.typography.labelLarge)
        Text(annotationVersionSummary(conflict.submitted), style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant, maxLines = 4, overflow = TextOverflow.Ellipsis)
        Text("From another device", modifier = Modifier.padding(top = 10.dp), style = MaterialTheme.typography.labelLarge)
        Text(annotationVersionSummary(conflict.current), style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant, maxLines = 4, overflow = TextOverflow.Ellipsis)
        if (annotationVersionSummary(conflict.submitted) == annotationVersionSummary(conflict.current)) {
            Text("The visible content matches, but the synchronization history differs.", modifier = Modifier.padding(top = 6.dp), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
        }
        Row {
            val excerpt = conflict.submitted.quote.ifBlank { conflict.submitted.note }.ifBlank { "annotation" }.take(48)
            TextButton(onClick = { onKeepLocal(conflict) }, modifier = Modifier.semantics { contentDescription = "Keep my version of $excerpt" }) { Text("Keep mine") }
            TextButton(onClick = { onUseRemote(conflict) }, modifier = Modifier.semantics { contentDescription = "Use the other device version of $excerpt" }) { Text("Use other") }
        }
    }
}

private fun annotationVersionSummary(annotation: SyncedAnnotation): String {
    if (annotation.deleted) return "Deleted"
    val content = buildList {
        annotation.quote.takeIf(String::isNotBlank)?.let(::add)
        annotation.note.takeIf(String::isNotBlank)?.let { add("Note: $it") }
    }.ifEmpty { listOf("Saved annotation") }.joinToString("\n")
    val context = buildList {
        annotation.chapter.takeIf(String::isNotBlank)?.let(::add)
        annotation.updatedAt.takeIf { it > 0 }?.let { add(DateFormat.getDateTimeInstance(DateFormat.MEDIUM, DateFormat.SHORT).format(Date(it))) }
    }.joinToString(" · ")
    return if (context.isBlank()) content else "$content\n$context"
}
