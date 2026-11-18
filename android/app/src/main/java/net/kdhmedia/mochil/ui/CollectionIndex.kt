package net.kdhmedia.mochil.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.ExperimentalFoundationApi
import androidx.compose.foundation.combinedClickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.grid.items as gridItems
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.FolderOpen
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import net.kdhmedia.mochil.catalogue.LibrarySummary
import net.kdhmedia.mochil.catalogue.PublicationCollection
import net.kdhmedia.mochil.ui.theme.LocalEInkPalette

/**
 * The collections of one library, or of all of them: two stacked-cover tiles to a row, or flat rows. When [libraryOf] is given
 * each collection also says which library it lives in, with that library's icon and colour.
 */
@OptIn(ExperimentalFoundationApi::class)
@Composable
internal fun CollectionIndex(
    collections: List<PublicationCollection>,
    covers: Boolean,
    collectionCoverUrl: (PublicationCollection) -> String?,
    header: String?,
    onOpen: (PublicationCollection) -> Unit,
    libraryOf: ((PublicationCollection) -> LibrarySummary?)? = null,
    onRename: ((PublicationCollection, String) -> Unit)? = null,
    onDelete: ((PublicationCollection) -> Unit)? = null,
) {
    var managing by androidx.compose.runtime.remember { androidx.compose.runtime.mutableStateOf<PublicationCollection?>(null) }
    val manage = onRename != null && onDelete != null
    if (manage) managing?.let { CollectionManageDialogs(it, { managing = null }, onRename!!, onDelete!!) }
if (covers) androidx.compose.foundation.lazy.grid.LazyVerticalGrid(
        columns = androidx.compose.foundation.lazy.grid.GridCells.Fixed(2), contentPadding = androidx.compose.foundation.layout.PaddingValues(16.dp),
        horizontalArrangement = Arrangement.spacedBy(12.dp), verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        gridItems(collections, key = { it.id }) { collection ->
            val cover = collectionCoverUrl(collection)
            Column(
                Modifier.gridCard().combinedClickable(role = Role.Button, onClickLabel = "Open ${collection.name}", onLongClickLabel = "Manage ${collection.name}", onLongClick = if (manage) ({ managing = collection }) else null, onClick = { onOpen(collection) }).padding(12.dp),
                verticalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                // The first cover in front and, for several books, a stack behind it: the same tile the series use.
                Box(Modifier.fillMaxWidth().aspectRatio(0.75f), contentAlignment = Alignment.Center) {
                    Box(Modifier.fillMaxWidth(if (collection.count > 1) .88f else 1f)) {
                        if (cover == null) Box(Modifier.fillMaxWidth().aspectRatio(0.75f).clip(RoundedCornerShape(12.dp)).background(MaterialTheme.colorScheme.surfaceVariant), contentAlignment = Alignment.Center) {
                            Icon(Icons.Outlined.FolderOpen, null, Modifier.size(40.dp), tint = MaterialTheme.colorScheme.primary)
                        } else {
                            if (collection.count > 1) SeriesStack(cover, header, collection.name, Modifier.fillMaxWidth().aspectRatio(0.75f))
                            PublicationArtwork(cover, header, "Cover of ${collection.name}", Modifier.fillMaxWidth().aspectRatio(0.75f))
                        }
                        libraryOf?.invoke(collection)?.let { library ->
                            val monochrome = if (LocalEInkPalette.current) MaterialTheme.colorScheme.onSurface else null
                            val style = mediaKindStyle(library.kind, monochromeColor = monochrome)
                            Surface(
                                Modifier.align(Alignment.TopStart).padding(6.dp),
                                shape = RoundedCornerShape(8.dp),
                                color = if (LocalEInkPalette.current) MaterialTheme.colorScheme.surface else MaterialTheme.colorScheme.surface.copy(alpha = .85f),
                            ) {
                                Icon(style.icon, style.label, Modifier.padding(5.dp).size(16.dp), tint = style.color)
                            }
                        }
                        Surface(Modifier.align(Alignment.BottomEnd).padding(8.dp), shape = RoundedCornerShape(8.dp), color = MaterialTheme.colorScheme.surface, contentColor = MaterialTheme.colorScheme.onSurface) {
                            Text("${collection.count}", Modifier.padding(horizontal = 7.dp, vertical = 4.dp), style = MaterialTheme.typography.labelSmall)
                        }
                    }
                }
                Text(collection.name, style = MaterialTheme.typography.titleSmall, fontWeight = FontWeight.Medium, maxLines = 2, overflow = TextOverflow.Ellipsis)
                Text(listOfNotNull(libraryOf?.invoke(collection)?.name, "${collection.count} ${if (collection.count == 1) "book" else "books"}").joinToString(" · "), maxLines = 1, overflow = TextOverflow.Ellipsis, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
        }
    } else LazyColumn(Modifier.fillMaxSize()) {
        items(collections, key = PublicationCollection::id) { collection ->
            Column {
                Row(Modifier.fillMaxWidth().combinedClickable(role = Role.Button, onClickLabel = "Open ${collection.name}", onLongClickLabel = "Manage ${collection.name}", onLongClick = if (manage) ({ managing = collection }) else null, onClick = { onOpen(collection) }).padding(horizontal = 16.dp, vertical = 14.dp), verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(14.dp)) {
                    val cover = collectionCoverUrl(collection)
                    if (cover == null) Icon(Icons.Outlined.FolderOpen, null, Modifier.size(width = 46.dp, height = 46.dp), tint = MaterialTheme.colorScheme.primary)
                    else PublicationArtwork(cover, header, "Cover of ${collection.name}", Modifier.size(width = 46.dp, height = 69.dp).clip(RoundedCornerShape(6.dp)))
                    Column(Modifier.weight(1f)) {
                        Text(collection.name, style = MaterialTheme.typography.titleSmall, fontWeight = FontWeight.SemiBold, maxLines = 1, overflow = TextOverflow.Ellipsis)
                        Text(listOfNotNull(libraryOf?.invoke(collection)?.name, "${collection.count} ${if (collection.count == 1) "book" else "books"}").joinToString(" · "), maxLines = 1, overflow = TextOverflow.Ellipsis, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                    }
                }
                androidx.compose.material3.HorizontalDivider(Modifier.padding(start = 76.dp), color = MaterialTheme.colorScheme.outline.copy(alpha = .14f))
            }
        }
    }
}
