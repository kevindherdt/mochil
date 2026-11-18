package net.kdhmedia.mochil.reader.epub

import android.graphics.Color
import android.view.Gravity
import android.view.View
import android.widget.FrameLayout
import android.widget.TextView
import androidx.compose.runtime.Composable
import androidx.compose.ui.platform.ComposeView
import androidx.compose.ui.platform.ViewCompositionStrategy
import androidx.fragment.app.FragmentActivity
import net.kdhmedia.mochil.ui.theme.MochilTheme

/** Owns the layered Android/Compose surface used by the EPUB reader. */
internal class EpubReaderSurface(private val activity: FragmentActivity) {
    val containerId: Int = View.generateViewId()
    private lateinit var chromeView: ComposeView
    private lateinit var announcer: TextView

    fun mount(
        progressContent: @Composable () -> Unit,
        chromeContent: @Composable () -> Unit,
    ) {
        val root = FrameLayout(activity)
        val readerContainer = FrameLayout(activity).apply { id = containerId }
        val progressView = composeHost(progressContent)
        chromeView = composeHost(chromeContent).apply { visibility = View.GONE }
        announcer = TextView(activity).apply {
            importantForAccessibility = View.IMPORTANT_FOR_ACCESSIBILITY_YES
            accessibilityLiveRegion = View.ACCESSIBILITY_LIVE_REGION_POLITE
            setTextColor(Color.TRANSPARENT)
        }

        root.addView(readerContainer, FrameLayout.LayoutParams(match, match))
        root.addView(progressView, FrameLayout.LayoutParams(match, wrap, Gravity.BOTTOM))
        root.addView(chromeView, FrameLayout.LayoutParams(match, match))
        root.addView(announcer, FrameLayout.LayoutParams(1, 1))
        activity.setContentView(root)
    }

    fun announce(message: String?) {
        if (::announcer.isInitialized) announcer.text = message.orEmpty()
    }

    fun showChrome() {
        if (::chromeView.isInitialized) chromeView.visibility = View.VISIBLE
    }

    fun hideChrome() {
        if (::chromeView.isInitialized) chromeView.visibility = View.GONE
    }

    /** Moves the remote's focus onto the controls once they are showing. */
    fun focusChrome() {
        if (::chromeView.isInitialized) chromeView.post { chromeView.requestFocus() }
    }

    fun chromeHasFocus(): Boolean = ::chromeView.isInitialized && chromeView.hasFocus()

    private fun composeHost(content: @Composable () -> Unit) = ComposeView(activity).apply {
        setViewCompositionStrategy(ViewCompositionStrategy.DisposeOnViewTreeLifecycleDestroyed)
        setContent { MochilTheme { content() } }
    }

    private companion object {
        const val match = FrameLayout.LayoutParams.MATCH_PARENT
        const val wrap = FrameLayout.LayoutParams.WRAP_CONTENT
    }
}
