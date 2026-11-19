package net.kdhmedia.mochil.ui

import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.FilterChip
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Slider
import androidx.compose.material3.Surface
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import net.kdhmedia.mochil.narration.NarrationPreferences
import net.kdhmedia.mochil.narration.narrationSpeeds
import net.kdhmedia.mochil.narration.recordingSpeeds
import net.kdhmedia.mochil.reader.epub.CustomFontStore
import net.kdhmedia.mochil.reader.epub.CustomFontViewModel
import net.kdhmedia.mochil.reader.epub.EpubReaderPreferencesStore
import net.kdhmedia.mochil.reader.epub.ReaderFont
import net.kdhmedia.mochil.reader.epub.ReaderPreferences
import net.kdhmedia.mochil.reader.epub.TapZones
import net.kdhmedia.mochil.reader.epub.TextSharpness

@Composable
internal fun ReaderDefaultSettings(customFonts: CustomFontViewModel = viewModel()) {
    val context = LocalContext.current
    val store = remember(context) { EpubReaderPreferencesStore(context) }
    val customFontStore = remember(context) { CustomFontStore(context) }
    var preferences by remember { mutableStateOf(store.load()) }
    val fontState by customFonts.state.collectAsStateWithLifecycle()
    val fontPicker = rememberLauncherForActivityResult(ActivityResultContracts.OpenMultipleDocuments(), customFonts::import)
    val selectedCustomFont = fontState.fonts.firstOrNull { preferences.font == ReaderFont.Custom && preferences.customFontId == it.id }
    val previewCustomFamily = remember(selectedCustomFont?.id, fontState.fonts) {
        selectedCustomFont?.let(customFontStore::typeface)?.let(::FontFamily)
    }
    fun update(transform: ReaderPreferences.() -> ReaderPreferences) {
        preferences = store.update(transform)
    }
    SettingsCard("Reader defaults") {
    Text("Font", Modifier.padding(horizontal = 8.dp, vertical = 4.dp), style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
    LazyRow(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(6.dp)) {
        items(ReaderFont.entries.filterNot { it == ReaderFont.Custom }, key = ReaderFont::name) { font ->
            FilterChip(selected = preferences.font == font, onClick = { update { copy(font = font) } }, label = { Text(font.label) })
        }
        items(fontState.fonts, key = { "custom:${it.id}" }) { font ->
            FilterChip(selected = preferences.font == ReaderFont.Custom && preferences.customFontId == font.id, onClick = { update { copy(font = ReaderFont.Custom, customFontId = font.id) } }, label = { Text(font.name) })
        }
    }
    Row(Modifier.fillMaxWidth().padding(horizontal = 8.dp, vertical = 4.dp), verticalAlignment = Alignment.CenterVertically) {
        OutlinedButton(onClick = { fontPicker.launch(arrayOf("font/ttf", "font/otf", "application/x-font-ttf", "application/vnd.ms-opentype", "application/octet-stream")) }, enabled = !fontState.importing) {
            Text(if (fontState.importing) "Adding…" else "Add font files")
        }
        selectedCustomFont?.let { font -> TextButton(onClick = { update { copy(font = ReaderFont.Publisher, customFontId = null) }; customFonts.remove(font) }) { Text("Remove") } }
    }
    Text(
        "Choose TTF or OTF files from this device. Android may hide fonts installed by another app or theme store.",
        Modifier.padding(horizontal = 8.dp),
        style = MaterialTheme.typography.bodySmall,
        color = MaterialTheme.colorScheme.onSurfaceVariant,
    )
    fontState.message?.let { Text(it, Modifier.padding(horizontal = 8.dp, vertical = 2.dp), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant) }
    Text("Text size ${((preferences.fontSize * 100).toInt())}%", Modifier.padding(start = 8.dp, top = 10.dp), style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
    Slider(
        value = preferences.fontSize.toFloat(),
        onValueChange = { size -> update { copy(fontSize = size.toDouble()) } },
        valueRange = 0.7f..2f,
        steps = 12,
        modifier = Modifier.padding(horizontal = 8.dp),
    )
    Surface(Modifier.fillMaxWidth().padding(horizontal = 8.dp, vertical = 8.dp), shape = MaterialTheme.shapes.medium, color = MaterialTheme.colorScheme.surfaceVariant) {
        Column(Modifier.padding(18.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
            Text("Reader preview", style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
            Text(
                "Mochil keeps the page quiet so the story can lead.",
                fontSize = (17f * preferences.fontSize.toFloat()).sp,
                lineHeight = (25f * preferences.fontSize.toFloat()).sp,
                fontFamily = when (preferences.font) {
                    ReaderFont.Custom -> previewCustomFamily ?: FontFamily.Serif
                    ReaderFont.SansSerif, ReaderFont.Accessible -> FontFamily.SansSerif
                    ReaderFont.Monospace -> FontFamily.Monospace
                    else -> FontFamily.Serif
                },
            )
        }
    }
    Text("Sharper text", Modifier.padding(start = 8.dp, top = 10.dp), style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
    LazyRow(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(6.dp)) {
        items(TextSharpness.entries, key = TextSharpness::name) { level ->
            FilterChip(selected = preferences.sharpness == level, onClick = { update { copy(sharpness = level) } }, label = { Text(level.label.replaceFirstChar(Char::uppercase)) })
        }
    }
    Text("For e-ink screens: pure black on white with slightly heavier strokes.", Modifier.padding(horizontal = 8.dp, vertical = 2.dp), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
    Text("Publisher font preserves the EPUB's own typography. Imported fonts are copied into private app storage so they remain available offline.", Modifier.padding(horizontal = 8.dp, vertical = 2.dp), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
    }
}

@Composable
internal fun ReaderHardwareSettings(volumePageTurn: Boolean, onVolumePageTurn: (Boolean) -> Unit) {
    SettingsCard("Turning pages") {
    TapZoneSettings()
    Row(Modifier.fillMaxWidth().padding(horizontal = 8.dp, vertical = 6.dp)) {
        Column(Modifier.weight(1f).padding(end = 12.dp)) {
            Text("Use volume buttons to turn pages", style = MaterialTheme.typography.bodyLarge)
            Text("Page Up and Page Down always work. Enable this for BOOX and other e-ink devices that report their page buttons as volume keys.", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
        }
        Switch(checked = volumePageTurn, onCheckedChange = onVolumePageTurn)
    }
    }
}

@Composable
private fun TapZoneSettings() {
    val context = LocalContext.current
    val store = remember(context) { EpubReaderPreferencesStore(context) }
    var preferences by remember { mutableStateOf(store.load()) }
    Text("Tap zones", Modifier.padding(start = 8.dp, top = 10.dp), style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
    LazyRow(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(6.dp)) {
        items(TapZones.entries, key = TapZones::name) { zones ->
            FilterChip(selected = preferences.tapZones == zones, onClick = { preferences = store.update { it.copy(tapZones = zones) } }, label = { Text(zones.label.replaceFirstChar(Char::uppercase)) })
        }
    }
    Text("Edges turns the page at the outer quarters. One hand goes back from the left fifth and forward from the rest. The middle always opens the menu, except with Off, where every tap does.", Modifier.padding(horizontal = 8.dp, vertical = 2.dp), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
}

/** How fast reading aloud goes by default, for the server voice and for a book's own recording. */
@Composable
internal fun ReadAloudSettings() {
    val context = LocalContext.current
    val preferences = remember(context) { NarrationPreferences(context) }
    var voiceSpeed by remember { mutableStateOf(preferences.speed()) }
    var recordingSpeed by remember { mutableStateOf(preferences.recordingSpeed()) }
    SettingsCard("Read aloud") {
    Text("Voice speed", Modifier.padding(horizontal = 8.dp, vertical = 4.dp), style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
    LazyRow(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(6.dp)) {
        items(narrationSpeeds, key = { "voice:$it" }) { speed ->
            FilterChip(selected = voiceSpeed == speed, onClick = { voiceSpeed = speed; preferences.saveSpeed(speed) }, label = { Text("${speed}×") })
        }
    }
    Text("Recording speed", Modifier.padding(start = 8.dp, top = 10.dp), style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
    LazyRow(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(6.dp)) {
        items(recordingSpeeds, key = { "recording:$it" }) { speed ->
            FilterChip(selected = recordingSpeed == speed, onClick = { recordingSpeed = speed; preferences.saveRecordingSpeed(speed) }, label = { Text("${speed}×") })
        }
    }
    Text("The voice is the server's reading voice. Recording is a book that carries its own narration. The voice itself is chosen in the book's read-aloud menu.", Modifier.padding(horizontal = 8.dp, vertical = 2.dp), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
    }
}
