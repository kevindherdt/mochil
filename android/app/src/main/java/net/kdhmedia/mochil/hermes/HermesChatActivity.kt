package net.kdhmedia.mochil.hermes

import android.content.Context
import android.content.Intent
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.activity.viewModels
import net.kdhmedia.mochil.audio.AudioClient
import net.kdhmedia.mochil.audio.AudioTrack
import net.kdhmedia.mochil.auth.DeviceCredentialStore
import net.kdhmedia.mochil.playback.PlaybackViewModel
import net.kdhmedia.mochil.ui.theme.MochilTheme

/** The chat with the assistant (Hermes). Opened from the menu when the server offers it to this device. */
class HermesChatActivity : ComponentActivity() {
    private val viewModel: HermesChatViewModel by viewModels()
    private val playback: PlaybackViewModel by viewModels()

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        setContent { MochilTheme { HermesChatScreen(viewModel, onClose = ::finish, onPlayCard = ::playCard) } }
    }

    /**
     * A Play card starts the item in the app's own player, so it keeps going with the screen off, with the other cards of
     * the same answer queued after it. Tapping the song that is already loaded pauses or resumes it instead of restarting.
     */
    private fun playCard(card: HermesCard, siblings: List<HermesCard>) {
        viewModel.stopSpeaking() // a "tap Play" sentence read out after the song started is only noise
        val current = playback.state.value.current
        if (current != null && current.id == card.id && current.libraryId == card.library) { playback.toggle(); return }
        val credential = DeviceCredentialStore(this).load() ?: return
        fun track(item: HermesCard) = AudioTrack(id = item.id, libraryId = item.library, title = item.title, artist = item.subtitle, album = "", durationSeconds = 0.0, libraryKind = item.kind)
        val queue = siblings.ifEmpty { listOf(card) }.map(::track)
        val selected = queue.first { it.id == card.id && it.libraryId == card.library }
        val client = AudioClient()
        playback.play(queue, selected, streamUrl = { client.streamUrl(credential, it) })
    }

    companion object {
        fun intent(context: Context) = Intent(context, HermesChatActivity::class.java)
    }
}
