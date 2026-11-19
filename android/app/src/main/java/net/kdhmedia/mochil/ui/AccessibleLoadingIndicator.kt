package net.kdhmedia.mochil.ui

import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.size
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.semantics.LiveRegionMode
import androidx.compose.ui.semantics.ProgressBarRangeInfo
import androidx.compose.ui.semantics.clearAndSetSemantics
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.liveRegion
import androidx.compose.ui.semantics.progressBarRangeInfo
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import net.kdhmedia.mochil.ui.theme.LocalReduceMotion

/**
 * Quiet visually, but names indefinite network work for TalkBack and switch access.
 *
 * The default indeterminate [CircularProgressIndicator] spins forever via its own internal
 * infinite transition. On e-ink that is a continuous full-panel repaint for as long as a screen
 * is loading, so with "Reduce animations" on we show a static full ring instead (same still-vs-
 * animated switch as `.shimmer()` in Shimmer.kt).
 */
@Composable
internal fun AccessibleLoadingIndicator(
    label: String,
    modifier: Modifier = Modifier,
    size: Dp = 32.dp,
) {
    val reduceMotion = LocalReduceMotion.current
    Box(
        modifier.clearAndSetSemantics {
            contentDescription = label
            liveRegion = LiveRegionMode.Polite
            progressBarRangeInfo = ProgressBarRangeInfo.Indeterminate
        },
        contentAlignment = Alignment.Center,
    ) {
        val strokeWidth = if (size < 30.dp) 2.dp else 4.dp
        if (reduceMotion) {
            CircularProgressIndicator(progress = { 1f }, modifier = Modifier.size(size), strokeWidth = strokeWidth)
        } else {
            CircularProgressIndicator(Modifier.size(size), strokeWidth = strokeWidth)
        }
    }
}
