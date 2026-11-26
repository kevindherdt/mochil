package net.kdhmedia.mochil.ui.kit

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.semantics.Role
import net.kdhmedia.mochil.ui.theme.MochilChrome
import net.kdhmedia.mochil.ui.theme.MochilRadius
import net.kdhmedia.mochil.ui.theme.MochilSpace

/**
 * The web's Card (web/src/ui/Card.module.css): the surface colour, a hairline in the line colour, radius 12, 16 inside.
 * Flat: the web's card shadow is dropped on the phone (the "flat ground" ruling), the line carries the edge.
 * With [onClick] the whole card is one button, so TalkBack reads its text as a single control.
 */
@Composable
internal fun MochilCard(
    modifier: Modifier = Modifier,
    onClick: (() -> Unit)? = null,
    contentPadding: PaddingValues = PaddingValues(MochilSpace.s16),
    content: @Composable ColumnScope.() -> Unit,
) {
    val shape = RoundedCornerShape(MochilRadius.r12)
    val frame = modifier
        .fillMaxWidth()
        .clip(shape)
        .background(MaterialTheme.colorScheme.surface)
        .border(MochilChrome.hairline, MaterialTheme.colorScheme.outlineVariant, shape)
    Column((if (onClick != null) frame.clickable(role = Role.Button, onClick = onClick) else frame).padding(contentPadding), content = content)
}
