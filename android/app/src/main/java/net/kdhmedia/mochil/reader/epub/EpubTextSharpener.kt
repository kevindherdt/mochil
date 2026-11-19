package net.kdhmedia.mochil.reader.epub

import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.collectLatest
import kotlinx.coroutines.launch
import org.readium.r2.navigator.epub.EpubNavigatorFragment

/** The style that thickens the strokes of a page's text by [sharpness]; empty for Off, which removes it. */
internal fun sharpTextScript(sharpness: TextSharpness): String {
    val css = if (sharpness == TextSharpness.Off) "" else "html,body,body *{-webkit-text-stroke:${sharpness.stroke}px currentColor!important}"
    return "(function(){var s=document.getElementById('bobo-sharp');if(!s){s=document.createElement('style');s.id='bobo-sharp';(document.head||document.documentElement).appendChild(s)}s.textContent='$css'})()"
}

/**
 * Applies [TextSharpness] to the page on screen and to each page turned to afterwards. Readium only offers
 * colours, not stroke weight, so the weight is a small style put into the page itself.
 */
internal class EpubTextSharpener(private val scope: CoroutineScope) {
    private var following: Job? = null
    private var sharpness = TextSharpness.Off

    fun apply(navigator: EpubNavigatorFragment?, wanted: TextSharpness) {
        sharpness = wanted
        navigator ?: return
        if (following == null) following = scope.launch {
            // A page that has just loaded is not ready for the style at once.
            navigator.currentLocator.collectLatest { delay(150); inject(navigator) }
        }
        scope.launch { inject(navigator) }
    }

    private suspend fun inject(navigator: EpubNavigatorFragment) {
        runCatching { navigator.evaluateJavascript(sharpTextScript(sharpness)) }
    }
}
