package net.kdhmedia.mochil.ui

import net.kdhmedia.mochil.ui.theme.TvSettingsTypography
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * TV Settings is read as a list, so it renders one step denser than the shelf ramp that the rest of the TV UI uses:
 * at the shelf sizes the category column, the group headings and a full changelog read as magnified.
 */
class TvSettingsTypeScaleTest {
    @Test
    fun settingsTypeIsSmallerThanTheShelfRampEverywhereItIsUsed() {
        // The shelf ramp's own values (TvTypography in Theme.kt) are private, so these are the specific steps that were
        // reported as huge, pinned just below them.
        assertTrue("Settings heading", TvSettingsTypography.headlineSmall.fontSize.value <= 26f)
        assertTrue("group headings", TvSettingsTypography.titleLarge.fontSize.value <= 20f)
        assertTrue("category labels", TvSettingsTypography.titleSmall.fontSize.value <= 16f)
        assertTrue("release-note bullets", TvSettingsTypography.bodyLarge.fontSize.value <= 16f)
        assertTrue("body copy", TvSettingsTypography.bodyMedium.fontSize.value <= 15f)
    }

    @Test
    fun everyStepKeepsRoomToBreathe() {
        listOf(
            TvSettingsTypography.headlineSmall,
            TvSettingsTypography.titleLarge,
            TvSettingsTypography.titleMedium,
            TvSettingsTypography.titleSmall,
            TvSettingsTypography.bodyLarge,
            TvSettingsTypography.bodyMedium,
            TvSettingsTypography.bodySmall,
            TvSettingsTypography.labelLarge,
            TvSettingsTypography.labelMedium,
            TvSettingsTypography.labelSmall,
        ).forEach { style ->
            assertTrue(
                "line height ${style.lineHeight} must clear the ${style.fontSize} type",
                style.lineHeight.value >= style.fontSize.value,
            )
        }
    }

    @Test
    fun theRampStaysInReadingOrder() {
        val ramp = listOf(
            TvSettingsTypography.headlineSmall.fontSize,
            TvSettingsTypography.titleLarge.fontSize,
            TvSettingsTypography.titleMedium.fontSize,
            TvSettingsTypography.titleSmall.fontSize,
            TvSettingsTypography.bodyMedium.fontSize,
            TvSettingsTypography.bodySmall.fontSize,
        ).map { it.value }
        assertEquals("headings must stay heavier than captions", ramp, ramp.sortedDescending())
        assertTrue("sizes are sp, not raw px", TvSettingsTypography.bodyMedium.fontSize.value > 5f)
    }
}
