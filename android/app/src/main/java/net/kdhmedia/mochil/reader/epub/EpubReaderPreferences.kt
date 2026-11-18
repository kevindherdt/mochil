package net.kdhmedia.mochil.reader.epub

import android.content.Context
import org.readium.r2.navigator.epub.EpubPreferences
import org.readium.r2.navigator.preferences.Color
import org.readium.r2.navigator.preferences.Theme
import org.readium.r2.navigator.preferences.FontFamily
import org.readium.r2.shared.ExperimentalReadiumApi
import net.kdhmedia.mochil.catalogue.ClientDeviceClass
import net.kdhmedia.mochil.catalogue.detectClientDeviceClass

enum class ReaderFont(val label: String) {
    Publisher("Publisher font"), Serif("Serif"), SansSerif("Sans serif"), Monospace("Monospace"), OpenDyslexic("OpenDyslexic"), Accessible("Accessible"), Custom("Custom")
}

/**
 * Darker, heavier text for screens that show thin strokes faintly (e-ink). On draws text in pure black
 * on pure white (or white on black) and thickens each stroke by [stroke] pixels. On is the strength the
 * old Medium had; the former Light, Medium and Strong all load as On.
 */
enum class TextSharpness(val label: String, val stroke: Double) {
    Off("off", 0.0), On("on", 0.2);

    fun next(): TextSharpness = entries[(ordinal + 1) % entries.size]

    companion object {
        fun fromSaved(name: String?): TextSharpness? = when (name) {
            null -> null
            "Off" -> Off
            "On", "Light", "Medium", "Strong" -> On
            else -> Off
        }
    }
}

/** What stays on the page while reading: nothing, a progress bar with its percentage, or the percentage with chapter and book time left. */
enum class ReadingProgress(val label: String) {
    None("none"), Bar("bar"), Time("time");

    fun next(): ReadingProgress = entries[(ordinal + 1) % entries.size]

    companion object {
        /** The nearest value for the three separate switches this replaced: any time readout wins, since Time keeps the percentage too. */
        fun fromLegacy(bar: Boolean, chapterTime: Boolean, bookTime: Boolean): ReadingProgress = when {
            chapterTime || bookTime -> Time
            bar -> Bar
            else -> None
        }
    }
}

/** Which parts of the page turn it when tapped; a tap in the middle always opens the menu. */
enum class TapZones(val label: String) {
    Off("off"), Edges("edges"), OneHand("one hand");

    fun next(): TapZones = entries[(ordinal + 1) % entries.size]
}

data class ReaderPreferences(
    val fontSize: Double = 1.0,
    val theme: Theme = Theme.LIGHT,
    val scroll: Boolean = false,
    val font: ReaderFont = ReaderFont.Publisher,
    val customFontId: String? = null,
    val progress: ReadingProgress = ReadingProgress.None,
    val showSystemStatusBar: Boolean = false,
    val sharpness: TextSharpness = TextSharpness.Off,
    val tapZones: TapZones = TapZones.Edges,
) {
    @OptIn(ExperimentalReadiumApi::class)
    fun toReadium(customFontFamily: String? = null) = EpubPreferences(
        fontSize = fontSize,
        theme = theme,
        scroll = scroll,
        textColor = sharpColor(text = true),
        backgroundColor = sharpColor(text = false),
        publisherStyles = font == ReaderFont.Publisher,
        fontFamily = when (font) {
            ReaderFont.Publisher -> null
            ReaderFont.Serif -> FontFamily.SERIF
            ReaderFont.SansSerif -> FontFamily.SANS_SERIF
            ReaderFont.Monospace -> FontFamily.MONOSPACE
            ReaderFont.OpenDyslexic -> FontFamily.OPEN_DYSLEXIC
            ReaderFont.Accessible -> FontFamily.ACCESSIBLE_DFA
            ReaderFont.Custom -> customFontFamily?.let(::FontFamily)
        },
    )

    private fun sharpColor(text: Boolean): Color? =
        sharpTextColors(sharpness, dark = theme == Theme.DARK, sepia = theme == Theme.SEPIA)?.let { Color(if (text) it.first else it.second) }
}

/** Text and background (ARGB) for sharp text: pure black on white, or white on black. Null when colours stay the theme's own (Off, or sepia). */
internal fun sharpTextColors(sharpness: TextSharpness, dark: Boolean, sepia: Boolean): Pair<Int, Int>? = when {
    sharpness == TextSharpness.Off || sepia -> null
    dark -> 0xFFFFFFFF.toInt() to 0xFF000000.toInt()
    else -> 0xFF000000.toInt() to 0xFFFFFFFF.toInt()
}

/** [eInk] decides only the default of a setting nobody has saved yet; it never overrides a saved choice. */
class EpubReaderPreferencesStore(context: Context, eInk: Boolean? = null) {
    private val eInkDevice by lazy { eInk ?: runCatching { detectClientDeviceClass(context) == ClientDeviceClass.EInk }.getOrDefault(false) }
    private val preferences = context.getSharedPreferences("epub_reader_preferences", Context.MODE_PRIVATE)

    fun update(transform: (ReaderPreferences) -> ReaderPreferences): ReaderPreferences = synchronized(updateLock) {
        transform(load()).also(::save)
    }

    private companion object { val updateLock = Any() }

    fun load() = ReaderPreferences(
        fontSize = preferences.getFloat("font_size", 1f).toDouble().coerceIn(0.7, 2.0),
        theme = runCatching { Theme.valueOf(preferences.getString("theme", Theme.LIGHT.name)!!) }.getOrDefault(Theme.LIGHT),
        scroll = preferences.getBoolean("scroll", false),
        font = runCatching { ReaderFont.valueOf(preferences.getString("font", ReaderFont.Publisher.name)!!) }.getOrDefault(ReaderFont.Publisher),
        customFontId = preferences.getString("custom_font_id", null),
        progress = preferences.getString("reading_progress", null)?.let { saved -> ReadingProgress.entries.firstOrNull { it.name == saved } }
            ?: ReadingProgress.fromLegacy(
                preferences.getBoolean("show_progress_bar", false),
                preferences.getBoolean("show_chapter_time_remaining", false),
                preferences.getBoolean("show_book_time_remaining", false),
            ),
        showSystemStatusBar = preferences.getBoolean("show_system_status_bar", false),
        sharpness = TextSharpness.fromSaved(preferences.getString("text_sharpness", null)) ?: if (eInkDevice) TextSharpness.On else TextSharpness.Off,
        tapZones = runCatching { TapZones.valueOf(preferences.getString("tap_zones", TapZones.Edges.name)!!) }.getOrDefault(TapZones.Edges),
    )

    private fun save(value: ReaderPreferences) {
        preferences.edit()
            .putFloat("font_size", value.fontSize.toFloat())
            .putString("theme", value.theme.name)
            .putBoolean("scroll", value.scroll)
            .putString("font", value.font.name)
            .putString("custom_font_id", value.customFontId)
            .putString("reading_progress", value.progress.name)
            .remove("show_progress_bar").remove("show_chapter_time_remaining").remove("show_book_time_remaining")
            .putBoolean("show_system_status_bar", value.showSystemStatusBar)
            .putString("text_sharpness", value.sharpness.name)
            .putString("tap_zones", value.tapZones.name)
            .apply()
    }
}
