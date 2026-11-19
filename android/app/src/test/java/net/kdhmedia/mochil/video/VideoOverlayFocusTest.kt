package net.kdhmedia.mochil.video

import android.app.Activity
import android.graphics.drawable.GradientDrawable
import android.graphics.drawable.StateListDrawable
import android.view.KeyEvent
import android.view.View
import android.view.ViewGroup
import android.widget.TextView
import org.junit.Assert.assertNotEquals
import org.junit.Assert.assertTrue
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.Robolectric
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35])
class VideoOverlayFocusTest {
    @Test
    fun `mobile bottom bar quick access activates tracks without an overflow menu`() {
        val activity = Robolectric.buildActivity(Activity::class.java).setup().get()
        val activated = mutableListOf<String>()
        val overlay = VideoPlayerOverlay(activity, {}, {}, { activated.add("audio") }, { activated.add("subtitles") }, {})
        activity.setContentView(overlay)
        overlay.setTrackAvailability(audio = true, subtitles = false)
        overlay.setControlsVisible(true)
        fun find(view: View, description: String): View? {
            if (view.contentDescription == description) return view
            if (view is ViewGroup) for (index in 0 until view.childCount) {
                find(view.getChildAt(index), description)?.let { return it }
            }
            return null
        }
        val audio = find(overlay, "Audio tracks")!!
        val subtitles = find(overlay, "Subtitles")!!
        assertTrue(audio.isShown)
        assertTrue(!subtitles.isShown)
        audio.performClick()
        assertTrue(activated == listOf("audio"))
        overlay.setTrackAvailability(audio = false, subtitles = true)
        assertTrue(!audio.isShown)
        assertTrue(subtitles.isShown)
        subtitles.performClick()
        assertTrue(activated == listOf("audio", "subtitles"))
    }

    @Test
    fun `mobile overflow menu has no audio or subtitle track entries`() {
        val activity = Robolectric.buildActivity(Activity::class.java).setup().get()
        val overlay = VideoPlayerOverlay(activity, {}, {}, {}, {}, {})
        activity.setContentView(overlay)
        overlay.setTrackAvailability(audio = true, subtitles = true)
        overlay.setControlsVisible(true)
        fun find(view: View, description: String): View? {
            if (view.contentDescription == description) return view
            if (view is ViewGroup) for (index in 0 until view.childCount) {
                find(view.getChildAt(index), description)?.let { return it }
            }
            return null
        }
        val more = find(overlay, "More video options")!!
        more.performClick()
        assertTrue(find(overlay, "Audio track") == null)
        assertTrue(find(overlay, "Subtitle track") == null)
        assertTrue(find(overlay, "Rotate")!!.isShown)
        assertTrue(find(overlay, "Picture in picture")!!.isShown)
        assertTrue(find(overlay, "Close video")!!.isShown)
    }

    @Test
    fun `tv separates track menu from playback settings`() {
        val activity = Robolectric.buildActivity(Activity::class.java).setup().get()
        val overlay = VideoPlayerOverlay(activity, {}, {}, {}, {}, {}, television = true)
        activity.setContentView(overlay)
        overlay.setTrackAvailability(audio = true, subtitles = true)
        overlay.setControlsVisible(true)
        fun find(view: View, description: String): View? {
            if (view.contentDescription == description) return view
            if (view is ViewGroup) for (index in 0 until view.childCount) {
                find(view.getChildAt(index), description)?.let { return it }
            }
            return null
        }
        val tracks = find(overlay, "Audio and subtitles")!!
        tracks.performClick()
        val audio = find(overlay, "Audio track")!!
        assertTrue(audio.isShown)
        assertTrue(audio.isFocused)
        assertTrue(!find(overlay, "Close video")!!.isShown)
        assertTrue(overlay.dismissMenu())
        assertTrue(tracks.isFocused)
        assertTrue(!audio.isShown)
        assertTrue(!overlay.dismissMenu())
    }
    @Test
    fun `tv track availability changes update menu and back focus`() {
        val activity = Robolectric.buildActivity(Activity::class.java).setup().get()
        val activated = mutableListOf<String>()
        val overlay = VideoPlayerOverlay(activity, {}, {}, { activated.add("audio") }, { activated.add("subtitles") }, {}, television = true)
        activity.setContentView(overlay)
        overlay.setTrackAvailability(audio = false, subtitles = true)
        overlay.setControlsVisible(true)
        fun find(view: View, description: String): View? {
            if (view.contentDescription == description) return view
            if (view is ViewGroup) for (index in 0 until view.childCount) {
                find(view.getChildAt(index), description)?.let { return it }
            }
            return null
        }
        val tracks = find(overlay, "Audio and subtitles")!!
        val more = find(overlay, "More video options")!!
        tracks.performClick()
        val audio = find(overlay, "Audio track")!!
        val subtitles = find(overlay, "Subtitle track")!!
        assertTrue(!audio.isShown)
        assertTrue(subtitles.isShown && subtitles.isFocused)
        assertTrue(overlay.dismissMenu())
        assertTrue(tracks.isFocused)
        tracks.performClick()
        subtitles.performClick()
        assertTrue(activated == listOf("subtitles"))
        assertTrue(tracks.isFocused)
        tracks.performClick()
        overlay.setTrackAvailability(audio = false, subtitles = false)
        assertTrue(!tracks.isShown && !subtitles.isShown)
        assertTrue(more.isFocused)
        assertTrue(!overlay.dismissMenu())
    }

    @Test
    fun `tv video menu actions visibly indicate remote focus`() {
        val activity = Robolectric.buildActivity(Activity::class.java).setup().get()
        val overlay = VideoPlayerOverlay(activity, {}, {}, {}, {}, {}, television = true)
        activity.setContentView(overlay)
        overlay.setTrackAvailability(audio = true, subtitles = true)
        overlay.setControlsVisible(true)
        fun find(view: View, description: String): View? {
            if (view.contentDescription == description) return view
            if (view is ViewGroup) for (index in 0 until view.childCount) {
                find(view.getChildAt(index), description)?.let { return it }
            }
            return null
        }
        find(overlay, "Audio and subtitles")!!.performClick()
        val action = find(overlay, "Audio track") as TextView
        assertTrue("TV menu action needs a focus-aware background", action.background is StateListDrawable)
        val background = action.background as StateListDrawable
        background.state = intArrayOf()
        val unfocused = (background.current as GradientDrawable).color!!.defaultColor
        background.state = intArrayOf(android.R.attr.state_focused)
        val focused = (background.current as GradientDrawable).color!!.defaultColor
        assertNotEquals("Focused action must be visually distinct", unfocused, focused)
    }

    @Test
    fun `all available tv menu actions show focus and activate with remote or keyboard`() {
        val activity = Robolectric.buildActivity(Activity::class.java).setup().get()
        val activated = mutableListOf<String>()
        val overlay = VideoPlayerOverlay(
            activity, { activated.add("Close video") }, {},
            { activated.add("Audio track") }, { activated.add("Subtitle track") }, {}, television = true,
        )
        activity.setContentView(overlay)
        overlay.setTrackAvailability(audio = true, subtitles = true)
        overlay.setControlsVisible(true)

        fun find(view: View, description: String): View? {
            if (view.contentDescription == description && (description != "Close video" || view is TextView)) return view
            if (view is ViewGroup) for (index in 0 until view.childCount) {
                find(view.getChildAt(index), description)?.let { return it }
            }
            return null
        }
        val more = find(overlay, "More video options")!!
        for ((label, key) in listOf(
            "Audio track" to KeyEvent.KEYCODE_DPAD_CENTER,
            "Subtitle track" to KeyEvent.KEYCODE_ENTER,
            "Close video" to KeyEvent.KEYCODE_DPAD_CENTER,
        )) {
            (if (label == "Close video") more else find(overlay, "Audio and subtitles")!!).performClick()
            val action = find(overlay, label)!!
            assertTrue("$label must take focus", action.requestFocus())
            assertTrue("$label must be focused", action.isFocused)
            val background = action.background as StateListDrawable
            val focused = (background.current as GradientDrawable).color!!.defaultColor
            action.clearFocus()
            val unfocused = (background.current as GradientDrawable).color!!.defaultColor
            assertNotEquals("$label needs visible focus", unfocused, focused)
            assertTrue(action.requestFocus())
            action.dispatchKeyEvent(KeyEvent(KeyEvent.ACTION_DOWN, key))
            action.dispatchKeyEvent(KeyEvent(KeyEvent.ACTION_UP, key))
            assertTrue("$label callback must run", activated.lastOrNull() == label)
            assertTrue("menu must close after $label", (action.parent as View).visibility == View.GONE)
        }
    }
}
