package net.kdhmedia.mochil.ui

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import net.kdhmedia.mochil.video.VideoTitleDetails

/** "2006 · 24m · Mystery, Horror": the year, then TMDB's runtime and genres when it has them. */
internal fun tvFactsLine(year: Int, details: VideoTitleDetails?): String = listOfNotNull(
    year.takeIf { it > 0 }?.toString(),
    details?.runtimeMinutes?.takeIf { it > 0 }?.let { if (it >= 60) "${it / 60}h ${it % 60}m".replace(" 0m", "") else "${it}m" },
    details?.genres?.takeIf { it.isNotEmpty() }?.joinToString(", "),
).joinToString(" · ")

internal fun tvScoreLabel(details: VideoTitleDetails?): String? = details?.rating?.takeIf { it > 0 }?.let { "★ ${"%.1f".format(java.util.Locale.US, it)}" }

/** The facts under a title: the line above, the age rating in a small outline and the TMDB score, in one row that wraps nowhere. */
@Composable
internal fun TvTitleFacts(line: String, details: VideoTitleDetails?, modifier: Modifier = Modifier) {
    val muted = MaterialTheme.colorScheme.onSurfaceVariant
    Row(modifier, horizontalArrangement = Arrangement.spacedBy(10.dp), verticalAlignment = Alignment.CenterVertically) {
        if (line.isNotBlank()) Text(line, style = MaterialTheme.typography.bodyMedium, color = muted, maxLines = 1)
        details?.certification?.takeIf(String::isNotBlank)?.let {
            Text(it, Modifier.border(BorderStroke(1.dp, muted), RoundedCornerShape(6.dp)).padding(horizontal = 6.dp, vertical = 1.dp), style = MaterialTheme.typography.labelMedium, color = muted, maxLines = 1)
        }
        tvScoreLabel(details)?.let { Text("$it TMDB", style = MaterialTheme.typography.bodyMedium, fontWeight = FontWeight.SemiBold, color = MaterialTheme.colorScheme.primary, maxLines = 1) }
    }
}
