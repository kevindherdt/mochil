package net.kdhmedia.mochil.reader.epub

import org.readium.r2.navigator.epub.EpubNavigatorFragment
import org.readium.r2.navigator.input.InputListener
import org.readium.r2.navigator.input.TapEvent
import org.readium.r2.navigator.preferences.ReadingProgression
import org.readium.r2.shared.ExperimentalReadiumApi

internal enum class TapAction { Back, Forward, Menu }

/**
 * What a tap at [x] (of [width]) does. Edges: the outer quarters turn the page. One hand: the left fifth goes back and
 * all the rest goes forward, so a thumb can read without moving. Either way the middle opens the menu. In a book that
 * reads right to left, the left side goes forward.
 */
internal fun tapAction(zones: TapZones, x: Float, width: Float, rightToLeft: Boolean): TapAction {
    if (zones == TapZones.Off || width <= 0f) return TapAction.Menu
    val share = (x / width).coerceIn(0f, 1f)
    val side = when (zones) {
        TapZones.Edges -> when { share < 0.25f -> TapAction.Back; share > 0.75f -> TapAction.Forward; else -> TapAction.Menu }
        else -> when { share < 0.2f -> TapAction.Back; share in 0.4f..0.6f -> TapAction.Menu; else -> TapAction.Forward }
    }
    if (!rightToLeft || side == TapAction.Menu) return side
    return if (side == TapAction.Back) TapAction.Forward else TapAction.Back
}

/** Turns pages from taps by the chosen zones, and opens the menu otherwise. Reading aloud is never affected by a page turn. */
@OptIn(ExperimentalReadiumApi::class)
internal class EpubTapZones(
    private val navigator: () -> EpubNavigatorFragment?,
    private val zones: () -> TapZones,
    private val animate: () -> Boolean,
    private val onMenu: () -> Unit,
) : InputListener {
    override fun onTap(event: TapEvent): Boolean {
        val reader = navigator() ?: return false
        val rightToLeft = reader.overflow.value.readingProgression == ReadingProgression.RTL
        return when (tapAction(zones(), event.point.x, reader.publicationView.width.toFloat(), rightToLeft)) {
            TapAction.Menu -> { onMenu(); false }
            TapAction.Forward -> { reader.goForward(animated = animate()); true }
            TapAction.Back -> { reader.goBackward(animated = animate()); true }
        }
    }
}
