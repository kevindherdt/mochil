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
    fun clear() = preferences.edit().remove(KEY).commit()

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
        if (report.length <= MAX_BYTES) preferences.edit().putString(KEY, report).commit()
    }

    private companion object {
        const val KEY = "pending"
        const val MAX_BYTES = 64 * 1024
    }
}
