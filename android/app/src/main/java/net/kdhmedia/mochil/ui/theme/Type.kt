package net.kdhmedia.mochil.ui.theme

import androidx.compose.material3.Typography
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.Font
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontVariation
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.TextUnit
import androidx.compose.ui.unit.sp
import net.kdhmedia.mochil.R

/**
 * The two families the web is set in (design/tokens.json typeFamily): Source Sans for everything you read or tap,
 * Newsreader for the few large headings. Both are the web's own variable fonts converted to TTF, so one file serves
 * every weight; minSdk 26 is the first Android that applies a variation, so the weight is real on every device we ship to.
 */
object MochilFonts {
    /** 430 is the web's body weight (`font: 430 16px/1.5`), a hair above regular so small text holds up on a dim screen. */
    private val weights = listOf(400, 430, 500, 600, 700)

    val ui: FontFamily = FontFamily(
        weights.map { w -> Font(R.font.source_sans, FontWeight(w), variationSettings = FontVariation.Settings(FontVariation.weight(w))) },
    )
    val display: FontFamily = FontFamily(
        weights.map { w -> Font(R.font.newsreader, FontWeight(w), variationSettings = FontVariation.Settings(FontVariation.weight(w))) },
    )
}

private val Body = FontWeight(430)

private fun ui(size: TextUnit, line: TextUnit, weight: FontWeight = Body) =
    TextStyle(fontFamily = MochilFonts.ui, fontSize = size, lineHeight = line, fontWeight = weight, letterSpacing = 0.sp)

private fun display(size: TextUnit, line: TextUnit) =
    TextStyle(fontFamily = MochilFonts.display, fontSize = size, lineHeight = line, fontWeight = FontWeight.Medium, letterSpacing = 0.sp)

/** The phone/tablet type scale: Material's slots filled from the web's size tokens, so a `bodyMedium` here is the web's 14. */
fun mochilTypography(): Typography = Typography(
    displayLarge = display(MochilFontSize.s30, 36.sp),
    displayMedium = display(MochilFontSize.s30, 36.sp),
    displaySmall = display(MochilFontSize.s26, 32.sp),
    headlineLarge = display(MochilFontSize.s30, 36.sp),
    headlineMedium = display(MochilFontSize.s26, 32.sp),
    headlineSmall = display(MochilFontSize.s22, 28.sp),
    titleLarge = ui(MochilFontSize.s22, 28.sp, FontWeight.SemiBold),
    titleMedium = ui(MochilFontSize.s16, 24.sp, FontWeight.SemiBold),
    titleSmall = ui(MochilFontSize.s14, 20.sp, FontWeight.SemiBold),
    bodyLarge = ui(MochilFontSize.s16, 24.sp),
    bodyMedium = ui(MochilFontSize.s14, 20.sp),
    bodySmall = ui(MochilFontSize.s12, 16.sp),
    labelLarge = ui(MochilFontSize.s14, 20.sp, FontWeight.SemiBold),
    labelMedium = ui(MochilFontSize.s12, 16.sp, FontWeight.SemiBold),
    labelSmall = ui(MochilFontSize.s11, 16.sp, FontWeight.Medium),
)
