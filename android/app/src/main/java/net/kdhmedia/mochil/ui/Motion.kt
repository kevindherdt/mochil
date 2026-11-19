package net.kdhmedia.mochil.ui

import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.AnimatedVisibilityScope
import androidx.compose.animation.EnterTransition
import androidx.compose.animation.ExitTransition
import androidx.compose.animation.expandVertically
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.shrinkVertically
import androidx.compose.foundation.lazy.LazyListState
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import net.kdhmedia.mochil.ui.theme.LocalReduceMotion

/** Shows or hides [content], without animating when animations are switched off. */
@Composable
internal fun MotionVisibility(visible: Boolean, modifier: Modifier = Modifier, content: @Composable AnimatedVisibilityScope.() -> Unit) {
    val still = LocalReduceMotion.current
    AnimatedVisibility(
        visible = visible,
        modifier = modifier,
        enter = if (still) EnterTransition.None else fadeIn() + expandVertically(),
        exit = if (still) ExitTransition.None else fadeOut() + shrinkVertically(),
        content = content,
    )
}

/** Scrolls a list to [index], jumping instead of gliding when animations are switched off. */
internal suspend fun LazyListState.scrollTo(index: Int, still: Boolean) {
    if (still) scrollToItem(index) else animateScrollToItem(index)
}
