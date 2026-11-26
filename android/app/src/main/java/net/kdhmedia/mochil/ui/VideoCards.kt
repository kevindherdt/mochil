package net.kdhmedia.mochil.ui

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import net.kdhmedia.mochil.video.VideoItem
import net.kdhmedia.mochil.video.VideoShow

/** What a title or episode shows beside its picture: how far along, in one short phrase. */
internal data class VideoCaption(val text: String, val fraction: Float?, val done: Boolean)

internal fun VideoShow.caption() = VideoCaption(
    text = if (completed) "Played" else if (watchedCount > 0 || inProgressCount > 0) watchLabel else "$count ${if (count == 1) "episode" else "episodes"}",
    fraction = if (completed) 1f else if (watchedCount > 0 || inProgressCount > 0) watchFraction else null,
    done = completed,
)

/** A film or episode: the same two lines as a show (what it is, then where you are) so movies and series read alike. */
internal fun VideoItem.caption(): VideoCaption {
    val length = durationSeconds.takeIf { it > 0 }?.let(::videoLengthLabel)
    val text = when {
        !playable -> "Unsupported on this server"
        finished -> "Played"
        resumeSeconds > 0 -> "${(resumeSeconds / 60).toInt()} min watched"
        episode > 0 -> listOfNotNull("Episode $episode", length).joinToString(" · ")
        else -> listOfNotNull(year.takeIf { it > 0 }?.toString(), length).joinToString(" · ")
    }
    return VideoCaption(text, if (finished) 1f else if (resumeSeconds > 0) progressFraction else null, finished)
}

/**
 * The strip the web prints on a film's artwork: the real probe's resolution, and "Direct Play" only when the server says the
 * file plays as it is (web/src/features/library/api/media.ts `resolutionLabel`). Never invented, so a film without a probe
 * carries no strip at all.
 */
internal fun videoFormatLabel(height: Int?, directPlay: Boolean): String? =
    resolutionLabel(height)?.let { label -> if (directPlay) "$label · Direct Play" else label }

/** The usual ladder, rounded down: an 800-line encode is "720p" in every interface a person has used. */
internal fun resolutionLabel(height: Int?): String? {
    if (height == null || height <= 0) return null
    return when {
        height >= 2000 -> "4K"
        height >= 1000 -> "1080p"
        height >= 700 -> "720p"
        height >= 400 -> "480p"
        else -> "${height}p"
    }
}

internal fun videoLengthLabel(seconds: Double): String {
    val minutes = (seconds / 60).toInt()
    return if (minutes >= 60) "${minutes / 60} h ${(minutes % 60).toString().padStart(2, '0')} min" else "$minutes min"
}

/**
 * One cover tile for every kind of video: picture, title (always two lines tall), a progress line (always there, empty when
 * unstarted) and a caption. Fixed slots keep every tile in a row the same height.
 */
@Composable
internal fun VideoCard(art: @Composable (Modifier) -> Unit, aspect: Float, title: String, caption: VideoCaption, formatLabel: String? = null, modifier: Modifier = Modifier) {
    // The card carries its own surface, like a book's and an album's, so every family is one object (web parity).
    Column(modifier.gridCard().padding(8.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
        Box(Modifier.fillMaxWidth().aspectRatio(aspect).clip(RoundedCornerShape(8.dp))) {
            art(Modifier.fillMaxSize())
            // The strip replaces the plain bar while it is set, as on the web: the format is what a film has to say about itself.
            // A finished film keeps the words and loses the bar: the tick and "Played" already say it.
            if (formatLabel != null) ArtworkFormatStrip(formatLabel, caption.fraction?.takeIf { !caption.done }?.let { fraction -> { fraction } }, Modifier.align(Alignment.BottomCenter))
            else if (!caption.done) caption.fraction?.takeIf { it > 0f }?.let { ArtworkProgressScrim({ it }, modifier = Modifier.align(Alignment.BottomCenter)) }
            if (caption.done) BookFinishedBadge(Modifier.align(Alignment.BottomStart).padding(8.dp))
        }
        Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
            Text(title, style = MaterialTheme.typography.titleSmall, fontWeight = FontWeight.SemiBold, minLines = 2, maxLines = 2, overflow = TextOverflow.Ellipsis)
            Text(caption.text, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant, minLines = 1, maxLines = 1, overflow = TextOverflow.Ellipsis)
        }
    }
}

/**
 * One row in the list layout: a small picture, the title, one line of what it is, and the percentage or a tick at the right edge,
 * with a hairline underneath. The same flat row the books use, so every library's list reads alike.
 */
@Composable
internal fun VideoListRow(art: @Composable () -> Unit, artWidth: Dp, artHeight: Dp, title: String, caption: VideoCaption, modifier: Modifier = Modifier) {
    Column {
        Row(modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 7.dp), verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(12.dp)) {
            Box(Modifier.size(artWidth, artHeight).clip(RoundedCornerShape(5.dp))) {
                art()
                caption.fraction?.let { ThinProgress({ it }, Modifier.fillMaxWidth().height(3.dp).align(Alignment.BottomCenter)) }
            }
            Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
                Text(title, style = MaterialTheme.typography.titleSmall, fontWeight = FontWeight.SemiBold, maxLines = 2, overflow = TextOverflow.Ellipsis)
                Text(caption.text, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant, maxLines = 1, overflow = TextOverflow.Ellipsis)
            }
            val label = if (caption.done) "✓" else caption.fraction?.takeIf { it > 0f }?.let { "${(it * 100).toInt()}%" }
            if (label != null) Text(label, style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.primary, fontWeight = FontWeight.SemiBold)
        }
        HorizontalDivider(Modifier.padding(start = artWidth + 28.dp), color = MaterialTheme.colorScheme.outline.copy(alpha = .14f))
    }
}
