package net.kdhmedia.mochil.video

import android.content.Context
import android.graphics.Color
import android.graphics.drawable.GradientDrawable
import android.graphics.drawable.StateListDrawable
import android.view.Gravity
import android.view.View
import android.widget.FrameLayout
import android.widget.ImageButton
import android.widget.LinearLayout
import android.widget.TextView
import net.kdhmedia.mochil.R

@android.annotation.SuppressLint("ViewConstructor")
internal class VideoPlayerOverlay(
    context: Context,
    onClose: () -> Unit,
    onRotate: () -> Unit,
    onAudioTracks: () -> Unit,
    onSubtitleTracks: () -> Unit,
    onPictureInPicture: () -> Unit,
    title: String = "",
    television: Boolean = false,
) : FrameLayout(context) {
    private val isTelevision = television
    private var customControls = true
    private var audioAvailable = false
    private var subtitlesAvailable = false
    private val audioAction: TextView = menuAction("Audio track", R.drawable.ic_player_audio) { onAudioTracks() }
    private val subtitleAction: TextView = menuAction("Subtitle track", R.drawable.ic_player_subtitles) { onSubtitleTracks() }
    private val menu: LinearLayout = LinearLayout(context).apply {
        orientation = LinearLayout.VERTICAL
        gravity = Gravity.END
        setPadding(dp(6), dp(6), dp(6), dp(6))
        background = roundedBackground(0xE61A1A1E.toInt(), dp(12).toFloat())

        // A TV is never turned on its side and rarely floats a window, so those two only clutter its menu.
        // On mobile, audio/subtitle selection already has a quick-access path in the bottom bar
        // (audioLabel/subtitleLabel), so this overflow menu does not duplicate those entries.
        if (!isTelevision) {
            addView(menuAction("Rotate", R.drawable.ic_player_rotate) { onRotate() })
            addView(menuAction("Picture in picture", R.drawable.ic_player_pip) { onPictureInPicture() })
        }
        addView(menuAction("Close video", R.drawable.ic_player_close) { onClose() })
        visibility = GONE
    }
    private val trackMenu = LinearLayout(context).apply {
        orientation = LinearLayout.VERTICAL
        setPadding(dp(6), dp(6), dp(6), dp(6))
        background = roundedBackground(0xE61A1A1E.toInt(), dp(12).toFloat())
        if (isTelevision) {
            addView(audioAction)
            addView(subtitleAction)
        }
        visibility = GONE
    }
    private val tracks = ImageButton(context).apply {
        setImageResource(R.drawable.ic_player_subtitles)
        contentDescription = "Audio and subtitles"
        setColorFilter(Color.WHITE)
        background = focusBackground()
        setOnClickListener { toggleMenu(trackMenu, this) }
        visibility = GONE
    }
    private val more = ImageButton(context).apply {
        setImageResource(if (isTelevision) R.drawable.ic_player_settings else R.drawable.ic_player_more)
        contentDescription = "More video options"
        setColorFilter(Color.WHITE)
        background = focusBackground()
        setOnClickListener { toggleMenu(menu, this) }
        visibility = GONE
    }
    private val back = ImageButton(context).apply {
        setImageResource(R.drawable.ic_player_back)
        contentDescription = "Close video"
        setColorFilter(Color.WHITE)
        background = focusBackground()
        setOnClickListener { onClose() }
        visibility = GONE
    }
    private val heading = TextView(context).apply {
        text = title
        setTextColor(Color.WHITE)
        textSize = 17f
        maxLines = 1
        ellipsize = android.text.TextUtils.TruncateAt.END
        visibility = GONE
    }
    private val subtitleLabel = labelled("Subtitles", R.drawable.ic_player_subtitles) { onSubtitleTracks() }
    private val audioLabel = labelled("Audio tracks", R.drawable.ic_player_audio) { onAudioTracks() }
    private val bottomRow = LinearLayout(context).apply {
        orientation = LinearLayout.HORIZONTAL
        setPadding(dp(8), 0, dp(8), 0)
        addView(subtitleLabel)
        addView(audioLabel)
        visibility = GONE
    }
    private val message = TextView(context).apply {
        setTextColor(Color.WHITE)
        textSize = 15f
        gravity = Gravity.CENTER
        setPadding(dp(20), dp(12), dp(20), dp(8))
    }
    private val messageAction = TextView(context).apply {
        setTextColor(0xFFFFB18F.toInt())
        textSize = 16f
        gravity = Gravity.CENTER
        isFocusable = true
        background = focusBackground()
        setPadding(dp(20), dp(8), dp(20), dp(12))
        visibility = GONE
    }
    private val messagePanel = LinearLayout(context).apply {
        orientation = LinearLayout.VERTICAL
        gravity = Gravity.CENTER
        background = roundedBackground(0xE618181B.toInt(), dp(12).toFloat())
        addView(message, LinearLayout.LayoutParams(LayoutParams.WRAP_CONTENT, LayoutParams.WRAP_CONTENT))
        addView(messageAction, LinearLayout.LayoutParams(LayoutParams.MATCH_PARENT, dp(48)))
        visibility = GONE
    }

    init {
        isClickable = false
        if (!television) {
            addView(back, LayoutParams(dp(48), dp(48), Gravity.TOP or Gravity.START).apply { topMargin = dp(12); marginStart = dp(12) })
            addView(heading, LayoutParams(LayoutParams.MATCH_PARENT, dp(48), Gravity.TOP or Gravity.START).apply { topMargin = dp(12); marginStart = dp(68); marginEnd = dp(72) }.also { heading.gravity = Gravity.CENTER_VERTICAL })
        }
        if (!television) addView(bottomRow, LayoutParams(LayoutParams.MATCH_PARENT, dp(48), Gravity.BOTTOM or Gravity.START).apply { bottomMargin = dp(8) })
        addView(more, LayoutParams(dp(48), dp(48), Gravity.TOP or Gravity.END).apply { topMargin = dp(12); marginEnd = dp(12) })
        if (television) addView(tracks, LayoutParams(dp(48), dp(48), Gravity.TOP or Gravity.END).apply { topMargin = dp(12); marginEnd = dp(64) })
        addView(menu, LayoutParams(dp(210), LayoutParams.WRAP_CONTENT, Gravity.TOP or Gravity.END).apply { topMargin = dp(68); marginEnd = dp(12) })
        if (television) addView(trackMenu, LayoutParams(dp(210), LayoutParams.WRAP_CONTENT, Gravity.TOP or Gravity.END).apply { topMargin = dp(68); marginEnd = dp(12) })
        addView(messagePanel, LayoutParams(LayoutParams.WRAP_CONTENT, LayoutParams.WRAP_CONTENT, Gravity.BOTTOM or Gravity.CENTER_HORIZONTAL).apply { marginStart = dp(24); marginEnd = dp(24); bottomMargin = dp(190) })
        refreshAvailability()
    }

    fun setTitle(value: String) { heading.text = value }

    /** The Mochil controller could not be built; the stock one is showing, so say why in the title bar. */
    fun useStockControls(reason: String) {
        customControls = false
        heading.maxLines = 4
        heading.textSize = 11f
        heading.text = reason
    }

    fun setControlsVisible(visible: Boolean) {
        more.visibility = if (visible) VISIBLE else GONE
        tracks.visibility = if (visible && isTelevision && (audioAvailable || subtitlesAvailable)) VISIBLE else GONE
        val bar = if (visible && !isTelevision) VISIBLE else GONE
        back.visibility = bar
        bottomRow.visibility = if (customControls) bar else GONE
        heading.visibility = bar
        if (!visible) closeMenu()
    }

    fun setTrackAvailability(audio: Boolean, subtitles: Boolean) {
        audioAvailable = audio
        subtitlesAvailable = subtitles
        refreshAvailability()
        tracks.visibility = if (more.visibility == VISIBLE && isTelevision && (audio || subtitles)) VISIBLE else GONE
        if (!audio && !subtitles && trackMenu.visibility == VISIBLE) { closeMenu(); more.requestFocus() }
    }

    fun showMessage(value: String, actionLabel: String? = null, action: (() -> Unit)? = null) {
        message.text = value
        messageAction.visibility = if (actionLabel != null && action != null) VISIBLE else GONE
        messageAction.text = actionLabel.orEmpty()
        messageAction.contentDescription = actionLabel.orEmpty()
        messageAction.setOnClickListener { action?.invoke() }
        messagePanel.visibility = VISIBLE
        if (messageAction.visibility == VISIBLE) messageAction.requestFocus()
    }

    fun clearMessage() {
        messagePanel.visibility = GONE
        messageAction.setOnClickListener(null)
    }

    private fun refreshAvailability() {
        audioAction.visibility = if (audioAvailable) VISIBLE else GONE
        subtitleAction.visibility = if (subtitlesAvailable) VISIBLE else GONE
        audioLabel.visibility = if (audioAvailable) VISIBLE else GONE
        subtitleLabel.visibility = if (subtitlesAvailable) VISIBLE else GONE
    }

    private fun menuAction(label: String, icon: Int, action: () -> Unit): TextView = TextView(context).apply {
        text = label
        setCompoundDrawablesRelativeWithIntrinsicBounds(icon, 0, 0, 0)
        compoundDrawablePadding = dp(16)
        contentDescription = label
        setTextColor(Color.WHITE)
        textSize = 16f
        gravity = Gravity.CENTER_VERTICAL
        setPadding(dp(16), dp(12), dp(16), dp(12))
        isFocusable = true
        background = focusBackground()
        setOnClickListener { dismissMenu(); action() }
        layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, dp(48))
    }

    private fun labelled(label: String, icon: Int, action: () -> Unit): TextView = TextView(context).apply {
        text = label
        contentDescription = label
        setTextColor(Color.WHITE)
        textSize = 14f
        gravity = Gravity.CENTER_VERTICAL
        compoundDrawablePadding = dp(8)
        setCompoundDrawablesRelativeWithIntrinsicBounds(icon, 0, 0, 0)
        setPadding(dp(12), 0, dp(16), 0)
        isFocusable = true
        background = focusBackground()
        setOnClickListener { action() }
        layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.WRAP_CONTENT, LinearLayout.LayoutParams.MATCH_PARENT)
    }

    private fun toggleMenu(target: LinearLayout, anchor: View) {
        if (target.visibility == VISIBLE) { closeMenu(); anchor.requestFocus(); return }
        closeMenu()
        target.visibility = VISIBLE
        (0 until target.childCount).map { target.getChildAt(it) }.firstOrNull { it.visibility == VISIBLE }?.requestFocus()
    }

    fun dismissMenu(): Boolean {
        val anchor = when {
            trackMenu.visibility == VISIBLE -> tracks
            menu.visibility == VISIBLE -> more
            else -> return false
        }
        closeMenu()
        anchor.requestFocus()
        return true
    }

    private fun closeMenu() { menu.visibility = GONE; trackMenu.visibility = GONE }

    private fun roundedBackground(color: Int, radius: Float) = GradientDrawable().apply { setColor(color); cornerRadius = radius }
    private fun focusBackground() = StateListDrawable().apply {
        addState(intArrayOf(android.R.attr.state_focused), roundedBackground(0x33FFFFFF, dp(8).toFloat()))
        addState(intArrayOf(android.R.attr.state_pressed), roundedBackground(0x33FFFFFF, dp(8).toFloat()))
        addState(intArrayOf(), roundedBackground(Color.TRANSPARENT, dp(8).toFloat()))
    }
    private fun dp(value: Int) = (value * resources.displayMetrics.density).toInt()
}
