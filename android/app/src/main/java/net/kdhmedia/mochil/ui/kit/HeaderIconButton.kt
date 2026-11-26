package net.kdhmedia.mochil.ui.kit

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.semantics.Role
import net.kdhmedia.mochil.ui.theme.MochilChrome
import net.kdhmedia.mochil.ui.theme.MochilRadius

/** Material's own disabled opacity for a control's content. */
private const val DISABLED_ALPHA = 0.38f

/**
 * The web's IconButton (web/src/ui/IconButton.module.css): a label-less square for the header's tools. Drawn at the control
 * height (32) with an 18 glyph so a cluster of three reads as a cluster, and a full 44 by 44 to touch. Idle it is muted; while
 * what it opens is on (an open search, an active filter) it is raised and in the accent. [description] is its name for TalkBack.
 */
@Composable
internal fun HeaderIconButton(
    icon: ImageVector,
    description: String,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    active: Boolean = false,
    enabled: Boolean = true,
) {
    val scheme = MaterialTheme.colorScheme
    val ink = when {
        !enabled -> scheme.onSurfaceVariant.copy(alpha = DISABLED_ALPHA)
        active -> scheme.primary
        else -> scheme.onSurfaceVariant
    }
    Box(
        modifier.size(MochilChrome.touchMin).clickable(enabled = enabled, role = Role.Button, onClick = onClick),
        contentAlignment = Alignment.Center,
    ) {
        Box(
            Modifier.size(MochilChrome.controlHeight).clip(RoundedCornerShape(MochilRadius.r8)).background(if (active) scheme.surfaceVariant else Color.Transparent),
            contentAlignment = Alignment.Center,
        ) {
            Icon(icon, contentDescription = description, Modifier.size(MochilChrome.iconRow), tint = ink)
        }
    }
}
