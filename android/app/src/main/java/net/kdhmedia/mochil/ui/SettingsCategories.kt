package net.kdhmedia.mochil.ui

/**
 * The pages behind the Settings hub, in the order people look for them. [description] is the one muted line under the
 * name on the hub (web/src/features/settings/screens/SettingsScreen.tsx has the same shape); [iconId] is a Lucide glyph
 * from ui/kit/LucideIcons. Saved by name, so a renamed or dropped page reads back as "the hub" instead of crashing.
 */
internal enum class SettingsCategory(val label: String, val description: String, val iconId: String) {
    Server("Server", "Address, account and app updates.", "server"),
    Appearance("Appearance", "Theme and light or dark mode.", "palette"),
    Libraries("Libraries", "Which libraries this device shows.", "library-big"),
    Sources("Sources", "Other catalogues: OPDS and WebDAV.", "cloud"),
    Downloads("Downloads", "Where offline copies are kept.", "download"),
    Reading("Reading", "Reader defaults, page turns, read aloud.", "book-open-text"),
    Playback("Playback", "Streaming quality, ReplayGain, subtitles.", "monitor-play"),
    Privacy("Privacy", "Annotations, diagnostics, crash reports.", "shield-check"),
    About("About", "Version and the notices this app owes.", "info");

    companion object {
        fun fromName(name: String?): SettingsCategory? = entries.firstOrNull { it.name == name }
    }
}
