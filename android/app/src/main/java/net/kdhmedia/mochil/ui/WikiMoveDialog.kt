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
internal fun WikiMoveDialog(currentPath: String, busy: Boolean, onDismiss: () -> Unit, onMove: (String) -> Unit) {
    var path by remember(currentPath) { mutableStateOf(currentPath) }
    AlertDialog(
        onDismissRequest = { if (!busy) onDismiss() },
        title = { Text("Rename or move page") },
        text = {
            OutlinedTextField(
                value = path,
                onValueChange = { path = it.take(500) },
                label = { Text("Path inside this vault") },
                supportingText = { Text("Use folders like Projects/Plan.md") },
                singleLine = true,
            )
        },
        confirmButton = { TextButton(onClick = { onMove(path) }, enabled = !busy && path.isNotBlank() && path != currentPath) { Text(if (busy) "Moving…" else "Move") } },
        dismissButton = { TextButton(onClick = onDismiss, enabled = !busy) { Text("Cancel") } },
    )
}
