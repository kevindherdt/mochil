package net.kdhmedia.mochil.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.FilterChip
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Slider
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.Shadow
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import net.kdhmedia.mochil.video.SubtitlePreferences
import net.kdhmedia.mochil.video.SubtitlePreferencesStore
import net.kdhmedia.mochil.video.SubtitlePreset

@Composable
internal fun SubtitleAppearanceSettings(television: Boolean = false) {
    val context = androidx.compose.ui.platform.LocalContext.current
    val store = remember(context) { SubtitlePreferencesStore(context) }
    var preferences by remember { mutableStateOf(store.load()) }
    fun update(value: SubtitlePreferences) { preferences = value.normalized(); store.save(preferences) }

    Text("Subtitle style", color = MaterialTheme.colorScheme.onSurfaceVariant)
    Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
        if (television) TvSettingsChoiceRail(
            choices = SubtitlePreset.entries,
            selected = { preferences.preset == it },
            label = SubtitlePreset::label,
            key = SubtitlePreset::name,
            onSelect = { update(preferences.copy(preset = it)) },
            modifier = Modifier.fillMaxWidth(),
        ) else SubtitlePreset.entries.chunked(2).forEach { choices ->
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                choices.forEach { preset -> FilterChip(
                    selected = preferences.preset == preset,
                    onClick = { update(preferences.copy(preset = preset)) },
                    label = { Text(preset.label) },
                ) }
            }
        }
    }
    Text(preferences.preset.description, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
    Text("Text size ${(preferences.textScale * 100).toInt()}%", style = MaterialTheme.typography.labelMedium)
    Slider(preferences.textScale, { update(preferences.copy(textScale = it)) }, valueRange = .75f..1.5f, steps = 5, modifier = Modifier.fillMaxWidth().tvFocusBorder())
    if (preferences.preset == SubtitlePreset.Cinema || preferences.preset == SubtitlePreset.HighContrast) {
        Text("Background ${(preferences.backgroundOpacity * 100).toInt()}%", style = MaterialTheme.typography.labelMedium)
        Slider(preferences.backgroundOpacity, { update(preferences.copy(backgroundOpacity = it)) }, valueRange = 0f..1f, steps = 9, modifier = Modifier.fillMaxWidth().tvFocusBorder())
    }
    val previewBackground = when (preferences.preset) {
        SubtitlePreset.HighContrast -> Color.Black.copy(alpha = preferences.backgroundOpacity)
        SubtitlePreset.Cinema -> Color.Black.copy(alpha = preferences.backgroundOpacity * .22f)
        else -> Color.Transparent
    }
    Column(Modifier.fillMaxWidth().background(Color(0xFF334155), RoundedCornerShape(12.dp)).padding(vertical = 26.dp, horizontal = 14.dp)) {
        Text(
            "The story stays in focus.",
            modifier = Modifier.background(previewBackground, RoundedCornerShape(3.dp)).padding(horizontal = 5.dp, vertical = 2.dp),
            color = Color.White,
            fontSize = (18f * preferences.textScale).sp,
            style = TextStyle(shadow = Shadow(Color.Black, blurRadius = if (preferences.preset == SubtitlePreset.Anime) 1.5f else 5f)),
        )
    }
}
