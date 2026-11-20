package net.kdhmedia.mochil.ui

import androidx.compose.foundation.ExperimentalFoundationApi
import androidx.compose.foundation.combinedClickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Headphones
import androidx.compose.material3.Icon
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.Dp
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import net.kdhmedia.mochil.catalogue.BookSummary
import net.kdhmedia.mochil.catalogue.ReadingStatus
import net.kdhmedia.mochil.catalogue.readingStatus

/** "3" for book three of a series, "3.5" for a novella between three and four, null when the book has no number. */
internal fun seriesNumberLabel(index: Double?): String? = index?.let { if (it % 1.0 == 0.0) it.toLong().toString() else it.toString() }

private fun BookSummary.progressLabel(): String? = when {
    finished -> "✓"
    readingFraction != null -> "${(readingFraction * 100).toInt()}%"
    else -> null
}

/**
 * One book in the plain list of every book: a small cover with a thin progress line, the title, the author, and the percentage read
 * at the right edge. Dense on purpose, so many books fit on one screen; a hairline sets the rows apart.
 */
@OptIn(ExperimentalFoundationApi::class)
@Composable
internal fun BookListRow(book: BookSummary, coverUrl: String?, header: String?, onRead: (BookSummary) -> Unit, onCollect: (BookSummary) -> Unit) {
    val fraction = if (book.finished) 1f else book.readingFraction?.toFloat()
    val status = book.readingStatus().takeIf { it == ReadingStatus.OnHold || it == ReadingStatus.Dropped }
    Column {
        Row(
            Modifier.fillMaxWidth()
                .combinedClickable(onClickLabel = "Read ${book.title}", role = Role.Button, onClick = { onRead(book) }, onLongClickLabel = "Add ${book.title} to collection", onLongClick = { onCollect(book) })
                .padding(horizontal = 16.dp, vertical = 7.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            ListCover(book, coverUrl, header, 32.dp, 48.dp, fraction)
            Column(Modifier.weight(1f)) {
                Text(book.title, style = MaterialTheme.typography.titleSmall, fontWeight = FontWeight.SemiBold, maxLines = 1, overflow = TextOverflow.Ellipsis)
                val detail = listOfNotNull(book.authors.joinToString().takeIf(String::isNotBlank), status?.label).joinToString(" · ")
                if (detail.isNotBlank()) Text(detail, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant, maxLines = 1, overflow = TextOverflow.Ellipsis)
            }
            if (book.readAlong) Icon(Icons.Default.Headphones, "Read-along", Modifier.size(14.dp), tint = MaterialTheme.colorScheme.primary)
            book.progressLabel()?.let { Text(it, style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.primary, fontWeight = FontWeight.SemiBold) }
        }
        HorizontalDivider(Modifier.padding(start = 60.dp), color = MaterialTheme.colorScheme.outline.copy(alpha = .14f))
    }
}

/**
 * One book inside a series or under an author: the number in the series leads the title, then the author (left out when it is the
 * same for the whole group) and the percentage read. The order is the point here, so the number is always the first thing you read.
 */
@OptIn(ExperimentalFoundationApi::class)
@Composable
internal fun BookTableRow(book: BookSummary, coverUrl: String?, header: String?, showAuthors: Boolean, onRead: (BookSummary) -> Unit, onCollect: (BookSummary) -> Unit) {
    val fraction = if (book.finished) 1f else book.readingFraction?.toFloat()
    val title = listOfNotNull(seriesNumberLabel(book.seriesIndex), book.title).joinToString(" · ")
    Column {
        Row(
            Modifier.fillMaxWidth()
                .combinedClickable(onClickLabel = "Read ${book.title}", role = Role.Button, onClick = { onRead(book) }, onLongClickLabel = "Add ${book.title} to collection", onLongClick = { onCollect(book) })
                .padding(horizontal = 16.dp, vertical = 6.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            ListCover(book, coverUrl, header, 28.dp, 42.dp, fraction)
            Text(title, Modifier.weight(if (showAuthors) 1.5f else 1f), style = MaterialTheme.typography.titleSmall, fontWeight = FontWeight.SemiBold, maxLines = 2, overflow = TextOverflow.Ellipsis)
            if (showAuthors) Text(book.authors.joinToString(), Modifier.weight(1f), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant, maxLines = 2, overflow = TextOverflow.Ellipsis)
            Text(book.progressLabel().orEmpty(), Modifier.width(38.dp), style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.primary, fontWeight = FontWeight.SemiBold, textAlign = TextAlign.End)
        }
        HorizontalDivider(Modifier.padding(start = 54.dp), color = MaterialTheme.colorScheme.outline.copy(alpha = .14f))
    }
}

@Composable
private fun ListCover(book: BookSummary, coverUrl: String?, header: String?, width: Dp, height: Dp, fraction: Float?) {
    Box(Modifier.size(width, height).clip(RoundedCornerShape(4.dp))) {
        PublicationArtwork(coverUrl, header, "Cover of ${book.title}", Modifier.size(width, height))
        if (fraction != null) ThinProgress(progress = { fraction }, modifier = Modifier.fillMaxWidth().height(3.dp).align(Alignment.BottomCenter))
    }
}
