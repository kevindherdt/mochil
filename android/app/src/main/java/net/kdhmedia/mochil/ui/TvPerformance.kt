package net.kdhmedia.mochil.ui

import android.os.SystemClock
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.remember
import androidx.compose.runtime.withFrameNanos
import net.kdhmedia.mochil.diagnostics.PerformanceDiagnostics

/** Records only bounded route timing; no library, title, query, or account data. */
@Composable
internal fun RecordTvFirstFrame(route: String) {
    val started = remember(route) { SystemClock.elapsedRealtime() }
    LaunchedEffect(route) {
        withFrameNanos { }
        PerformanceDiagnostics.record("tv_ui", "${route}_first_frame", SystemClock.elapsedRealtime() - started)
    }
}
