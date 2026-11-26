package net.kdhmedia.mochil.ui

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.PaddingValues
import net.kdhmedia.mochil.ui.kit.MochilCard
import net.kdhmedia.mochil.ui.theme.MochilChrome
import net.kdhmedia.mochil.ui.theme.MochilRadius
import net.kdhmedia.mochil.ui.theme.MochilSpace
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.selection.selectable
import androidx.compose.foundation.horizontalScroll
import androidx.compose.material.icons.filled.BookmarkAdd
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.heading
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp

private val CardShape = RoundedCornerShape(16.dp)

/** A Home row as a card: the same quiet surface and hairline border the web cards use, so lists read as one design. */
@Composable
internal fun Modifier.homeCard(): Modifier = this
    .fillMaxWidth()
    .padding(horizontal = 16.dp, vertical = 2.dp)
    .clip(CardShape)
    .background(MaterialTheme.colorScheme.surfaceVariant.copy(alpha = .55f))
    .border(BorderStroke(1.dp, MaterialTheme.colorScheme.outline.copy(alpha = .22f)), CardShape)

/** A settings page's cards, one under the other. */
@Composable
internal fun SettingsSurface(modifier: Modifier = Modifier, content: @Composable ColumnScope.() -> Unit) {
    Column(modifier.fillMaxWidth().padding(bottom = MochilSpace.s24), verticalArrangement = androidx.compose.foundation.layout.Arrangement.spacedBy(MochilSpace.s24), content = content)
}

/**
 * One group of related settings on the web's card, with its small heading above it in the mock-up's muted caps. The rows
 * inside carry 8 of their own padding, so the card adds 8 more at the sides and they sit 16 in, like the web card.
 */
@Composable
internal fun SettingsCard(title: String?, content: @Composable ColumnScope.() -> Unit) {
    Column(Modifier.fillMaxWidth()) {
        if (title != null) Text(
            title.uppercase(),
            Modifier.padding(start = MochilSpace.s4, bottom = MochilSpace.s8).semantics { heading() },
            style = MaterialTheme.typography.labelMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
        MochilCard(contentPadding = PaddingValues(horizontal = MochilSpace.s8, vertical = MochilSpace.s12), content = content)
    }
}

/** A small label with a state in it: always words, never colour alone, so it reads on e-ink. */
@Composable
internal fun StatusPill(text: String, positive: Boolean = false, modifier: Modifier = Modifier) {
    val tone = if (positive) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurfaceVariant
    Text(
        text,
        modifier.clip(RoundedCornerShape(50)).background(tone.copy(alpha = .16f)).padding(horizontal = 10.dp, vertical = 3.dp),
        style = MaterialTheme.typography.labelMedium, fontWeight = FontWeight.SemiBold, color = tone,
    )
}

/** A cover card in a grid: the same bordered surface the web uses, so a cover and its title read as one object. */
@Composable
internal fun Modifier.gridCard(): Modifier = this
    .clip(RoundedCornerShape(MochilRadius.r12))
    .background(MaterialTheme.colorScheme.surface)
    .border(BorderStroke(MochilChrome.hairline, MaterialTheme.colorScheme.outlineVariant), RoundedCornerShape(MochilRadius.r12))

/** The small round "add to a collection" button on a cover. */
@Composable
internal fun CollectButton(label: String, onClick: () -> Unit, modifier: Modifier = Modifier) {
    androidx.compose.material3.Surface(
        onClick, modifier.size(32.dp), shape = androidx.compose.foundation.shape.CircleShape,
        color = MaterialTheme.colorScheme.surface.copy(alpha = .78f), contentColor = MaterialTheme.colorScheme.onSurface,
    ) {
        androidx.compose.foundation.layout.Box(Modifier.semantics { contentDescription = label }, contentAlignment = androidx.compose.ui.Alignment.Center) {
            androidx.compose.material3.Icon(androidx.compose.material.icons.Icons.Default.BookmarkAdd, null, Modifier.size(18.dp))
        }
    }
}
