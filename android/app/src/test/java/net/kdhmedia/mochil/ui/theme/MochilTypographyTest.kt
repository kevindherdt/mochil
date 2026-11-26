package net.kdhmedia.mochil.ui.theme

import androidx.compose.material3.MaterialTheme
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.sp
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotEquals
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35])
class MochilTypographyTest {
    @get:Rule val compose = createComposeRule()

    @Test fun `body and label sizes come from the token scale`() {
        val type = mochilTypography()
        assertEquals(MochilFontSize.s16, type.bodyLarge.fontSize)
        assertEquals(MochilFontSize.s14, type.bodyMedium.fontSize)
        assertEquals(MochilFontSize.s12, type.bodySmall.fontSize)
        assertEquals(MochilFontSize.s12, type.labelMedium.fontSize)
        assertEquals(MochilFontSize.s11, type.labelSmall.fontSize)
    }

    @Test fun `chips and tabs are semibold like the web`() {
        val type = mochilTypography()
        assertEquals(FontWeight.SemiBold, type.labelMedium.fontWeight)
        assertEquals(FontWeight.SemiBold, type.labelLarge.fontWeight)
    }

    @Test fun `interface text is Source Sans and large headings are Newsreader`() {
        val type = mochilTypography()
        assertEquals(MochilFonts.ui, type.bodyMedium.fontFamily)
        assertEquals(MochilFonts.ui, type.titleSmall.fontFamily)
        assertEquals(MochilFonts.display, type.headlineSmall.fontFamily)
        assertEquals(MochilFonts.display, type.displaySmall.fontFamily)
        assertNotEquals(MochilFonts.ui, MochilFonts.display)
    }

    @Test fun `the phone theme uses the Mochil type`() {
        var phoneBody = 0.sp
        compose.setContent {
            MochilTheme(AppThemeSelection(), glow = false) { phoneBody = MaterialTheme.typography.bodyMedium.fontSize }
        }
        compose.waitForIdle()
        assertEquals(MochilFontSize.s14, phoneBody)
    }

    @Test fun `cinema keeps the ten-foot body size`() {
        var tvBody = 0.sp
        compose.setContent {
            MochilTheme(AppThemeSelection(), glow = false, cinema = true) { tvBody = MaterialTheme.typography.bodyMedium.fontSize }
        }
        compose.waitForIdle()
        assertEquals(18.sp, tvBody)
    }
}
