package net.kdhmedia.mochil.ui

import android.app.ActivityManager
import android.content.Context
import android.media.AudioDeviceInfo
import android.media.AudioManager
import android.os.Build
import android.view.KeyEvent
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Button
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.Stable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.input.key.onPreviewKeyEvent
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp

internal enum class TvRemoteAction(val label: String) {
    Up("Up"), Down("Down"), Left("Left"), Right("Right"), Select("Select"), PlayPause("Play / pause")
}

internal fun tvRemoteAction(keyCode: Int): TvRemoteAction? = when (keyCode) {
    KeyEvent.KEYCODE_DPAD_UP -> TvRemoteAction.Up
    KeyEvent.KEYCODE_DPAD_DOWN -> TvRemoteAction.Down
    KeyEvent.KEYCODE_DPAD_LEFT -> TvRemoteAction.Left
    KeyEvent.KEYCODE_DPAD_RIGHT -> TvRemoteAction.Right
    KeyEvent.KEYCODE_DPAD_CENTER, KeyEvent.KEYCODE_ENTER, KeyEvent.KEYCODE_BUTTON_A -> TvRemoteAction.Select
    KeyEvent.KEYCODE_MEDIA_PLAY_PAUSE, KeyEvent.KEYCODE_MEDIA_PLAY, KeyEvent.KEYCODE_MEDIA_PAUSE -> TvRemoteAction.PlayPause
    else -> null
}

internal data class TvDeviceProfile(
    val androidVersion: String,
    val memoryClassMb: Int,
    val lowRam: Boolean,
    val hdmiDetected: Boolean,
) {
    companion object {
        fun read(context: Context): TvDeviceProfile {
            val activity = context.getSystemService(ActivityManager::class.java)
            val audio = context.getSystemService(AudioManager::class.java)
            val hdmiTypes = setOf(AudioDeviceInfo.TYPE_HDMI, AudioDeviceInfo.TYPE_HDMI_ARC) +
                if (Build.VERSION.SDK_INT >= 31) setOf(AudioDeviceInfo.TYPE_HDMI_EARC) else emptySet()
            return TvDeviceProfile(
                androidVersion = Build.VERSION.RELEASE,
                memoryClassMb = activity?.memoryClass ?: 0,
                lowRam = activity?.isLowRamDevice == true,
                hdmiDetected = audio?.getDevices(AudioManager.GET_DEVICES_OUTPUTS)?.any { it.type in hdmiTypes } == true,
            )
        }
    }
}

@Stable
internal class TvReadinessState {
    var active by mutableStateOf(false)
        private set
    var observed by mutableStateOf(emptySet<TvRemoteAction>())
        private set

    fun start() { observed = emptySet(); active = true }
    fun stop() { active = false }
    fun observe(keyCode: Int) {
        if (!active) return
        tvRemoteAction(keyCode)?.let { observed = observed + it }
    }
}

@Composable
internal fun rememberTvReadinessState(): TvReadinessState = remember { TvReadinessState() }

internal fun Modifier.observeTvReadiness(state: TvReadinessState): Modifier = onPreviewKeyEvent { event ->
    if (event.nativeKeyEvent.action == KeyEvent.ACTION_DOWN) state.observe(event.nativeKeyEvent.keyCode)
    false
}

@Composable
internal fun TvReadinessCheck(state: TvReadinessState) {
    val context = LocalContext.current
    val profile = remember { TvDeviceProfile.read(context) }
    if (!state.active) {
        Button(onClick = state::start, modifier = Modifier.tvFocusBorder()) {
            Text("Test this TV")
        }
        Text("Checks remote input and shows the device conditions useful for TV performance testing.", color = MaterialTheme.colorScheme.onSurfaceVariant)
        return
    }

    val directions = listOf(TvRemoteAction.Up, TvRemoteAction.Down, TvRemoteAction.Left, TvRemoteAction.Right)
    val directionCount = directions.count(state.observed::contains)
    Column(
        Modifier.fillMaxWidth().border(2.dp, MaterialTheme.colorScheme.outline, RoundedCornerShape(14.dp)).padding(20.dp),
        verticalArrangement = Arrangement.spacedBy(14.dp),
    ) {
        Text("Remote check", style = MaterialTheme.typography.titleLarge)
        Text("Press each direction, Select, and Play/Pause. Focus should move with a clear outline; this panel records the keys without intercepting them.", color = MaterialTheme.colorScheme.onSurfaceVariant)
        Row(horizontalArrangement = Arrangement.spacedBy(24.dp)) {
            TvCheckResult("Directions", "$directionCount / 4", directionCount == 4)
            TvCheckResult("Select", if (TvRemoteAction.Select in state.observed) "Received" else "Waiting", TvRemoteAction.Select in state.observed)
            TvCheckResult("Play/Pause", if (TvRemoteAction.PlayPause in state.observed) "Received" else "Waiting", TvRemoteAction.PlayPause in state.observed)
        }
        Text("Android ${profile.androidVersion} · ${profile.memoryClassMb} MB app memory · ${if (profile.lowRam) "low-memory device" else "standard-memory device"}")
        Text(if (profile.hdmiDetected) "HDMI audio output detected. Remote directions received through HDMI-CEC count above." else "No HDMI audio output is currently reported by Android.", color = MaterialTheme.colorScheme.onSurfaceVariant)
        Text("Display policy: video keeps the screen awake; audio continues if the TV starts its screen saver.", color = MaterialTheme.colorScheme.onSurfaceVariant)
        Button(onClick = state::stop, modifier = Modifier.tvFocusBorder()) { Text("Finish check") }
    }
}

@Composable
private fun TvCheckResult(label: String, value: String, complete: Boolean) {
    Column {
        Text(label, color = MaterialTheme.colorScheme.onSurfaceVariant)
        Text(value, color = if (complete) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurface)
    }
}
