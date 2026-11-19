package net.kdhmedia.mochil.ui

import android.os.SystemClock
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.withFrameNanos
import net.kdhmedia.mochil.diagnostics.PerformanceDiagnostics

/** Records one bounded activity-start-to-first-Compose-frame measurement. */
@Composable
internal fun RecordAppFirstFrame(television: Boolean, activityStartedAt: Long) {
    LaunchedEffect(activityStartedAt) {
        withFrameNanos { }
        PerformanceDiagnostics.record(
            "lifecycle",
            if (television) "tv_app_first_frame" else "mobile_first_frame",
            SystemClock.elapsedRealtime() - activityStartedAt,
        )
    }
}
