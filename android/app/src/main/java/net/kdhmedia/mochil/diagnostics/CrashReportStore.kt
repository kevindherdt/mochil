@file:android.annotation.SuppressLint("ApplySharedPref") // Crash capture must reach disk before the process terminates.
package net.kdhmedia.mochil.diagnostics

import android.content.Context
import android.os.Build
import net.kdhmedia.mochil.BuildConfig
import org.json.JSONArray
import org.json.JSONObject

class CrashReportStore(context: Context) {
    private val preferences = context.getSharedPreferences("crash_reports", Context.MODE_PRIVATE)

    fun hasPending(): Boolean = preferences.contains(KEY)
    fun pending(): String? = preferences.getString(KEY, null)
    fun clear() = preferences.edit().remove(KEY).remove(KEY_ID).commit()

    /**
     * True at most once per crash. Resuming a saved session after a crash reproduces the crash before
     * the report can be read, so the first start after one deliberately stays signed out — but only
     * the first: parking on every start turned "the app crashed once" into "sign in with the server
     * address again every time you open it", which is what this id exists to prevent.
     */
    fun aFreshCrashShouldStartSignedOut(): Boolean {
        val crashId = preferences.getString(KEY_ID, null) ?: return false
        if (crashId == preferences.getString(KEY_PARKED_ID, null)) return false
        preferences.edit().putString(KEY_PARKED_ID, crashId).commit()
        return true
    }

    fun capture(error: Throwable) {
        val frames = JSONArray()
        var current: Throwable? = error
        var causes = 0
        while (current != null && causes < 4 && frames.length() < 80) {
            current.stackTrace.take(80 - frames.length()).forEach { frame ->
                frames.put(JSONObject().apply {
                    put("class", frame.className.take(240))
                    put("method", frame.methodName.take(240))
                    frame.fileName?.substringAfterLast('/')?.substringAfterLast('\\')?.take(240)?.let { put("file", it) }
                    put("line", frame.lineNumber.coerceIn(-1, 10_000_000))
                })
            }
            current = current.cause
            causes++
        }
        if (frames.length() == 0) return
        val report = JSONObject().apply {
            put("exception", error.javaClass.name.take(240))
            put("app_version", BuildConfig.VERSION_NAME.take(32))
            put("android", Build.VERSION.RELEASE.orEmpty().take(32))
            put("device", "${Build.MANUFACTURER} ${Build.MODEL}".trim().take(120))
            put("frames", frames)
        }.toString()
        if (report.length <= MAX_BYTES) {
            preferences.edit()
                .putString(KEY, report)
                .putString(KEY_ID, java.util.UUID.randomUUID().toString())
                .commit()
        }
    }

    private companion object {
        const val KEY = "pending"
        const val KEY_ID = "pending_id"
        const val KEY_PARKED_ID = "parked_id"
        const val MAX_BYTES = 64 * 1024
    }
}
