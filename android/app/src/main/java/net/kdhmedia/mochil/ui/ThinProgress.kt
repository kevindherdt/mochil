package net.kdhmedia.mochil.ui

import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier

/**
 * A progress line that ends where the progress ends. The stock indicator also draws a small dot at the far end of its track, which
 * read as a stray mark spilling past the bar on every card.
 */
@Composable
internal fun ThinProgress(progress: () -> Float, modifier: Modifier = Modifier) {
    LinearProgressIndicator(progress = progress, modifier = modifier, drawStopIndicator = {})
}
