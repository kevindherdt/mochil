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

// Mochil' own four. Espresso is the default and keeps a light twin (Oat) so the device
// light/dark switch still means something; Kiln and Paper deliberately have ONE scheme
// each, so they look the same whatever the device reports. Hexes match theme-tokens.css on the web.
private val Espresso = dark(0xFFF09A72, 0xFF211D1A, 0xFF2A2521, 0xFF352F2A, muted = 0xFFC0B3A6)
private val Oat = light(0xFFBD5232, 0xFFF1E8DC, 0xFFF8F0E6, 0xFFE6DACB)
private val Kiln = dark(0xFFE0A458, 0xFF26262B, 0xFF303037, 0xFF3A3A43, 0xFFEDE7DE, 0xFFB0A79B)
private val Paper = light(0xFF1A1A18, 0xFFF7F7F5, 0xFFFFFFFF, 0xFFEDEDE9, 0xFF1A1A18, 0xFF5D5D57)
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
    val picked: ColorScheme = when (chosen.palette) {
        AppPalette.Espresso -> if (isDark || cinema) Espresso else Oat
        AppPalette.Kiln -> Kiln
        AppPalette.Paper -> Paper
        AppPalette.EInk -> if (isDark || cinema) EInkDark else EInkLight
    }
    val colors = if (cinema) picked.asCinema(if (chosen.palette == AppPalette.Kiln) Color(0xFFF0A35A) else null) else picked
    CompositionLocalProvider(
        LocalReduceMotion provides chosen.reduceMotion,
        LocalEInkPalette provides (chosen.palette == AppPalette.EInk),
    ) {
        MaterialTheme(colorScheme = colors, typography = if (cinema) TvTypography else Typography()) {
            // Every theme but e-ink wears a faint glow of its own accent, like the web (Kiln adds amber); it sits over the screen and never takes touches.
            if (glow && chosen.palette != AppPalette.EInk) Box(Modifier.fillMaxSize()) { content(); ThemeGlow(isDark, if (chosen.palette == AppPalette.Kiln) Color(0xFFF0A35A) else null) } else content()
        }
    }
}

/** On a TV the room is dim and the artwork carries the colour, so the base goes near-black and keeps only the theme's accent. */
private fun ColorScheme.asCinema(accentOverride: Color? = null): ColorScheme {
    val primary = accentOverride ?: this.primary
    val base = Color(0xFF0C0D12)
    val card = Color(0xFF171921)
    val raised = Color(0xFF232632)
    // Every palette needs its own tertiary: leaving it unset made all ten fall back to
    // Material's baseline pink (#EFB8C8), which is what the backdrop wash was painting.
    val wash = accentOverride ?: primary.copy(alpha = .55f).compositeOver(base)
    return copy(
        background = base, surface = card, surfaceVariant = raised, secondaryContainer = raised,
        tertiary = wash, onTertiary = base,
        primaryContainer = primary.copy(alpha = .22f).compositeOver(raised),
        onBackground = Color(0xFFF3F4F8), onSurface = Color(0xFFF3F4F8), onPrimaryContainer = Color(0xFFFFFFFF),
        onSurfaceVariant = Color(0xFFADB1C2), outline = Color(0xFFADB1C2),
        primary = if (primary.luminance() < .35f) primary.copy(alpha = 1f).compositeOver(Color.White.copy(alpha = .35f)) else primary,
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
    AppPalette.Espresso -> if (dark) Espresso else Oat
    AppPalette.Kiln -> Kiln
    AppPalette.Paper -> Paper
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
