package net.kdhmedia.mochil.ui

import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable

@Composable
internal fun WikiTrashDialog(title: String, busy: Boolean, onDismiss: () -> Unit, onConfirm: () -> Unit) {
    AlertDialog(
        onDismissRequest = { if (!busy) onDismiss() },
        title = { Text("Remove this page?") },
        text = { Text("“${title.ifBlank { "This page" }}” will move to the vault’s .trash folder, where it can still be recovered.") },
        confirmButton = { TextButton(onClick = onConfirm, enabled = !busy) { Text(if (busy) "Moving…" else "Move to trash") } },
        dismissButton = { TextButton(onClick = onDismiss, enabled = !busy) { Text("Cancel") } },
    )
}
