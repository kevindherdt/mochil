package net.kdhmedia.mochil.ui

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.selection.selectable
import androidx.compose.foundation.horizontalScroll
import androidx.compose.material.icons.filled.BookmarkAdd
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.semantics.contentDescription
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

/** A settings tab's cards, one under the other. */
@Composable
internal fun SettingsSurface(modifier: Modifier = Modifier, content: @Composable ColumnScope.() -> Unit) {
    Column(modifier.fillMaxWidth().padding(bottom = 24.dp), verticalArrangement = androidx.compose.foundation.layout.Arrangement.spacedBy(14.dp), content = content)
}

/** One group of related settings on its own rounded card, with its heading above it, as the web groups them. */
@Composable
internal fun SettingsCard(title: String?, content: @Composable ColumnScope.() -> Unit) {
    Column(Modifier.fillMaxWidth().padding(top = 14.dp)) {
        if (title != null) Text(title, Modifier.padding(start = 6.dp, bottom = 6.dp), style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.SemiBold)
        Column(
            Modifier.fillMaxWidth().clip(CardShape).background(MaterialTheme.colorScheme.surfaceVariant.copy(alpha = .45f))
                .border(BorderStroke(1.dp, MaterialTheme.colorScheme.outline.copy(alpha = .22f)), CardShape)
                .padding(horizontal = 6.dp, vertical = 10.dp),
            content = content,
        )
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
    .clip(RoundedCornerShape(18.dp))
    .background(MaterialTheme.colorScheme.surfaceVariant.copy(alpha = .5f))
    .border(BorderStroke(1.dp, MaterialTheme.colorScheme.outline.copy(alpha = .22f)), RoundedCornerShape(18.dp))

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

/** A row of pill tabs in one rounded container, the same control Books uses for Series, Authors and All. */
@Composable
internal fun <T> PillTabs(choices: List<T>, selected: T, label: (T) -> String, onSelect: (T) -> Unit, modifier: Modifier = Modifier) {
    val shape = RoundedCornerShape(16.dp)
    androidx.compose.foundation.layout.Row(
        modifier.clip(shape).background(MaterialTheme.colorScheme.surfaceVariant.copy(alpha = .45f))
            .border(BorderStroke(1.dp, MaterialTheme.colorScheme.outline.copy(alpha = .22f)), shape).padding(4.dp)
            .androidx_horizontalScroll(),
        horizontalArrangement = androidx.compose.foundation.layout.Arrangement.spacedBy(2.dp),
    ) {
        choices.forEach { choice ->
            val on = choice == selected
            Text(
                label(choice),
                Modifier.clip(RoundedCornerShape(12.dp)).background(if (on) MaterialTheme.colorScheme.primaryContainer else Color.Transparent)
                    .androidx_selectable(on) { onSelect(choice) }.padding(horizontal = 12.dp, vertical = 8.dp),
                style = MaterialTheme.typography.titleSmall, fontWeight = if (on) FontWeight.SemiBold else FontWeight.Medium,
                color = if (on) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurfaceVariant, maxLines = 1,
            )
        }
    }
}

@Composable
private fun Modifier.androidx_horizontalScroll(): Modifier = this.then(Modifier.horizontalScroll(androidx.compose.foundation.rememberScrollState()))

private fun Modifier.androidx_selectable(on: Boolean, onClick: () -> Unit): Modifier =
    this.selectable(on, role = androidx.compose.ui.semantics.Role.Tab, onClick = onClick)
