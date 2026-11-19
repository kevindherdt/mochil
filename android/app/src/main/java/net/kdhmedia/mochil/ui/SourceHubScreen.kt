package net.kdhmedia.mochil.ui

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.automirrored.filled.KeyboardArrowRight
import androidx.compose.material.icons.automirrored.outlined.MenuBook
import androidx.compose.material.icons.outlined.Cloud
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.heading
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp

@Composable
fun SourceHubScreen(onBack: () -> Unit) {
    var active by rememberSaveable { mutableStateOf<String?>(null) }
    when (active) {
        "opds" -> return OpdsScreen(onBack = { active = null })
        "webdav" -> return WebDavScreen(onBack = { active = null })
    }
    Column(Modifier.fillMaxSize().padding(horizontal = 12.dp, vertical = 8.dp)) {
        Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
            IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, "Back") }
            Column(Modifier.padding(start = 6.dp)) {
                Text("Sources", Modifier.semantics { heading() }, style = MaterialTheme.typography.headlineSmall, fontWeight = FontWeight.SemiBold)
                Text("Personal libraries outside this server", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
        }
        Column(Modifier.fillMaxWidth().padding(top = 20.dp)) {
            SourceRow(
                title = "OPDS catalogues",
                detail = "Browse catalogues and save publications for offline reading.",
                icon = { Icon(Icons.AutoMirrored.Outlined.MenuBook, null, tint = MaterialTheme.colorScheme.primary) },
            ) { active = "opds" }
            HorizontalDivider(Modifier.padding(start = 56.dp), color = MaterialTheme.colorScheme.outlineVariant)
            SourceRow(
                title = "WebDAV libraries",
                detail = "Read-only access to cloud folders, books, audio, and video.",
                icon = { Icon(Icons.Outlined.Cloud, null, tint = MaterialTheme.colorScheme.primary) },
            ) { active = "webdav" }
        }
        Text(
            "Connections and credentials stay encrypted on this device. Removing a source never changes its remote files.",
            Modifier.padding(horizontal = 12.dp, vertical = 24.dp),
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
    }
}

@Composable
private fun SourceRow(title: String, detail: String, icon: @Composable () -> Unit, onClick: () -> Unit) {
    Row(
        Modifier.fillMaxWidth().clickable(role = Role.Button, onClickLabel = "Open $title", onClick = onClick).padding(horizontal = 12.dp, vertical = 16.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(16.dp),
    ) {
        icon()
        Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(3.dp)) {
            Text(title, style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.Medium)
            Text(detail, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
        }
        Icon(Icons.AutoMirrored.Filled.KeyboardArrowRight, null, tint = MaterialTheme.colorScheme.onSurfaceVariant)
    }
}
