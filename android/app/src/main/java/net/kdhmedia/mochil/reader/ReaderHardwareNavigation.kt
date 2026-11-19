package net.kdhmedia.mochil.reader

import android.content.Context
import android.view.KeyEvent

enum class ReaderPageTurn { Previous, Next }

internal fun readerPageTurn(keyCode: Int, volumeButtonsEnabled: Boolean): ReaderPageTurn? = when (keyCode) {
    KeyEvent.KEYCODE_PAGE_UP -> ReaderPageTurn.Previous
    KeyEvent.KEYCODE_PAGE_DOWN -> ReaderPageTurn.Next
    KeyEvent.KEYCODE_DPAD_LEFT -> ReaderPageTurn.Previous
    KeyEvent.KEYCODE_DPAD_RIGHT -> ReaderPageTurn.Next
    KeyEvent.KEYCODE_BUTTON_L1 -> ReaderPageTurn.Previous
    KeyEvent.KEYCODE_BUTTON_R1 -> ReaderPageTurn.Next
    KeyEvent.KEYCODE_VOLUME_UP -> if (volumeButtonsEnabled) ReaderPageTurn.Previous else null
    KeyEvent.KEYCODE_VOLUME_DOWN -> if (volumeButtonsEnabled) ReaderPageTurn.Next else null
    else -> null
}

/**
 * Keys a remote uses to ask for the reader's controls, since it cannot tap.
 * Left and right stay page turns, so up, down and select open the menu.
 */
internal fun opensReaderControls(keyCode: Int): Boolean = when (keyCode) {
    KeyEvent.KEYCODE_DPAD_CENTER,
    KeyEvent.KEYCODE_ENTER,
    KeyEvent.KEYCODE_NUMPAD_ENTER,
    KeyEvent.KEYCODE_BUTTON_A,
    KeyEvent.KEYCODE_DPAD_UP,
    KeyEvent.KEYCODE_DPAD_DOWN,
    KeyEvent.KEYCODE_MENU -> true
    else -> false
}

/** Arrow keys drive focus, not pages, while a reader's controls are on screen. */
internal fun isFocusKey(keyCode: Int): Boolean = when (keyCode) {
    KeyEvent.KEYCODE_DPAD_LEFT, KeyEvent.KEYCODE_DPAD_RIGHT -> true
    else -> false
}

class ReaderHardwareNavigationPreferences(context: Context) {
    private val preferences = context.getSharedPreferences("reader_hardware_navigation", Context.MODE_PRIVATE)
    fun volumeButtonsEnabled(): Boolean = preferences.getBoolean(KEY_VOLUME_BUTTONS, false)
    fun setVolumeButtonsEnabled(enabled: Boolean) { preferences.edit().putBoolean(KEY_VOLUME_BUTTONS, enabled).apply() }

    private companion object { const val KEY_VOLUME_BUTTONS = "volume_page_turn" }
}
