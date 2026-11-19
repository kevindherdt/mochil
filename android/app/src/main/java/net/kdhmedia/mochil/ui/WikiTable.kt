package net.kdhmedia.mochil.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import net.kdhmedia.mochil.wiki.WikiMarkdownBlock

@Composable
internal fun WikiTable(block: WikiMarkdownBlock.Table, onWikiLink: (String) -> Unit) {
    val columnWidth = 176.dp
    Column(
        Modifier.horizontalScroll(rememberScrollState())
            .background(MaterialTheme.colorScheme.surfaceVariant.copy(alpha = .38f), MaterialTheme.shapes.small),
    ) {
        Row {
            block.headers.forEach { value -> WikiTableCell(value, columnWidth, onWikiLink, FontWeight.SemiBold) }
        }
        HorizontalDivider(color = MaterialTheme.colorScheme.outlineVariant)
        block.rows.forEach { row ->
            Row {
                block.headers.indices.forEach { column -> WikiTableCell(row.getOrElse(column) { "" }, columnWidth, onWikiLink) }
            }
        }
    }
}

@Composable
private fun WikiTableCell(value: String, width: Dp, onWikiLink: (String) -> Unit, weight: FontWeight? = null) {
    WikiInlineText(
        text = value,
        onWikiLink = onWikiLink,
        style = MaterialTheme.typography.bodyMedium,
        weight = weight,
        modifier = Modifier.width(width).padding(horizontal = 12.dp, vertical = 9.dp),
    )
}
