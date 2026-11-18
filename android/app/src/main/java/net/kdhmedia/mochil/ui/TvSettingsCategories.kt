package net.kdhmedia.mochil.ui

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.Alignment
import androidx.compose.ui.focus.onFocusChanged
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.focus.focusRestorer
import androidx.compose.foundation.focusGroup
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp

internal enum class TvSettingsCategory(val label: String, val description: String) {
    Appearance("Appearance", "Theme and brightness"),
    Libraries("Libraries", "Visible media types"),
    Playback("Playback", "Quality and network use"),
    Updates("Updates", "Version and installation"),
    Connection("Connection", "Server and TV readiness"),
    Diagnostics("Diagnostics", "Performance and crashes"),
}

@Composable
internal fun TvSettingsCategoryList(selected: TvSettingsCategory, updatesEnabled: Boolean = true, onSelected: (TvSettingsCategory) -> Unit) {
    val initialFocus = remember { FocusRequester() }
    LaunchedEffect(Unit) { initialFocus.requestFocus() }
    LazyColumn(
        modifier = Modifier.fillMaxHeight().focusRestorer().focusGroup(),
        verticalArrangement = Arrangement.spacedBy(4.dp),
    ) {
        val categories = TvSettingsCategory.entries.filter { updatesEnabled || it != TvSettingsCategory.Updates }
        itemsIndexed(categories, key = { _, category -> category.name }) { index, category ->
            var focused by remember { mutableStateOf(false) }
            val shape = RoundedCornerShape(12.dp)
            Card(
                onClick = { onSelected(category) },
                modifier = Modifier.fillMaxWidth().heightIn(min = 48.dp)
                    .then(if (index == 0) Modifier.focusRequester(initialFocus) else Modifier)
                    .onFocusChanged { focused = it.isFocused; if (it.isFocused && selected != category) onSelected(category) }
                    .tvFocusVisual(focused, shape, selected == category, focusedScale = 1f),
                shape = shape,
                colors = CardDefaults.cardColors(containerColor = if (focused || selected == category) MaterialTheme.colorScheme.primaryContainer.copy(alpha = .72f) else MaterialTheme.colorScheme.surface),
            ) {
                Box(Modifier.fillMaxWidth().heightIn(min = 48.dp).padding(horizontal = 12.dp, vertical = 4.dp), contentAlignment = Alignment.CenterStart) {
                    Text(category.label, style = MaterialTheme.typography.titleSmall, fontWeight = if (selected == category) FontWeight.SemiBold else FontWeight.Medium)
                }
            }
        }
    }
}
