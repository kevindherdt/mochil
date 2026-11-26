package net.kdhmedia.mochil.ui

import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.outlined.MenuBook
import androidx.compose.material.icons.outlined.AutoAwesome
import androidx.compose.material.icons.outlined.AutoStories
import androidx.compose.material.icons.outlined.ChatBubbleOutline
import androidx.compose.material.icons.outlined.Dashboard
import androidx.compose.material.icons.outlined.Description
import androidx.compose.material.icons.outlined.Headphones
import androidx.compose.material.icons.outlined.Mic
import androidx.compose.material.icons.outlined.Movie
import androidx.compose.material.icons.outlined.MusicNote
import androidx.compose.material.icons.outlined.Tv
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.luminance
import androidx.compose.ui.graphics.vector.ImageVector
import net.kdhmedia.mochil.catalogue.LibraryKind

/** One icon and one strong colour per kind of library, the same on every screen and on the web, so a row says what it is at a glance.
 * [color] is tuned for dark surfaces; [onLight] is its dark twin, used when the surface behind it is light, because a pastel
 * icon on a near-white surface measures barely above 1:1 and reads as an empty pill. */
internal data class MediaKindStyle(val icon: ImageVector, val color: Color, val onLight: Color, val label: String)

/** The colour to paint this kind's icon with on a surface of [background]. */
internal fun MediaKindStyle.tint(background: Color): Color =
    if (background.luminance() > 0.5f) onLight else color

/** Parses [kind] through [LibraryKind] so this never drifts from the canonical kind -> category mapping
 * that TvNavRail and TvConnectedApp also use; manga and comics keep their own icon/colour here (that is
 * about per-kind identification, not category grouping) while still agreeing they are both publications. */
internal fun mediaKindStyle(kind: String, monochromeColor: Color? = null): MediaKindStyle {
    val style = when (LibraryKind.fromWire(kind)) {
    LibraryKind.LightNovels -> MediaKindStyle(Icons.Outlined.AutoStories, Color(0xFFFF8FC0), Color(0xFF8E1F55), "Light novel")
    LibraryKind.Manga -> MediaKindStyle(Icons.Outlined.Dashboard, Color(0xFFFFA45C), Color(0xFF8A3F00), "Manga")
    LibraryKind.Comics -> MediaKindStyle(Icons.Outlined.ChatBubbleOutline, Color(0xFFFF7A7A), Color(0xFF8E1B24), "Comic")
    LibraryKind.Movies -> MediaKindStyle(Icons.Outlined.Movie, Color(0xFFFFD666), Color(0xFF6B5200), "Movie")
    LibraryKind.Tv -> MediaKindStyle(Icons.Outlined.Tv, Color(0xFF6FDC9A), Color(0xFF005B33), "TV show")
    LibraryKind.Anime -> MediaKindStyle(Icons.Outlined.AutoAwesome, Color(0xFFB69CFB), Color(0xFF46248F), "Anime")
    LibraryKind.Music -> MediaKindStyle(Icons.Outlined.MusicNote, Color(0xFF5CC8F2), Color(0xFF00536F), "Music")
    LibraryKind.Audiobooks -> MediaKindStyle(Icons.Outlined.Headphones, Color(0xFF4FD1B8), Color(0xFF00544A), "Audiobook")
    LibraryKind.Podcasts -> MediaKindStyle(Icons.Outlined.Mic, Color(0xFFC3E94A), Color(0xFF3F5200), "Podcast")
    LibraryKind.Vault -> MediaKindStyle(Icons.Outlined.Description, Color(0xFF9AA8BD), Color(0xFF3E4A5C), "Notes")
    LibraryKind.Books, LibraryKind.Mixed, LibraryKind.Calibre -> MediaKindStyle(Icons.AutoMirrored.Outlined.MenuBook, Color(0xFF7FB2F7), Color(0xFF12457F), "Book")
    }
    return monochromeColor?.let { style.copy(color = it, onLight = it) } ?: style
}
