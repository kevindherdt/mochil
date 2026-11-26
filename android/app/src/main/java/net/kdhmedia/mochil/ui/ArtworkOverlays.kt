package net.kdhmedia.mochil.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.luminance
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.semantics.ProgressBarRangeInfo
import androidx.compose.ui.semantics.progressBarRangeInfo
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.semantics.stateDescription
import androidx.compose.ui.unit.dp
import net.kdhmedia.mochil.ui.theme.LocalEInkPalette

/**
 * The marks the web draws on a card's artwork (web/src/ui/CoverImage.tsx): a kind glyph in the top-left, a plate or
 * badge in the top-right, a scrim with a thin bar along the foot, a green tick when the book is read. Kevin's ruling:
 * a card's kind is an icon on the artwork, never a pill of text under the author.
 *
 * They sit on translucent raised surfaces rather than on the page colour, so they read on any cover; 82% holds its
 * contrast against a worst-case white artwork, since contrast cannot depend on the image underneath.
 */
private val ArtworkInk = Color.White

@Composable
private fun plateBackground(): Color {
    // The web's --elev-rgb is an *artwork* overlay, always dark: black in dark and e-ink, a brown in light
    // (tokens.css). The theme's raised fill is pale in light themes, and white ink on it vanished (1.1-1.9:1).
    // E-ink takes black whatever the ground is: the web's --elev-rgb is 0 0 0 there too, and a brown plate would print mid-grey.
    val dark = LocalEInkPalette.current || MaterialTheme.colorScheme.background.luminance() < 0.5f
    return (if (dark) Color.Black else Color(0xFF6B5035)).copy(alpha = .82f)
}

/** The kind glyph, 24dp in the cover's top-left: what this card is, without a line of text. */
@Composable
internal fun ArtworkKindBadge(icon: ImageVector, description: String? = null, modifier: Modifier = Modifier) {
    Box(modifier.size(24.dp).clip(CircleShape).background(plateBackground()), contentAlignment = Alignment.Center) {
        Icon(icon, contentDescription = description, modifier = Modifier.size(13.dp), tint = ArtworkInk)
    }
}

/** A short plate in the cover's top-right: "Vol. 3", "1080p". */
@Composable
internal fun ArtworkPlate(text: String, modifier: Modifier = Modifier) {
    Box(modifier.heightIn(min = 24.dp).clip(RoundedCornerShape(50)).background(plateBackground()).padding(horizontal = 8.dp), contentAlignment = Alignment.Center) {
        Text(text, style = MaterialTheme.typography.labelSmall, color = ArtworkInk, maxLines = 1)
    }
}

/** A scrim along the foot of the artwork with a thin bar: how far through, with no percentage in words. */
@Composable
internal fun ArtworkProgressScrim(progress: () -> Float, stateDescription: String? = null, modifier: Modifier = Modifier) {
    ArtworkScrim(modifier, label = false) { ArtworkBar(progress, stateDescription) }
}

/** A scrim that names a format ("1080p · Direct Play") above the bar, the same way the web's video and music cards do. */
@Composable
internal fun ArtworkFormatStrip(label: String, progress: (() -> Float)?, modifier: Modifier = Modifier) {
    // A long label at 2x is ellipsised rather than cut mid-word.
    ArtworkScrim(modifier) {
        Text(label, style = MaterialTheme.typography.labelSmall, color = ArtworkInk, maxLines = 1, overflow = TextOverflow.Ellipsis)
        if (progress != null) {
            Box(Modifier.padding(top = 4.dp)) { ArtworkBar(progress, null) }
        }
    }
}

@Composable
private fun ArtworkScrim(modifier: Modifier = Modifier, label: Boolean = true, content: @Composable () -> Unit) {
    Box(
        modifier.fillMaxWidth()
            .background(Brush.verticalGradient(listOf(Color.Transparent, Color.Black.copy(alpha = .75f))))
            // The web's bar-only variant pads 8 all round; the labelled one needs 24 at the top for the gradient.
            .padding(start = 8.dp, end = 8.dp, top = if (label) 24.dp else 8.dp, bottom = 8.dp),
    ) { Column(verticalArrangement = Arrangement.Top) { content() } }
}

@Composable
private fun ArtworkBar(progress: () -> Float, stateDescription: String?) {
    val fraction = progress()
    Box(
        Modifier.fillMaxWidth().height(3.dp).clip(RoundedCornerShape(50))
            // The bar is the only reading state left on the card, so it has to be readable by TalkBack too.
            .semantics {
                progressBarRangeInfo = ProgressBarRangeInfo(fraction, 0f..1f)
                if (stateDescription != null) this.stateDescription = stateDescription
            }
            .background(ArtworkInk.copy(alpha = .38f)),
    ) {
        Box(Modifier.fillMaxWidth(fraction).fillMaxHeight().background(MaterialTheme.colorScheme.primary))
    }
}
