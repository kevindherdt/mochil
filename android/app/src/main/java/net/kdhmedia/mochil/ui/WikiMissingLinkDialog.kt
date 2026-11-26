package net.kdhmedia.mochil.ui

import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable

@Composable
internal fun WikiMissingLinkDialog(
    label: String,
    path: String,
    busy: Boolean,
    onDismiss: () -> Unit,
    onCreate: () -> Unit,
) {
    AlertDialog(
        onDismissRequest = { if (!busy) onDismiss() },
        title = { Text("Create “$label”?") },
        text = { Text("This note link does not exist yet. Create $path in this vault?") },
        confirmButton = {
            TextButton(onClick = onCreate, enabled = !busy) { Text(if (busy) "Creating…" else "Create page") }
        },
        dismissButton = { TextButton(onClick = onDismiss, enabled = !busy) { Text("Cancel") } },
    )
}
