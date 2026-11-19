package net.kdhmedia.mochil.ui

import android.content.ClipData
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.foundation.lazy.LazyListState
import androidx.compose.foundation.text.selection.SelectionContainer
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.IconButton
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.CheckBox
import androidx.compose.material.icons.outlined.CheckBoxOutlineBlank
import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.ClipEntry
import androidx.compose.ui.platform.LocalClipboard
import androidx.compose.ui.text.LinkAnnotation
import androidx.compose.ui.text.SpanStyle
import androidx.compose.ui.text.TextLinkStyles
import androidx.compose.ui.text.buildAnnotatedString
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextDecoration
import androidx.compose.ui.text.withStyle
import androidx.compose.ui.unit.dp
import androidx.compose.ui.layout.ContentScale
import net.kdhmedia.mochil.wiki.WikiEntry
import net.kdhmedia.mochil.wiki.WikiMarkdownBlock
import net.kdhmedia.mochil.wiki.WikiInlineStyle
import net.kdhmedia.mochil.wiki.WikiAssetIndex
import net.kdhmedia.mochil.wiki.parseWikiInline
import kotlinx.coroutines.launch

@Composable
internal fun WikiMarkdownView(
    blocks: List<WikiMarkdownBlock>,
    modifier: Modifier = Modifier,
    pageId: String = "",
    onWikiLink: (String) -> Unit = {},
    entries: List<WikiEntry> = emptyList(),
    pagePath: String = "",
    assetUrl: (WikiEntry) -> String = { "" },
    authorizationHeader: String? = null,
    backlinks: List<WikiEntry> = emptyList(),
    onBacklink: (WikiEntry) -> Unit = {},
    onToggleTask: ((Int, Boolean) -> Unit)? = null,
    scrollToBlock: Int? = null,
    onScrollComplete: () -> Unit = {},
) {
    val listState = rememberSaveable(pageId, saver = LazyListState.Saver) { LazyListState() }
    val imageIndex = remember(entries) { WikiAssetIndex(entries) }
    LaunchedEffect(scrollToBlock, blocks.size) {
        val target = scrollToBlock ?: return@LaunchedEffect
        if (target in blocks.indices) listState.scrollToItem(target)
        onScrollComplete()
    }
    SelectionContainer {
        LazyColumn(modifier, state = listState, verticalArrangement = Arrangement.spacedBy(12.dp), contentPadding = androidx.compose.foundation.layout.PaddingValues(vertical = 16.dp)) {
            itemsIndexed(blocks, key = { index, block -> "$index:${block::class.simpleName}" }) { _, block ->
                when (block) {
                    is WikiMarkdownBlock.Heading -> WikiInlineText(block.text, onWikiLink, when (block.level) {
                        1 -> MaterialTheme.typography.headlineMedium
                        2 -> MaterialTheme.typography.headlineSmall
                        else -> MaterialTheme.typography.titleLarge
                    }, weight = FontWeight.SemiBold)
                    is WikiMarkdownBlock.Paragraph -> WikiInlineText(block.text, onWikiLink, MaterialTheme.typography.bodyLarge)
                    is WikiMarkdownBlock.ListItem -> WikiListItem(block, onWikiLink, onToggleTask)
                    is WikiMarkdownBlock.Quote -> Row(
                        Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.spacedBy(12.dp),
                    ) {
                        androidx.compose.foundation.layout.Box(
                            Modifier.width(3.dp).height(24.dp).background(MaterialTheme.colorScheme.primary.copy(alpha = .65f), MaterialTheme.shapes.extraSmall),
                        )
                        WikiInlineText(block.text, onWikiLink, MaterialTheme.typography.bodyLarge.copy(color = MaterialTheme.colorScheme.onSurfaceVariant))
                    }
                    is WikiMarkdownBlock.Image -> {
                        val entry = remember(imageIndex, pagePath, block.target) { imageIndex.resolve(pagePath, block.target) }
                        val url = remember(entry, assetUrl) { entry?.let(assetUrl).orEmpty() }
                        if (entry != null && url.isNotBlank()) {
                            AuthenticatedArtwork(
                                url = url,
                                authorizationHeader = authorizationHeader,
                                title = block.description.ifBlank { entry.label },
                                modifier = Modifier.fillMaxWidth().heightIn(max = 520.dp)
                                    .background(MaterialTheme.colorScheme.surfaceVariant, MaterialTheme.shapes.medium),
                                cornerRadius = 12.dp,
                                contentScale = ContentScale.Fit,
                            )
                        } else Text(
                            "Image unavailable · ${block.description.ifBlank { block.target.substringAfterLast('/') }}",
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                            style = MaterialTheme.typography.bodySmall,
                        )
                    }
                    is WikiMarkdownBlock.Properties -> WikiProperties(block)
                    is WikiMarkdownBlock.Table -> WikiTable(block, onWikiLink)
                    WikiMarkdownBlock.Rule -> HorizontalDivider(color = MaterialTheme.colorScheme.outlineVariant)
                    is WikiMarkdownBlock.Code -> WikiCodeBlock(block)
                }
            }
            if (backlinks.isNotEmpty()) item(key = "wiki-backlinks-heading", contentType = "wiki-backlinks-heading") {
                Column(
                    Modifier.fillMaxWidth().padding(top = 12.dp),
                ) {
                    HorizontalDivider(color = MaterialTheme.colorScheme.outlineVariant)
                    Text(
                        "Linked from",
                        style = MaterialTheme.typography.labelMedium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        modifier = Modifier.padding(top = 10.dp, bottom = 2.dp),
                    )
                }
            }
            items(
                items = backlinks,
                key = { entry -> "wiki-backlink:${entry.id}:${entry.path}" },
                contentType = { "wiki-backlink" },
            ) { entry ->
                TextButton(onClick = { onBacklink(entry) }) {
                    Text(entry.label, maxLines = 1)
                }
            }
        }
    }
}

@Composable
private fun WikiProperties(block: WikiMarkdownBlock.Properties) {
    Column(
        Modifier.fillMaxWidth().background(MaterialTheme.colorScheme.surfaceVariant.copy(alpha = .55f), MaterialTheme.shapes.medium)
            .padding(horizontal = 14.dp, vertical = 10.dp),
        verticalArrangement = Arrangement.spacedBy(5.dp),
    ) {
        Text("Properties", style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
        block.fields.forEach { field ->
            Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                Text(field.name, Modifier.width(100.dp), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                Text(field.value.ifBlank { "—" }, Modifier.weight(1f), style = MaterialTheme.typography.bodySmall)
            }
        }
    }
}

@Composable
private fun WikiListItem(block: WikiMarkdownBlock.ListItem, onWikiLink: (String) -> Unit, onToggleTask: ((Int, Boolean) -> Unit)?) {
    Row(
        Modifier.fillMaxWidth().padding(start = (block.depth * 18).dp),
        horizontalArrangement = Arrangement.spacedBy(10.dp),
    ) {
        if (block.checked != null) {
            if (onToggleTask != null && block.taskOrdinal >= 0) IconButton(
                onClick = { onToggleTask(block.taskOrdinal, !block.checked) },
            ) {
                Icon(
                    if (block.checked) Icons.Outlined.CheckBox else Icons.Outlined.CheckBoxOutlineBlank,
                    contentDescription = if (block.checked) "Mark task incomplete" else "Mark task complete",
                    tint = if (block.checked) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurfaceVariant,
                )
            } else Icon(
                if (block.checked) Icons.Outlined.CheckBox else Icons.Outlined.CheckBoxOutlineBlank,
                contentDescription = if (block.checked) "Completed task" else "Open task",
                tint = if (block.checked) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurfaceVariant,
            )
        } else Text(block.marker, color = MaterialTheme.colorScheme.onSurfaceVariant)
        WikiInlineText(block.text, onWikiLink, MaterialTheme.typography.bodyLarge)
    }
}

@Composable
internal fun WikiInlineText(
    text: String,
    onWikiLink: (String) -> Unit,
    style: androidx.compose.ui.text.TextStyle,
    modifier: Modifier = Modifier,
    weight: FontWeight? = null,
) {
    val color = MaterialTheme.colorScheme.primary
    val codeBackground = MaterialTheme.colorScheme.surfaceVariant
    val annotated = remember(text, color, codeBackground, onWikiLink) { buildAnnotatedString {
        parseWikiInline(text).forEach { segment ->
            when (segment.style) {
                WikiInlineStyle.Plain -> append(segment.text)
                WikiInlineStyle.Bold -> withStyle(SpanStyle(fontWeight = FontWeight.Bold)) { append(segment.text) }
                WikiInlineStyle.Italic -> withStyle(SpanStyle(fontStyle = FontStyle.Italic)) { append(segment.text) }
                WikiInlineStyle.Strikethrough -> withStyle(SpanStyle(textDecoration = TextDecoration.LineThrough)) { append(segment.text) }
                WikiInlineStyle.Code -> withStyle(SpanStyle(fontFamily = FontFamily.Monospace, background = codeBackground)) { append(segment.text) }
                WikiInlineStyle.Link -> {
                    pushLink(LinkAnnotation.Url(segment.target.orEmpty(), TextLinkStyles(SpanStyle(color = color, textDecoration = TextDecoration.Underline))))
                    append(segment.text)
                    pop()
                }
                WikiInlineStyle.WikiLink -> {
                    val target = segment.target.orEmpty()
                    pushLink(LinkAnnotation.Clickable(target, TextLinkStyles(SpanStyle(color = color, textDecoration = TextDecoration.Underline))) { onWikiLink(target) })
                    append(segment.text)
                    pop()
                }
            }
        }
    } }
    Text(annotated, modifier = modifier, style = style, fontWeight = weight)
}

@Composable
private fun WikiCodeBlock(block: WikiMarkdownBlock.Code) {
    val clipboard = LocalClipboard.current
    val scope = rememberCoroutineScope()
    Column(Modifier.fillMaxWidth().background(MaterialTheme.colorScheme.surfaceVariant, MaterialTheme.shapes.medium).padding(12.dp)) {
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
            Text(block.language.ifBlank { "Code" }, style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
            TextButton(onClick = { scope.launch { clipboard.setClipEntry(ClipEntry(ClipData.newPlainText("Code", block.text))) } }) { Text("Copy") }
        }
        Text(block.text, fontFamily = FontFamily.Monospace, style = MaterialTheme.typography.bodyMedium)
    }
}
