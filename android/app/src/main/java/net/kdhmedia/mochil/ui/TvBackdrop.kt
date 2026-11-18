package net.kdhmedia.mochil.ui

import androidx.compose.animation.Crossfade
import androidx.compose.animation.core.tween
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.MutableState
import androidx.compose.runtime.State
import androidx.compose.runtime.staticCompositionLocalOf
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.blur
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp
import net.kdhmedia.mochil.ui.theme.LocalReduceMotion

/** The art behind the whole screen, set by whichever screen has something focused; null outside the TV shell. */
internal val LocalTvBackdrop = staticCompositionLocalOf<MutableState<String>?> { null }

/** Shows [art] as the screen backdrop while this is composed, and clears it again when the screen goes away. */
@Composable
internal fun TvBackdropEffect(art: String) {
    val backdrop = LocalTvBackdrop.current
    DisposableEffect(art) {
        backdrop?.value = art
        onDispose { if (backdrop?.value == art) backdrop.value = "" }
    }
}

/** A wide still fills the screen; a tall poster becomes a soft blurred glow with the sharp poster set into the upper right. */
@Composable
internal fun TvBackdropLayer(art: State<String>, authorizationHeader: String?) {
    val background = MaterialTheme.colorScheme.background
    Box(Modifier.fillMaxSize()) {
        // A 400ms full-screen dissolve restarted on every D-pad step never settles (and smears on
        // e-ink), so reduced motion swaps the artwork instantly.
        val label: @Composable (String) -> Unit = { url ->
            when {
                url.isBlank() -> TvAmbientGlow()
                tvIsPosterArtwork(url) -> Box(Modifier.fillMaxSize()) {
                    AuthenticatedArtwork(url, authorizationHeader, "", Modifier.fillMaxSize().blur(48.dp), cornerRadius = 0.dp)
                    AuthenticatedArtwork(url, authorizationHeader, "", Modifier.align(Alignment.TopEnd).padding(end = 96.dp, top = 40.dp).fillMaxHeight(.62f).aspectRatio(2f / 3f), cornerRadius = 8.dp)
                }
                // Frames and TMDB backdrops are wide and fill the screen; album and show covers only lend their colours as a soft glow.
                "video-artwork" in url || "backdrop=1" in url -> TvWideArtBlock(url, authorizationHeader, background)
                else -> AuthenticatedArtwork(url, authorizationHeader, "", Modifier.fillMaxSize().blur(56.dp), cornerRadius = 0.dp)
            }
        }
        if (LocalReduceMotion.current) label(art.value) else Crossfade(art.value, animationSpec = tween(400), label = "tvBackdrop") { url -> label(url) }
        // Posters and covers still get the full-screen scrim; wide art carries its own fades (see TvWideArtBlock).
        if (art.value.isNotBlank() && !tvIsWideArtwork(art.value)) {
            Box(Modifier.fillMaxSize().background(background.copy(alpha = .3f)))
            Box(Modifier.fillMaxSize().background(Brush.horizontalGradient(0f to background.copy(alpha = .92f), .45f to background.copy(alpha = .5f), 1f to Color.Transparent)))
            Box(Modifier.fillMaxSize().background(Brush.verticalGradient(.3f to Color.Transparent, .8f to background.copy(alpha = .94f))))
        }
    }
}

internal fun tvIsWideArtwork(url: String): Boolean = !tvIsPosterArtwork(url) && ("video-artwork" in url || "backdrop=1" in url)

/** Wide art sits in the upper right, like Plex: it dissolves into the dark on its left and bottom edges, leaving the left and lower screen for text and rows. */
@Composable
private fun TvWideArtBlock(url: String, authorizationHeader: String?, background: Color) {
    Box(Modifier.fillMaxSize()) {
        Box(Modifier.align(Alignment.TopEnd).fillMaxWidth(.74f).fillMaxHeight(.8f)) {
            AuthenticatedArtwork(url, authorizationHeader, "", Modifier.fillMaxSize(), cornerRadius = 0.dp)
            Box(Modifier.fillMaxSize().background(Brush.horizontalGradient(0f to background, .38f to background.copy(alpha = .55f), .75f to Color.Transparent)))
            Box(Modifier.fillMaxSize().background(Brush.verticalGradient(.45f to Color.Transparent, 1f to background)))
        }
    }
}

/** Soft colour pools on the near-black screen for when nothing is focused or there is no art to borrow from. */
@Composable
private fun TvAmbientGlow() {
    val primary = MaterialTheme.colorScheme.primary
    val tertiary = MaterialTheme.colorScheme.tertiary
    Box(
        Modifier.fillMaxSize()
            .background(Brush.radialGradient(listOf(primary.copy(alpha = .30f), Color.Transparent), center = androidx.compose.ui.geometry.Offset(1500f, 120f), radius = 1100f))
            .background(Brush.radialGradient(listOf(tertiary.copy(alpha = .22f), Color.Transparent), center = androidx.compose.ui.geometry.Offset(200f, 1000f), radius = 1000f)),
    )
}
