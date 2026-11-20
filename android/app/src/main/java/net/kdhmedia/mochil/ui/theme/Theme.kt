package net.kdhmedia.mochil.ui.theme

import android.os.Build
import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.ColorScheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.dynamicDarkColorScheme
import androidx.compose.material3.dynamicLightColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.material3.Typography
import androidx.compose.runtime.Composable
import androidx.compose.runtime.staticCompositionLocalOf
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.remember
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.drawWithCache
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.lerp
import androidx.compose.ui.graphics.compositeOver
import androidx.compose.ui.unit.sp
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.graphics.luminance
import androidx.compose.ui.platform.LocalContext

private fun dark(primary: Long, background: Long, surface: Long, raised: Long, text: Long = 0xFFF4EEE6, muted: Long = 0xFFB8AEA3): ColorScheme {
    val accent = Color(primary)
    val foreground = Color(text)
    val raisedSurface = Color(raised)
    return darkColorScheme(
        primary = accent,
        onPrimary = Color(background),
        primaryContainer = accent.copy(alpha = .24f).compositeOver(raisedSurface),
        onPrimaryContainer = foreground,
        secondaryContainer = raisedSurface,
        onSecondaryContainer = foreground,
        background = Color(background),
        onBackground = foreground,
        surface = Color(surface),
        onSurface = foreground,
        surfaceVariant = raisedSurface,
        onSurfaceVariant = Color(muted),
        outline = Color(muted),
    )
}

private fun light(primary: Long, background: Long, surface: Long, raised: Long, text: Long = 0xFF241F1A, muted: Long = 0xFF72675D): ColorScheme {
    val accent = Color(primary)
    val foreground = Color(text)
    val raisedSurface = Color(raised)
    return lightColorScheme(
        primary = accent,
        onPrimary = Color.White,
        primaryContainer = accent.copy(alpha = .14f).compositeOver(raisedSurface),
        onPrimaryContainer = foreground,
        secondaryContainer = raisedSurface,
        onSecondaryContainer = foreground,
        background = Color(background),
        onBackground = foreground,
        surface = Color(surface),
        onSurface = foreground,
        surfaceVariant = raisedSurface,
        onSurfaceVariant = Color(muted),
        outline = Color(muted),
    )
}

// Mochil' own palettes. Ember (warm) is the default and Tide (cool) is its counterpart;
// both keep a light and a dark scheme so the device's switch still means something.
// Paper deliberately has ONE scheme, so it looks the same whatever the device reports.
// Hexes match theme-tokens.css on the web.
private val Ember = dark(0xFFFF7A34, 0xFF1A1512, 0xFF251E18, 0xFF382C24, 0xFFF7EFE2, 0xFFBFAE99)
private val EmberLight = light(0xFFB8420A, 0xFFFDF1E3, 0xFFFFFAF3, 0xFFF6E2CD, 0xFF241608, 0xFF6F5540)
private val Tide = dark(0xFF2FD6D0, 0xFF0D1618, 0xFF142023, 0xFF1C2B2F, 0xFFE9F6F7, 0xFFA3BCC1)
private val TideLight = light(0xFF0A7278, 0xFFEAF6F6, 0xFFF9FDFD, 0xFFD8EEEF, 0xFF0D2426, 0xFF4D6F73)
private val Paper = light(0xFF1A1A18, 0xFFF7F7F5, 0xFFFFFFFF, 0xFFEDEDE9, 0xFF1A1A18, 0xFF5D5D57)
// The mark's own colours (field #ff2e7e, pack #fff3e9, lid #8c003b), bolder than the rest on
// purpose: Ember and Tide lean into chroma where the old palettes stayed muted.
private val MagentaCream = dark(0xFFFF2E7E, 0xFF15121A, 0xFF1E1A24, 0xFF282230, 0xFFFFF3E9, 0xFFB8A8AE)
private val MagentaCreamLight = light(0xFFC81A5E, 0xFFFFF3E9, 0xFFFFFAF5, 0xFFFFE8DA, 0xFF2A0F1A, 0xFF7A5A60)
private val EInkDark = dark(0xFFFFFFFF, 0xFF000000, 0xFF000000, 0xFF171717, 0xFFFFFFFF, 0xFFC9C9C9)
private val EInkLight = light(0xFF111111, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFEEEEEE, 0xFF111111, 0xFF4A4A4A)

/** Whether animations are switched off (the default on e-ink screens). Screens use [MotionVisibility] and [scrollTo] instead of animating directly. */
val LocalReduceMotion = staticCompositionLocalOf { false }
val LocalEInkPalette = staticCompositionLocalOf { false }

@Composable
fun MochilTheme(selection: AppThemeSelection? = null, glow: Boolean = true, cinema: Boolean = false, content: @Composable () -> Unit) {
    val context = LocalContext.current
    val chosen = selection ?: remember(context) { AppThemePreferences(context).load() }
    val isDark = when (chosen.mode) { AppThemeMode.System -> isSystemInDarkTheme(); AppThemeMode.Dark -> true; AppThemeMode.Light -> false }
    val picked = paletteScheme(chosen.palette, isDark || cinema)
    val colors = if (cinema) picked.asCinema() else picked
    CompositionLocalProvider(
        LocalReduceMotion provides chosen.reduceMotion,
        LocalEInkPalette provides (chosen.palette == AppPalette.EInk),
    ) {
        MaterialTheme(colorScheme = colors, typography = if (cinema) TvTypography else Typography()) {
            // Every theme but e-ink wears a faint glow of its own accent, like the web; it sits over the screen and never takes touches.
            // The Box is unconditional on purpose: the glow used to be added by wrapping the content, so
            // crossing the e-ink boundary moved the content to a different call site, disposed the whole
            // app shell and dropped its rememberSaveable state. Changing the palette must not rebuild the
            // screen, only recolour it.
            Box(Modifier.fillMaxSize()) {
                content()
                if (glow && chosen.palette != AppPalette.EInk) ThemeGlow(isDark, null)
            }
        }
    }
}

/** On a TV the room is dim and the artwork carries the colour, so the base goes near-black and keeps only the theme's accent. */
/** The scheme a palette resolves to before the TV treatment. Also the seam the TV tests assert on. */
internal fun paletteScheme(palette: AppPalette, dark: Boolean): ColorScheme = when (palette) {
    AppPalette.Ember -> if (dark) Ember else EmberLight
    AppPalette.Tide -> if (dark) Tide else TideLight
    AppPalette.Paper -> Paper
    AppPalette.MagentaCream -> if (dark) MagentaCream else MagentaCreamLight
    AppPalette.EInk -> if (dark) EInkDark else EInkLight
}

internal fun ColorScheme.asCinema(): ColorScheme {
    // The ten-foot base used to be one fixed navy (#0C0D12) whatever theme was picked, so on a TV
    // every palette differed only in its accent and the screen never looked like the theme at all.
    // It now keeps the palette's own hue and pushes it down to near-black: a TV is watched in a dark
    // room and the artwork should carry the colour, but the page still belongs to the theme.
    // A light palette (Paper, e-ink) is pushed much further, because a light page at ten feet is
    // simply blinding and nothing on screen can be read against it.
    val dark = if (background.luminance() < .18f) background else lerp(background, Color.Black, .84f)
    val base = lerp(dark, Color.Black, .48f)
    // Cards and rails step up from the base rather than from the phone's surface, so the steps stay
    // in the same hue and keep the same gaps as the old fixed values did.
    val card = lerp(base, Color.White, .045f)
    val raised = lerp(base, Color.White, .11f)
    val primary = this.primary
    val ink = lerp(base, Color.White, .94f)
    val muted = lerp(base, Color.White, .64f)
    // Every palette needs its own tertiary: leaving it unset made all ten fall back to
    // Material's baseline pink (#EFB8C8), which is what the backdrop wash was painting.
    val wash = primary.copy(alpha = .55f).compositeOver(base)
    return copy(
        background = base, surface = card, surfaceVariant = raised, secondaryContainer = raised,
        tertiary = wash, onTertiary = base,
        primaryContainer = primary.copy(alpha = .22f).compositeOver(raised),
        onBackground = ink, onSurface = ink, onPrimaryContainer = Color.White,
        onSurfaceVariant = muted, outline = muted,
        // A dark accent disappears against a dark base, so lift it toward white once here rather
        // than overriding it per palette, which had drifted out of step with the palette list.
        // This used to composite an *opaque* accent over a translucent white, which is just the
        // accent again, so the lift never happened: Paper's near-black accent stayed invisible on a
        // TV and its buttons read as blank.
        primary = if (primary.luminance() < .35f) lerp(primary, Color.White, .5f) else primary,
        onPrimary = base,
    )
}

@Composable
private fun ThemeGlow(dark: Boolean, secondary: Color?) {
    val accent = MaterialTheme.colorScheme.primary
    val amber = secondary ?: accent
    Box(Modifier.fillMaxSize().drawWithCache {
        val first = Brush.radialGradient(listOf(accent.copy(alpha = if (dark) .16f else .10f), Color.Transparent), center = Offset(size.width * .15f, size.height * .08f), radius = size.maxDimension * .7f)
        val second = Brush.radialGradient(listOf(amber.copy(alpha = if (dark) .12f else .08f), Color.Transparent), center = Offset(size.width * .88f, size.height * .92f), radius = size.maxDimension * .65f)
        onDrawWithContent { drawRect(first); drawRect(second) }
    })
}

/** The colour a theme family is known by, for the little dot beside its name in Settings. */
fun AppPalette.swatch(dark: Boolean): Color = when (this) {
    AppPalette.Ember -> if (dark) Ember else EmberLight
    AppPalette.Tide -> if (dark) Tide else TideLight
    AppPalette.Paper -> Paper
    AppPalette.MagentaCream -> if (dark) MagentaCream else MagentaCreamLight
    AppPalette.EInk -> if (dark) EInkDark else EInkLight
}.primary

/**
 * The 10-foot type scale. Material's defaults are sized for a phone held at arm's length: at three
 * metres a 14sp caption is roughly 3.4mm of cap height, which is below the readable floor. Every TV
 * screen inherits this through MaterialTheme, so the ramp lives here rather than per screen.
 */
private val TvTypography = Typography(
    displaySmall = TextStyle(fontSize = 44.sp, lineHeight = 46.sp, fontWeight = FontWeight.SemiBold, letterSpacing = (-0.5).sp),
    headlineMedium = TextStyle(fontSize = 36.sp, lineHeight = 40.sp, fontWeight = FontWeight.SemiBold),
    headlineSmall = TextStyle(fontSize = 32.sp, lineHeight = 36.sp, fontWeight = FontWeight.SemiBold),
    titleLarge = TextStyle(fontSize = 26.sp, lineHeight = 30.sp, fontWeight = FontWeight.Medium),
    titleMedium = TextStyle(fontSize = 21.sp, lineHeight = 26.sp, fontWeight = FontWeight.Medium),
    titleSmall = TextStyle(fontSize = 18.sp, lineHeight = 24.sp, fontWeight = FontWeight.Medium),
    bodyLarge = TextStyle(fontSize = 20.sp, lineHeight = 28.sp),
    bodyMedium = TextStyle(fontSize = 18.sp, lineHeight = 26.sp),
    bodySmall = TextStyle(fontSize = 15.sp, lineHeight = 21.sp),
    labelLarge = TextStyle(fontSize = 16.sp, lineHeight = 20.sp, fontWeight = FontWeight.Medium),
    labelMedium = TextStyle(fontSize = 15.sp, lineHeight = 19.sp, fontWeight = FontWeight.Medium, letterSpacing = 0.4.sp),
    labelSmall = TextStyle(fontSize = 13.sp, lineHeight = 17.sp, fontWeight = FontWeight.Medium, letterSpacing = 0.5.sp),
)

/**
 * Settings is a dense list of labels and options, not a shelf read across the room: at the shelf ramp a category list,
 * its group headings and a changelog full of bullets read as magnified and a long release-note list filled the panel.
 * Same family one step denser, applied by wrapping TV Settings in `MaterialTheme(typography = ...)` so every control in
 * it — categories, chips, buttons, release notes — shrinks together instead of drifting out of proportion.
 */
internal val TvSettingsTypography = Typography(
    displaySmall = TvTypography.displaySmall,
    headlineMedium = TvTypography.headlineMedium.copy(fontSize = 28.sp, lineHeight = 32.sp),
    headlineSmall = TvTypography.headlineSmall.copy(fontSize = 26.sp, lineHeight = 30.sp),
    titleLarge = TvTypography.titleLarge.copy(fontSize = 20.sp, lineHeight = 24.sp),
    titleMedium = TvTypography.titleMedium.copy(fontSize = 17.sp, lineHeight = 22.sp),
    titleSmall = TvTypography.titleSmall.copy(fontSize = 15.sp, lineHeight = 20.sp),
    bodyLarge = TvTypography.bodyLarge.copy(fontSize = 16.sp, lineHeight = 22.sp),
    bodyMedium = TvTypography.bodyMedium.copy(fontSize = 15.sp, lineHeight = 21.sp),
    bodySmall = TvTypography.bodySmall.copy(fontSize = 13.sp, lineHeight = 18.sp),
    labelLarge = TvTypography.labelLarge.copy(fontSize = 14.sp, lineHeight = 18.sp),
    labelMedium = TvTypography.labelMedium.copy(fontSize = 13.sp, lineHeight = 17.sp),
    labelSmall = TvTypography.labelSmall.copy(fontSize = 12.sp, lineHeight = 16.sp),
)
