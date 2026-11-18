package net.kdhmedia.mochil.ui

import androidx.compose.foundation.clickable
import net.kdhmedia.mochil.ui.theme.LocalReduceMotion
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Check
import androidx.compose.material3.Button
import androidx.compose.material3.FilterChip
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.semantics.stateDescription
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import net.kdhmedia.mochil.catalogue.CollectionPickerState
import net.kdhmedia.mochil.catalogue.PublicationCollection
import net.kdhmedia.mochil.catalogue.ReadingStatus
import net.kdhmedia.mochil.catalogue.readingStatus
import net.kdhmedia.mochil.catalogue.contains
import net.kdhmedia.mochil.catalogue.holdsThroughSeries

@OptIn(ExperimentalMaterial3Api::class, ExperimentalLayoutApi::class)
@Composable
internal fun CollectionPickerSheet(
    state: CollectionPickerState,
    onToggle: (PublicationCollection) -> Unit,
    onCreate: (String) -> Unit,
    onShowBookDetails: (net.kdhmedia.mochil.catalogue.BookSummary) -> Unit,
    onSetReadingStatus: (ReadingStatus) -> Unit,
    onClose: () -> Unit,
) {
    val target = state.target ?: return
    var newName by remember(target) { mutableStateOf("") }
    ModalBottomSheet(onDismissRequest = onClose) {
        Column(Modifier.fillMaxWidth().padding(bottom = 24.dp)) {
            Text(if (state.book == null) "Add to collection" else "Book options", style = MaterialTheme.typography.titleLarge, modifier = Modifier.padding(horizontal = 20.dp))
            Text(target.label, style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant, maxLines = 2, overflow = TextOverflow.Ellipsis, modifier = Modifier.padding(horizontal = 20.dp, vertical = 4.dp))
            state.book?.let { book ->
                TextButton(
                    onClick = { onClose(); onShowBookDetails(book) },
                    modifier = Modifier.fillMaxWidth().padding(horizontal = 8.dp),
                ) { Text("Book details", Modifier.fillMaxWidth().padding(horizontal = 8.dp)) }
                HorizontalDivider()
                Text("Reading status", style = MaterialTheme.typography.titleSmall, modifier = Modifier.padding(horizontal = 20.dp, vertical = 12.dp))
                val current = book.readingStatus()
                FlowRow(Modifier.fillMaxWidth().padding(horizontal = 20.dp), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    ReadingStatus.entries.forEach { option ->
                        FilterChip(
                            selected = option == current,
                            onClick = { onSetReadingStatus(option) },
                            enabled = !state.statusBusy,
                            label = { Text(option.label) },
                        )
                    }
                }
                HorizontalDivider(Modifier.padding(top = 12.dp))
                Text("Collections", style = MaterialTheme.typography.titleSmall, modifier = Modifier.padding(horizontal = 20.dp, vertical = 12.dp))
            }
            when {
                state.loading -> Row(Modifier.fillMaxWidth().padding(28.dp), horizontalArrangement = Arrangement.Center) {
                    if (LocalReduceMotion.current) CircularProgressIndicator(progress = { 1f }) else CircularProgressIndicator()
                }
                !state.collectionsEnabled -> Text("Collections are available when this device is connected to a Mochil server.", color = MaterialTheme.colorScheme.onSurfaceVariant, modifier = Modifier.padding(20.dp))
                state.collections.isEmpty() -> Text("No collections yet. Create one below.", color = MaterialTheme.colorScheme.onSurfaceVariant, modifier = Modifier.padding(20.dp))
                else -> LazyColumn {
                    items(state.collections, key = PublicationCollection::id) { collection ->
                        val selected = collection.contains(target.member)
                        // Smart collections fill themselves from a search, and a series added whole holds its books: neither can be toggled per book.
                        val automatic = collection.smart && !selected
                        val viaSeries = !selected && !automatic && collection.holdsThroughSeries(state.book)
                        val locked = automatic || viaSeries
                        Row(
                            Modifier.fillMaxWidth()
                                .semantics { stateDescription = when { selected || viaSeries -> "In collection"; automatic -> "Automatic collection"; else -> "Not in collection" } }
                                .clickable(enabled = state.busyId == null && !locked, role = Role.Checkbox) { onToggle(collection) }
                                .padding(horizontal = 20.dp, vertical = 14.dp),
                            horizontalArrangement = Arrangement.spacedBy(12.dp), verticalAlignment = Alignment.CenterVertically,
                        ) {
                            Column(Modifier.weight(1f)) {
                                Text(collection.name, style = MaterialTheme.typography.titleSmall)
                                Text("${collection.count} ${if (collection.count == 1) "item" else "items"}", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                            }
                            if (state.busyId == collection.id) {
                                if (LocalReduceMotion.current) CircularProgressIndicator(progress = { 1f }, modifier = Modifier.size(22.dp)) else CircularProgressIndicator(Modifier.size(22.dp))
                            }
                            else if (selected) Text("Remove", color = MaterialTheme.colorScheme.primary)
                            else if (viaSeries) Text("Via series", color = MaterialTheme.colorScheme.onSurfaceVariant)
                            else if (automatic) Text("Automatic", color = MaterialTheme.colorScheme.onSurfaceVariant)
                            else Text("Add", color = MaterialTheme.colorScheme.primary)
                        }
                    }
                }
            }
            HorizontalDivider()
            Row(Modifier.fillMaxWidth().padding(20.dp), horizontalArrangement = Arrangement.spacedBy(10.dp), verticalAlignment = Alignment.CenterVertically) {
                OutlinedTextField(newName, { newName = it.take(80) }, modifier = Modifier.weight(1f), enabled = state.collectionsEnabled, singleLine = true, label = { Text("New collection") })
                Button(onClick = { onCreate(newName); newName = "" }, enabled = state.collectionsEnabled && newName.isNotBlank() && !state.creating) { Text(if (state.creating) "Creating…" else "Create") }
            }
            state.message?.let { Text(it, color = MaterialTheme.colorScheme.onSurfaceVariant, modifier = Modifier.padding(horizontal = 20.dp)) }
        }
    }
}

data class PublicationCollectionActions(
    val book: (net.kdhmedia.mochil.catalogue.BookSummary) -> Unit,
    val series: (net.kdhmedia.mochil.catalogue.SeriesSummary) -> Unit,
    val rename: (PublicationCollection, String) -> Unit = { _, _ -> },
    val delete: (PublicationCollection) -> Unit = {},
)
