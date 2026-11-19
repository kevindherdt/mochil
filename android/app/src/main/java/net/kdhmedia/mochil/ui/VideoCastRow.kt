package net.kdhmedia.mochil.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.focusable
import androidx.compose.foundation.focusGroup
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Person
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusProperties
import androidx.compose.ui.focus.onFocusChanged
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import coil3.compose.AsyncImage
import coil3.network.NetworkHeaders
import coil3.network.httpHeaders
import coil3.request.ImageRequest
import net.kdhmedia.mochil.video.VideoTitleDetails

/** Portrait sizes the cast row draws at, shared with the page that has to fit this row into its band. */
internal val TvCastPortraitFull = 68.dp
internal val TvCastPortraitCompact = 42.dp

/**
 * Informational cast cards: focusable on TV, but not buttons without an actor destination.
 *
 * [compact] shrinks the portraits and drops the character line, which is what lets a show page keep the cast on
 * screen beside its episode rail without giving up the description.
 */
@Composable
internal fun VideoCastRow(
    details: VideoTitleDetails?,
    artwork: (String) -> String,
    auth: String?,
    television: Boolean = true,
    compact: Boolean = false,
    focusUp: FocusRequester? = null,
) {
    val cast = details?.cast.orEmpty()
    if (cast.isEmpty()) return
    val portrait = if (compact) TvCastPortraitCompact else TvCastPortraitFull
    val columnWidth = if (compact) 72.dp else 100.dp
    Column(
        Modifier.fillMaxWidth()
            .then(if (television) Modifier.focusGroup() else Modifier)
            // A cast card is a leaf: sending Up back into the shelf needs an explicit target, because the focus
            // search from here prefers the season pills above the rail.
            .then(if (focusUp != null) Modifier.focusProperties { up = focusUp } else Modifier),
        verticalArrangement = Arrangement.spacedBy(4.dp),
    ) {
        Text("Cast", style = MaterialTheme.typography.titleSmall)
        LazyRow(contentPadding = PaddingValues(horizontal = 8.dp, vertical = 5.dp), horizontalArrangement = Arrangement.spacedBy(if (compact) 8.dp else 12.dp)) {
            items(cast) { member ->
                var focused by remember { mutableStateOf(false) }
                Column(
                    Modifier.width(columnWidth).then(
                        if (television) {
                            Modifier.onFocusChanged { focused = it.isFocused }
                                .semantics(mergeDescendants = true) {}
                                .then(if (focusUp != null) Modifier.focusProperties { up = focusUp } else Modifier)
                                .focusable()
                        } else {
                            Modifier
                        },
                    ),
                    horizontalAlignment = Alignment.CenterHorizontally,
                ) {
                    val url = member.portraitPath.takeIf(String::isNotBlank)?.let(artwork).orEmpty()
                    val context = LocalContext.current
                    val model = remember(url, auth) {
                        val headers = NetworkHeaders.Builder().apply { if (auth != null) set("Authorization", auth) }.build()
                        ImageRequest.Builder(context).data(url).httpHeaders(headers).build()
                    }
                    Box(Modifier.size(portrait).then(if (television) Modifier.tvFocusVisual(focused, CircleShape, focusedScale = 1.03f) else Modifier).testTag("cast-focus-${member.name}")) {
                        Box(Modifier.fillMaxSize().testTag("cast-portrait-${member.name}").clip(CircleShape).background(MaterialTheme.colorScheme.surfaceVariant), contentAlignment = Alignment.Center) {
                            Icon(Icons.Default.Person, null, Modifier.size(if (compact) 24.dp else 40.dp), tint = MaterialTheme.colorScheme.onSurfaceVariant)
                            if (url.isNotBlank()) AsyncImage(model, null, Modifier.fillMaxSize(), contentScale = ContentScale.Crop)
                        }
                    }
                    Text(member.name, style = MaterialTheme.typography.labelMedium, maxLines = 1, overflow = TextOverflow.Ellipsis)
                    if (!compact && member.character.isNotBlank()) Text(member.character, style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant, maxLines = 1, overflow = TextOverflow.Ellipsis)
                }
            }
        }
    }
}
