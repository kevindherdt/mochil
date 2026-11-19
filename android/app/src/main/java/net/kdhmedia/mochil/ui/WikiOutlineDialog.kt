package net.kdhmedia.mochil.ui

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import net.kdhmedia.mochil.wiki.WikiOutlineEntry

@Composable
internal fun WikiOutlineDialog(entries: List<WikiOutlineEntry>, onDismiss: () -> Unit, onSelect: (WikiOutlineEntry) -> Unit) {
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("Outline") },
        text = {
            LazyColumn {
                items(entries, key = { "${it.blockIndex}:${it.title}" }) { entry ->
                    Text(
                        entry.title,
                        modifier = Modifier.fillMaxWidth().clickable { onSelect(entry) }
                            .padding(start = ((entry.level - 1).coerceAtLeast(0) * 14).dp, top = 12.dp, bottom = 12.dp),
                        fontWeight = if (entry.level <= 2) FontWeight.SemiBold else FontWeight.Normal,
                        maxLines = 2,
                        overflow = TextOverflow.Ellipsis,
                    )
                }
            }
        },
        confirmButton = { TextButton(onClick = onDismiss) { Text("Close") } },
    )
}
