package net.kdhmedia.mochil.reader.epub

import androidx.compose.foundation.Canvas
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import org.readium.r2.navigator.preferences.Theme

/** Progress and learned reading-time presentation, independent from reader menus. */
@Composable
fun EpubPersistentProgress(
    progression: Double,
    chapterCount: Int,
    theme: Theme,
    eink: Boolean,
    modifier: Modifier = Modifier,
    statistics: ReadingStatistics = ReadingStatistics(),
    progress: ReadingProgress = ReadingProgress.Bar,
) {
    val showProgressBar = progress == ReadingProgress.Bar
    val bounded = progression.coerceIn(0.0, 1.0).toFloat()
    val surface = if (eink) theme.einkSurface else theme.progressSurface
    val content = if (eink) theme.einkContent else theme.progressContent
    Column(modifier.navigationBarsPadding().fillMaxWidth().background(Color.Transparent).padding(horizontal = 12.dp, vertical = if (eink) 7.dp else 5.dp)) {
        val labels = persistentReadingLabels(statistics, progress)
        if (progress != ReadingProgress.None) {
            Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                if (labels.isNotEmpty()) Text(labels.joinToString(" · "), color = content, style = MaterialTheme.typography.labelSmall, maxLines = 1, overflow = TextOverflow.Ellipsis)
                Text("${(bounded * 100).toInt()}%", color = content, style = MaterialTheme.typography.labelSmall)
            }
        }
        if (showProgressBar) Canvas(Modifier.fillMaxWidth().height(5.dp)) {
            val completedWidth = size.width * bounded
            drawRect(content, size = size.copy(width = completedWidth))
            if (chapterCount in 2..100) for (index in 1 until chapterCount) {
                val x = size.width * index / chapterCount
                if (x <= completedWidth) drawLine(surface, Offset(x, 0f), Offset(x, size.height), strokeWidth = if (eink) 2.dp.toPx() else 1.dp.toPx())
            }
        }
    }
}

internal val ReaderPreferences.hasPersistentFooter: Boolean
    get() = persistentFooterEnabled(progress)

internal fun persistentFooterEnabled(progress: ReadingProgress): Boolean = progress != ReadingProgress.None

/** Time is one readout: the time left in the chapter and in the book. */
internal fun persistentReadingLabels(statistics: ReadingStatistics, progress: ReadingProgress): List<String> =
    if (progress != ReadingProgress.Time) emptyList() else listOf(
        "Chapter ${readingDurationLabel(statistics.chapterRemainingMillis) ?: "learning pace"}",
        "Book ${readingDurationLabel(statistics.bookRemainingMillis) ?: "learning pace"}",
    )

@Composable
internal fun ReaderProgressLabel(progression: Double, statistics: ReadingStatistics, showChapter: Boolean = false) {
    val chapter = if (showChapter) readingDurationLabel(statistics.chapterRemainingMillis) else null
    val remaining = readingDurationLabel(statistics.bookRemainingMillis)
    Text(
        listOfNotNull(
            "${(progression * 100).toInt().coerceIn(0, 100)}%",
            chapter?.let { "$it left in chapter" },
            remaining?.let { "$it left" },
        ).joinToString(" · "),
        modifier = Modifier.navigationBarsPadding().padding(bottom = 10.dp).background(Color.Black.copy(alpha = 0.78f), MaterialTheme.shapes.large).padding(horizontal = 14.dp, vertical = 7.dp),
        color = Color.White,
        style = MaterialTheme.typography.labelLarge,
    )
}

@Composable
internal fun ReadingStatisticsPanel(statistics: ReadingStatistics) {
    Column(Modifier.fillMaxWidth().padding(horizontal = 24.dp, vertical = 18.dp), verticalArrangement = Arrangement.spacedBy(14.dp)) {
        Text("Reading statistics", style = MaterialTheme.typography.titleMedium)
        StatisticLine("Left in this chapter", readingDurationLabel(statistics.chapterRemainingMillis) ?: "Learning your pace…")
        StatisticLine("Left in this book", readingDurationLabel(statistics.bookRemainingMillis) ?: "Learning your pace…")
        StatisticLine("Read today", readingDurationLabel(statistics.todayReadingMillis) ?: "Less than a minute")
        StatisticLine("Total reading time", readingDurationLabel(statistics.totalReadingMillis) ?: "Less than a minute")
        StatisticLine("Reading speed", readingSpeedLabel(statistics.averagePositionMillis) ?: "Learning your pace…")
        StatisticLine("Average page/position", positionDurationLabel(statistics.averagePositionMillis) ?: "Learning your pace…")
        Text(
            "The estimate follows your usual pace: a burst of fast page turns, a jump through the book and any pause " +
                "over two minutes are left out, so skimming never makes you look faster than you read.",
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
    }
}

@Composable
private fun StatisticLine(label: String, value: String) {
    Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(16.dp), verticalAlignment = Alignment.CenterVertically) {
        Text(label, modifier = Modifier.weight(1f), style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
        Text(value, modifier = Modifier.weight(1f), style = MaterialTheme.typography.bodyMedium)
    }
}

private val Theme.progressSurface: Color get() = when (this) {
    Theme.LIGHT -> Color(0xFFFFF8F5)
    Theme.SEPIA -> Color(0xFFF4E6CF)
    Theme.DARK -> Color(0xFF201C1A)
}

private val Theme.progressContent: Color get() = when (this) {
    Theme.LIGHT -> Color(0xFF241A17)
    Theme.SEPIA -> Color(0xFF34251D)
    Theme.DARK -> Color(0xFFFFF8F4)
}

private val Theme.einkSurface: Color get() = if (this == Theme.DARK) Color.Black else Color.White
private val Theme.einkContent: Color get() = if (this == Theme.DARK) Color.White else Color.Black
