package net.kdhmedia.mochil.reader.epub

import androidx.compose.foundation.layout.Row
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
            if (ui.active) {
                DropdownMenuItem(text = { Text("Stop reading aloud") }, onClick = { menu = false; onStop() })
            }
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

@Composable
private fun MenuHeading(text: String) {
    DropdownMenuItem(
        text = { Text(text, style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.primary) },
        onClick = {},
        enabled = false,
    )
}

private fun choice(label: String, selected: Boolean) = if (selected) "✓ $label" else label
