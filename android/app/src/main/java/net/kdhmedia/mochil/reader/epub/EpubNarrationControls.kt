package net.kdhmedia.mochil.reader.epub

import androidx.compose.foundation.layout.Row
import androidx.compose.ui.Modifier
import androidx.compose.ui.Alignment
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Box
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Pause
import androidx.compose.material.icons.filled.PlayArrow
import androidx.compose.material.icons.filled.Tune
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import net.kdhmedia.mochil.narration.NarrationSleep
import net.kdhmedia.mochil.narration.NarrationUi

private val sleepChoices = listOf(
    "Off" to NarrationSleep.Off,
    "15 minutes" to NarrationSleep.After(15),
    "30 minutes" to NarrationSleep.After(30),
    "45 minutes" to NarrationSleep.After(45),
    "1 hour" to NarrationSleep.After(60),
    "End of this chapter" to NarrationSleep.EndOfChapter,
)

/** Play or pause reading aloud, and a menu for the voice, speed and sleep timer. */
@Composable
internal fun EpubNarrationControls(
    ui: NarrationUi,
    onToggle: () -> Unit,
    onVoice: (String) -> Unit,
    onSpeed: (Float) -> Unit,
    onSleep: (NarrationSleep) -> Unit,
    onStop: () -> Unit,
    /** Set while the reader has turned away from the sentence being read: takes the page back to it. */
    onBackToReading: (() -> Unit)? = null,
) {
    var menu by remember { mutableStateOf(false) }
    Row {
        IconButton(onClick = onToggle) {
            Icon(
                if (ui.playing) Icons.Default.Pause else Icons.Default.PlayArrow,
                contentDescription = if (ui.playing) "Pause reading aloud" else "Read aloud",
            )
        }
        onBackToReading?.let { TextButton(onClick = it) { Text("Back to reading") } }
        IconButton(onClick = { menu = true }) { Icon(Icons.Default.Tune, contentDescription = "Read aloud settings") }
        DropdownMenu(expanded = menu, onDismissRequest = { menu = false }) {
            if (ui.active) DropdownMenuItem(text = { Text("Stop reading aloud") }, onClick = { menu = false; onStop() })
            MenuHeading("Speed")
            ui.speeds.forEach { speed ->
                DropdownMenuItem(text = { Text(choice("${speed}×".replace(".0×", "×"), speed == ui.speed)) }, onClick = { onSpeed(speed) })
            }
            MenuHeading(if (ui.sleepText.isEmpty()) "Sleep timer" else ui.sleepText)
            sleepChoices.filter { ui.chapterSleep || it.second != NarrationSleep.EndOfChapter }.forEach { (label, sleep) ->
                DropdownMenuItem(text = { Text(label) }, onClick = { menu = false; onSleep(sleep) })
            }
            if (ui.voices.isNotEmpty()) MenuHeading("Voice")
            ui.voices.forEach { voice ->
                val name = voice.name + if (voice.language == "dutch") " (Dutch)" else ""
                DropdownMenuItem(text = { Text(choice(name, voice.id == ui.voice)) }, onClick = { menu = false; onVoice(voice.id) })
            }
        }
    }
}

/**
 * The same narration settings as the bar's menu, set out as rows for the reader settings sheet's Narration tab — the web
 * reader's third tab. The bar keeps the menu, so nobody has to open the sheet to change speed mid-sentence.
 */
@Composable
internal fun EpubNarrationSettings(
    ui: NarrationUi,
    onVoice: (String) -> Unit,
    onSpeed: (Float) -> Unit,
    onSleep: (NarrationSleep) -> Unit,
    onStop: () -> Unit,
) {
    var voices by remember { mutableStateOf(false) }
    Column(Modifier.fillMaxWidth()) {
        Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
            Text("Speed", Modifier.weight(1f))
            TextButton(onClick = {
                val index = ui.speeds.indexOf(ui.speed)
                if (ui.speeds.isNotEmpty()) onSpeed(ui.speeds[(index + 1) % ui.speeds.size])
            }) { Text("${ui.speed}× speed") }
        }
        Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
            Text("Sleep timer", Modifier.weight(1f))
            Box {
                var sleep by remember { mutableStateOf(false) }
                TextButton(onClick = { sleep = true }) { Text(if (ui.sleepText.isEmpty()) "Off" else ui.sleepText) }
                DropdownMenu(expanded = sleep, onDismissRequest = { sleep = false }) {
                    sleepChoices.filter { ui.chapterSleep || it.second != NarrationSleep.EndOfChapter }.forEach { (label, choiceSleep) ->
                        DropdownMenuItem(text = { Text(choice(label, label == ui.sleepText || (label == "Off" && ui.sleepText.isEmpty()))) }, onClick = { sleep = false; onSleep(choiceSleep) })
                    }
                }
            }
        }
        if (ui.voices.isNotEmpty()) Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
            Text("Voice", Modifier.weight(1f))
            Box {
                TextButton(onClick = { voices = true }) {
                    Text(ui.voices.firstOrNull { it.id == ui.voice }?.name ?: "Default")
                }
                DropdownMenu(expanded = voices, onDismissRequest = { voices = false }) {
                    ui.voices.forEach { voice ->
                        val name = voice.name + if (voice.language == "dutch") " (Dutch)" else ""
                        DropdownMenuItem(text = { Text(choice(name, voice.id == ui.voice)) }, onClick = { voices = false; onVoice(voice.id) })
                    }
                }
            }
        }
        if (ui.active) TextButton(onClick = onStop) { Text("Stop reading aloud") }
    }
}

@Composable
private fun MenuHeading(text: String) {
    DropdownMenuItem(
        text = { Text(text, style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.primary) },
        onClick = {},
        enabled = false,
    )
}

private fun choice(label: String, selected: Boolean) = if (selected) "✓ $label" else label
