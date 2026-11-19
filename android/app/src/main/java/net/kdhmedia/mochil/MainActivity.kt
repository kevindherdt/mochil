package net.kdhmedia.mochil

import android.os.Bundle
import android.content.pm.PackageManager
import android.content.res.Configuration
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.core.view.WindowCompat
import androidx.core.view.WindowInsetsCompat
import androidx.core.view.WindowInsetsControllerCompat
import net.kdhmedia.mochil.ui.MochilApp
import net.kdhmedia.mochil.ui.theme.MochilTheme
import net.kdhmedia.mochil.ui.theme.AppThemePreferences
import androidx.compose.runtime.remember
import androidx.compose.runtime.mutableStateOf
import net.kdhmedia.mochil.diagnostics.CrashReportStore
import net.kdhmedia.mochil.diagnostics.PerformanceDiagnostics
import net.kdhmedia.mochil.offline.PartialDownloadCleanup

class MainActivity : ComponentActivity() {
    private val activityStartedAt = android.os.SystemClock.elapsedRealtime()
    private var foregroundStarted = 0L

    override fun onCreate(savedInstanceState: Bundle?) {
        PerformanceDiagnostics.initialize(applicationContext)
        PartialDownloadCleanup.atProcessStart(applicationContext)
        installCrashCapture()
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        if (isTelevisionDevice(this)) hideTelevisionSystemBars()
        setContent {
            val preferences = remember { AppThemePreferences(applicationContext) }
            val selection = remember { mutableStateOf(preferences.load()) }
            MochilTheme(selection.value, glow = !isTelevisionDevice(this), cinema = isTelevisionDevice(this)) {
                net.kdhmedia.mochil.ui.RecordAppFirstFrame(isTelevisionDevice(this), activityStartedAt)
                MochilApp(television = isTelevisionDevice(this), themeSelection = selection.value, onThemeSelection = { preferences.save(it); selection.value = it })
            }
        }
    }

    override fun onStart() {
        super.onStart()
        foregroundStarted = android.os.SystemClock.elapsedRealtime()
    }

    override fun onWindowFocusChanged(hasFocus: Boolean) {
        super.onWindowFocusChanged(hasFocus)
        if (hasFocus && isTelevisionDevice(this)) hideTelevisionSystemBars()
    }

    private fun hideTelevisionSystemBars() {
        WindowCompat.getInsetsController(window, window.decorView).apply {
            systemBarsBehavior = WindowInsetsControllerCompat.BEHAVIOR_SHOW_TRANSIENT_BARS_BY_SWIPE
            hide(WindowInsetsCompat.Type.systemBars())
        }
    }

    override fun onStop() {
        if (foregroundStarted > 0) {
            PerformanceDiagnostics.record("lifecycle", "foreground_session", android.os.SystemClock.elapsedRealtime() - foregroundStarted)
            foregroundStarted = 0
        }
        super.onStop()
    }

    private fun installCrashCapture() {
        if (!crashCaptureInstalled.compareAndSet(false, true)) return
        val previous = Thread.getDefaultUncaughtExceptionHandler()
        Thread.setDefaultUncaughtExceptionHandler { thread, error ->
            runCatching { CrashReportStore(applicationContext).capture(error) }
            if (previous != null) previous.uncaughtException(thread, error)
            else android.os.Process.killProcess(android.os.Process.myPid())
        }
    }

    private companion object { val crashCaptureInstalled = java.util.concurrent.atomic.AtomicBoolean(false) }
}

internal fun isTelevisionDevice(activity: ComponentActivity): Boolean =
    isTelevisionUi(
        activity.resources.configuration.uiMode,
        activity.packageManager.hasSystemFeature(PackageManager.FEATURE_LEANBACK),
    ) || debugTelevisionPreview(activity)

/** Debug builds only: a file in the app's private storage previews the TV interface on a phone emulator. */
private fun debugTelevisionPreview(activity: ComponentActivity): Boolean =
    BuildConfig.DEBUG && java.io.File(activity.filesDir, "force-tv").exists()

internal fun isTelevisionUi(uiMode: Int, hasLeanbackFeature: Boolean): Boolean =
    uiMode and Configuration.UI_MODE_TYPE_MASK == Configuration.UI_MODE_TYPE_TELEVISION || hasLeanbackFeature
