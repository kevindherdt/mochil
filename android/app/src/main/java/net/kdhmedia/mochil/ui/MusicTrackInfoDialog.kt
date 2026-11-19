package net.kdhmedia.mochil.ui

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.widthIn
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import java.util.Locale
import net.kdhmedia.mochil.audio.AudioTrack

@Composable
internal fun MusicTrackInfoDialog(track: AudioTrack, onDismiss: () -> Unit) {
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(track.title, maxLines = 2, overflow = TextOverflow.Ellipsis) },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                trackMetadataDetails(track).forEach { (label, value) ->
                    Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(16.dp)) {
                        Text(label, Modifier.widthIn(min = 88.dp), style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
                        Text(value, Modifier.weight(1f), textAlign = TextAlign.End, style = MaterialTheme.typography.bodyMedium)
                    }
                }
            }
        },
        confirmButton = { TextButton(onClick = onDismiss) { Text("Done") } },
    )
}

internal fun trackMetadataDetails(track: AudioTrack): List<Pair<String, String>> = buildList {
    track.artist.trim().takeIf { it.isNotBlank() && !it.equals("Unknown artist", true) }?.let { add("Artist" to it) }
    track.albumArtist.trim().takeIf { it.isNotBlank() && !it.equals(track.artist.trim(), true) }?.let { add("Album artist" to it) }
    track.album.trim().takeIf(String::isNotBlank)?.let { add("Album" to it) }
    track.genre.trim().takeIf(String::isNotBlank)?.let { add("Genre" to it) }
    if (track.year in 1000..2100) add("Year" to track.year.toString())
    if (track.discNumber > 0) add("Disc" to track.discNumber.toString())
    if (track.trackNumber > 0) add("Track" to track.trackNumber.toString())
    track.container.trim().takeIf(String::isNotBlank)?.let { add("Format" to it.uppercase(Locale.ROOT)) }
    if (track.sampleRate >= 1_000) add("Sample rate" to formatSampleRate(track.sampleRate))
    if (track.bitDepth > 0) add("Bit depth" to "${track.bitDepth}-bit")
    if (track.bitrate in 1_000..1_000_000_000) add("Bitrate" to "${track.bitrate / 1_000} kbps")
    if (track.durationSeconds > 0) add("Duration" to formatTrackDuration(track.durationSeconds))
    if (track.sizeBytes > 0) add("File size" to formatByteSize(track.sizeBytes))
}

internal fun formatSampleRate(sampleRate: Int): String = if (sampleRate % 1_000 == 0) "${sampleRate / 1_000} kHz" else String.format(Locale.US, "%.1f kHz", sampleRate / 1_000.0)
