package net.kdhmedia.mochil.ui

import androidx.compose.material3.AlertDialog
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue

@Composable
internal fun WikiCreateDialog(folder: Boolean, busy: Boolean, onDismiss: () -> Unit, onCreate: (String) -> Unit) {
    var path by remember(folder) { mutableStateOf("") }
    AlertDialog(
        onDismissRequest = { if (!busy) onDismiss() },
        title = { Text(if (folder) "New folder" else "New page") },
        text = { OutlinedTextField(path, { path = it.take(500) }, singleLine = true, label = { Text(if (folder) "Folder path" else "Page path") }, supportingText = { Text("Use / to place it inside folders.") }) },
        confirmButton = { TextButton(onClick = { onCreate(path) }, enabled = path.isNotBlank() && !busy) { Text(if (busy) "Creating…" else "Create") } },
        dismissButton = { TextButton(onClick = onDismiss, enabled = !busy) { Text("Cancel") } },
    )
}
