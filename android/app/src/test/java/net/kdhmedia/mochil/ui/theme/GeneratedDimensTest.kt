package net.kdhmedia.mochil.ui.theme

import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import org.junit.Assert.assertEquals
import org.junit.Test

/** The sizes the phone reads come from design/tokens.json through the generator; these pin the ones screens lean on. */
class GeneratedDimensTest {
    @Test fun `space scale is the token scale`() {
        assertEquals(
            listOf(4.dp, 8.dp, 12.dp, 16.dp, 24.dp, 32.dp, 48.dp),
            listOf(MochilSpace.s4, MochilSpace.s8, MochilSpace.s12, MochilSpace.s16, MochilSpace.s24, MochilSpace.s32, MochilSpace.s48),
        )
    }

    @Test fun `font sizes keep the half sizes the web uses`() {
        assertEquals(11.5.sp, MochilFontSize.s11)
        assertEquals(12.5.sp, MochilFontSize.s12)
        assertEquals(14.sp, MochilFontSize.s14)
        assertEquals(16.sp, MochilFontSize.s16)
    }

    @Test fun `chrome dimensions are the phone ones`() {
        assertEquals(54.dp, MochilChrome.dockHeightPhone)
        assertEquals(56.dp, MochilChrome.headerHeightPhone)
        assertEquals(96.dp, MochilChrome.playerMiniHeight)
        assertEquals(44.dp, MochilChrome.touchMin)
        assertEquals(24.dp, MochilChrome.pillHeight)
        assertEquals(32.dp, MochilChrome.controlHeight)
    }

    @Test fun `line weights and row icons are tokens too`() {
        assertEquals(1.dp, MochilChrome.hairline)
        assertEquals(2.dp, MochilChrome.ring)
        assertEquals(18.dp, MochilChrome.iconRow)
        assertEquals(16.dp, MochilChrome.iconSmall)
    }

    @Test fun `radius scale ends in a pill`() {
        assertEquals(12.dp, MochilRadius.r12)
        assertEquals(999.dp, MochilRadius.pill)
    }
}
