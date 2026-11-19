package net.kdhmedia.mochil.ui

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import net.kdhmedia.mochil.catalogue.PublicationCollection

private enum class CollectionManageStep { Choose, Rename, ConfirmDelete }

/** What long-pressing a collection offers: rename it, or delete it (the books themselves are never touched). */
@Composable
internal fun CollectionManageDialogs(
    collection: PublicationCollection,
    onDismiss: () -> Unit,
    onRename: (PublicationCollection, String) -> Unit,
    onDelete: (PublicationCollection) -> Unit,
) {
    var step by remember(collection.id) { mutableStateOf(CollectionManageStep.Choose) }
    var name by remember(collection.id) { mutableStateOf(collection.name) }
    when (step) {
        CollectionManageStep.Choose -> AlertDialog(
            onDismissRequest = onDismiss,
            title = { Text(collection.name) },
            text = {
                Column(Modifier.fillMaxWidth()) {
                    TextButton({ step = CollectionManageStep.Rename }, Modifier.fillMaxWidth()) { Text("Rename", Modifier.fillMaxWidth()) }
                    TextButton({ step = CollectionManageStep.ConfirmDelete }, Modifier.fillMaxWidth()) { Text("Delete collection", Modifier.fillMaxWidth()) }
                }
            },
            confirmButton = { TextButton(onDismiss) { Text("Cancel") } },
        )
        CollectionManageStep.Rename -> AlertDialog(
            onDismissRequest = onDismiss,
            title = { Text("Rename collection") },
            text = { OutlinedTextField(name, { name = it.take(80) }, singleLine = true, label = { Text("Name") }) },
            confirmButton = { TextButton({ onRename(collection, name); onDismiss() }, enabled = name.isNotBlank()) { Text("Save") } },
            dismissButton = { TextButton(onDismiss) { Text("Cancel") } },
        )
        CollectionManageStep.ConfirmDelete -> AlertDialog(
            onDismissRequest = onDismiss,
            title = { Text("Delete ${collection.name}?") },
            text = { Text("The collection goes away. The books in it stay in your library.") },
            confirmButton = { TextButton({ onDelete(collection); onDismiss() }) { Text("Delete") } },
            dismissButton = { TextButton(onDismiss) { Text("Cancel") } },
        )
    }
}
