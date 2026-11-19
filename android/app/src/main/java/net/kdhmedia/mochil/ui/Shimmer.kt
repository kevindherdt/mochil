package net.kdhmedia.mochil.ui

import androidx.compose.animation.core.LinearEasing
import androidx.compose.animation.core.RepeatMode
import androidx.compose.animation.core.animateFloat
import androidx.compose.animation.core.infiniteRepeatable
import androidx.compose.animation.core.rememberInfiniteTransition
import androidx.compose.animation.core.tween
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.compositeOver
import androidx.compose.ui.semantics.clearAndSetSemantics
import androidx.compose.ui.unit.dp
import net.kdhmedia.mochil.ui.theme.LocalReduceMotion

/**
 * A quiet moving highlight over a placeholder tile. With "Reduce animations" on (the default on e-ink) it is a plain
 * tile that never animates, so loading states stay calm and cost nothing.
 */
@Composable
internal fun Modifier.shimmer(): Modifier {
    val base = MaterialTheme.colorScheme.surfaceVariant
    if (LocalReduceMotion.current) return background(base)
    val highlight = MaterialTheme.colorScheme.onSurface.copy(alpha = .08f)
    val progress by rememberInfiniteTransition(label = "shimmer").animateFloat(
        initialValue = -1f, targetValue = 2f,
        animationSpec = infiniteRepeatable(tween(1_300, easing = LinearEasing), RepeatMode.Restart), label = "shimmer-offset",
    )
    return background(Brush.linearGradient(
        colors = listOf(base, highlight.compositeOver(base), base),
        start = androidx.compose.ui.geometry.Offset(progress * 600f, 0f),
        end = androidx.compose.ui.geometry.Offset(progress * 600f + 400f, 160f),
    ))
}

/** Placeholder rows in the shape of Home cards, shown while Home loads. */
@Composable
internal fun HomeSkeleton(rows: Int = 4, modifier: Modifier = Modifier) {
    Column(modifier.fillMaxWidth().clearAndSetSemantics { }, verticalArrangement = Arrangement.spacedBy(8.dp)) {
        Box(Modifier.padding(start = 20.dp, top = 18.dp, bottom = 4.dp).size(width = 160.dp, height = 24.dp).clip(RoundedCornerShape(8.dp)).shimmer())
        repeat(rows) {
            Row(Modifier.homeCard(), horizontalArrangement = Arrangement.spacedBy(14.dp), verticalAlignment = Alignment.CenterVertically) {
                Box(Modifier.size(width = 40.dp, height = 56.dp).clip(RoundedCornerShape(8.dp)).shimmer())
                Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    Box(Modifier.fillMaxWidth(.7f).height(14.dp).clip(RoundedCornerShape(6.dp)).shimmer())
                    Box(Modifier.fillMaxWidth(.35f).height(12.dp).clip(RoundedCornerShape(6.dp)).shimmer())
                }
            }
        }
    }
}
