package net.kdhmedia.mochil.video

import android.content.Context
import android.graphics.Color
import android.graphics.Typeface
import androidx.media3.ui.CaptionStyleCompat
import androidx.media3.common.util.UnstableApi

enum class SubtitlePreset(val label: String, val description: String) {
    Clean("Clean", "White text with a soft shadow and no box"),
    Cinema("Cinema", "Strong shadow for bright and dark scenes"),
    Anime("Outlined", "Crisp dark outline for animated detail"),
    HighContrast("High contrast", "Solid background for maximum readability"),
}

data class SubtitlePreferences(
    val preset: SubtitlePreset = SubtitlePreset.Clean,
    val textScale: Float = 1f,
    val backgroundOpacity: Float = .72f,
) {
    fun normalized() = copy(textScale = textScale.coerceIn(.75f, 1.5f), backgroundOpacity = backgroundOpacity.coerceIn(0f, 1f))

    @androidx.annotation.OptIn(UnstableApi::class)
    fun captionStyle(): CaptionStyleCompat {
        val safe = normalized()
        val background = when (safe.preset) {
            SubtitlePreset.Clean, SubtitlePreset.Anime -> Color.TRANSPARENT
            SubtitlePreset.Cinema -> alphaBlack(safe.backgroundOpacity * .22f)
            SubtitlePreset.HighContrast -> alphaBlack(safe.backgroundOpacity)
        }
        val edge = when (safe.preset) {
            SubtitlePreset.Anime -> CaptionStyleCompat.EDGE_TYPE_OUTLINE
            SubtitlePreset.Clean, SubtitlePreset.Cinema -> CaptionStyleCompat.EDGE_TYPE_DROP_SHADOW
            SubtitlePreset.HighContrast -> CaptionStyleCompat.EDGE_TYPE_NONE
        }
        return CaptionStyleCompat(Color.WHITE, background, Color.TRANSPARENT, edge, Color.BLACK, if (safe.preset == SubtitlePreset.Cinema) Typeface.DEFAULT_BOLD else Typeface.DEFAULT)
    }

    private fun alphaBlack(opacity: Float): Int = Color.argb((opacity.coerceIn(0f, 1f) * 255).toInt(), 0, 0, 0)
}

class SubtitlePreferencesStore(context: Context) {
    private val values = context.getSharedPreferences("subtitle_appearance", Context.MODE_PRIVATE)

    fun load() = SubtitlePreferences(
        preset = runCatching { SubtitlePreset.valueOf(values.getString("preset", SubtitlePreset.Clean.name)!!) }.getOrDefault(SubtitlePreset.Clean),
        textScale = values.getFloat("text_scale", 1f),
        backgroundOpacity = values.getFloat("background_opacity", .72f),
    ).normalized()

    fun save(preferences: SubtitlePreferences) {
        val safe = preferences.normalized()
        values.edit().putString("preset", safe.preset.name).putFloat("text_scale", safe.textScale).putFloat("background_opacity", safe.backgroundOpacity).apply()
    }
}
