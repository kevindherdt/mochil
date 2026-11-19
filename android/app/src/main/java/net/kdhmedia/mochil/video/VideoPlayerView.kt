package net.kdhmedia.mochil.video

import android.app.Activity
import androidx.media3.common.Player
import androidx.media3.ui.PlayerView
import net.kdhmedia.mochil.R

/**
 * The player surface. Phones get the Mochil controller layout; television keeps the stock
 * controller, whose focus handling is built for a remote, in the TV accent colours.
 */
internal fun newPlayerView(activity: Activity, television: Boolean, player: Player, overlay: VideoPlayerOverlay): PlayerView {
    if (television) {
        val styled = runCatching { activity.layoutInflater.inflate(R.layout.tv_player_view, null) as PlayerView }.getOrNull()
        return (styled ?: PlayerView(activity)).apply { this.player = player }
    }
    val inflated = runCatching { activity.layoutInflater.inflate(R.layout.bobo_player_view, null) as PlayerView }
    val view = inflated.getOrNull() ?: run {
        val cause = generateSequence<Throwable>(inflated.exceptionOrNull()) { it.cause }.last()
        overlay.useStockControls("${cause.javaClass.simpleName}: ${cause.message}".take(220))
        return PlayerView(activity).apply { this.player = player }
    }
    return view.apply { this.player = player }
}
