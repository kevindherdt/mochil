package net.kdhmedia.mochil.reader.epub

import android.content.Context
import org.json.JSONObject
import org.readium.r2.shared.publication.Locator

internal interface EpubProgressStateStore {
    fun beginSync(publicationKey: String)
    fun endSync(publicationKey: String, acknowledged: Boolean = false)
    fun pending(publicationKey: String): Boolean
    fun markPending(publicationKey: String)
    fun markSynchronized(publicationKey: String, locator: Locator, revision: Long, nowMillis: Long = System.currentTimeMillis()): Long
    fun adoptRemote(publicationKey: String, expected: Locator?, locator: Locator, revision: Long): Boolean
    fun current(publicationKey: String): Locator?
    fun revision(publicationKey: String): Long
    fun saveRevision(publicationKey: String, revision: Long)
    fun lastSynchronizedAt(publicationKey: String): Long
}

class EpubLocatorStore(context: Context) : EpubProgressStateStore {
    private val preferences = context.getSharedPreferences("epub_reader_locations", Context.MODE_PRIVATE)
    private val inFlightKey: (String) -> Pair<android.content.SharedPreferences, String> = { preferences to it }

    override fun beginSync(publicationKey: String) = synchronized(preferences) {
        val key = inFlightKey(publicationKey)
        inFlight[key] = (inFlight[key] ?: 0) + 1
        preferences.edit().putBoolean(pendingKey(publicationKey), true).putBoolean(uncertainKey(publicationKey), true).apply()
    }

    override fun endSync(publicationKey: String, acknowledged: Boolean) = synchronized(preferences) {
        val key = inFlightKey(publicationKey)
        if (!acknowledged) {
            failedFlights.add(key)
            preferences.edit().putBoolean(pendingKey(publicationKey), true).apply()
        }
        val remaining = (inFlight[key] ?: 1) - 1
        if (remaining <= 0) {
            inFlight.remove(key)
            val failed = failedFlights.remove(key)
            if (failed) {
                preferences.edit().putBoolean(pendingKey(publicationKey), true).apply()
            } else {
                val local = preferences.getString(publicationKey, null)
                val synced = preferences.getString(synchronizedKey(publicationKey), null)
                preferences.edit().putBoolean(uncertainKey(publicationKey), false).putBoolean(pendingKey(publicationKey), local != null && positionNeedsSynchronization(local, synced)).apply()
            }
        } else inFlight[key] = remaining
        Unit
    }

    private companion object {
        val failedFlights = java.util.concurrent.ConcurrentHashMap.newKeySet<Pair<android.content.SharedPreferences, String>>()
        val inFlight = java.util.concurrent.ConcurrentHashMap<Pair<android.content.SharedPreferences, String>, Int>()
    }

    fun load(publicationKey: String): Locator? =
        preferences.getString(publicationKey, null)?.let { encoded ->
            runCatching { Locator.fromJSON(JSONObject(encoded)) }.getOrNull()
        }

    override fun current(publicationKey: String): Locator? = load(publicationKey)

    fun save(publicationKey: String, locator: Locator) = synchronized(preferences) {
        val encoded = locator.toJSON().toString()
        preferences.edit().putString(publicationKey, encoded)
            .putBoolean(pendingKey(publicationKey), preferences.getBoolean(uncertainKey(publicationKey), false) || positionNeedsSynchronization(encoded, preferences.getString(synchronizedKey(publicationKey), null)))
            .apply()
    }

    fun saveNow(publicationKey: String, locator: Locator): Boolean = synchronized(preferences) {
        val encoded = locator.toJSON().toString()
        preferences.edit().putString(publicationKey, encoded)
            .putBoolean(pendingKey(publicationKey), preferences.getBoolean(uncertainKey(publicationKey), false) || positionNeedsSynchronization(encoded, preferences.getString(synchronizedKey(publicationKey), null)))
            .commit()
    }

    override fun pending(publicationKey: String): Boolean = preferences.getBoolean(pendingKey(publicationKey), false)

    override fun markPending(publicationKey: String) = synchronized(preferences) {
        preferences.edit().putBoolean(pendingKey(publicationKey), true).apply()
    }

    override fun markSynchronized(publicationKey: String, locator: Locator, revision: Long, nowMillis: Long): Long = synchronized(preferences) {
        if (revision < revision(publicationKey)) return@synchronized lastSynchronizedAt(publicationKey)
        val encoded = locator.toJSON().toString()
        val local = preferences.getString(publicationKey, null)
        preferences.edit().putString(synchronizedKey(publicationKey), encoded)
            .putBoolean(uncertainKey(publicationKey), (inFlight[inFlightKey(publicationKey)] ?: 0) > 0)
            .putLong("revision:$publicationKey", revision).putLong(synchronizedAtKey(publicationKey), nowMillis)
            .putBoolean(pendingKey(publicationKey), (inFlight[inFlightKey(publicationKey)] ?: 0) > 1 || failedFlights.contains(inFlightKey(publicationKey)) || local != null && local != encoded).apply()
        nowMillis
    }

    override fun adoptRemote(publicationKey: String, expected: Locator?, locator: Locator, revision: Long): Boolean = synchronized(preferences) {
        if ((inFlight[inFlightKey(publicationKey)] ?: 0) > 0 || preferences.getString(publicationKey, null) != expected?.toJSON()?.toString() || revision < revision(publicationKey)) return@synchronized false
        val encoded = locator.toJSON().toString()
        preferences.edit().putString(publicationKey, encoded).putString(synchronizedKey(publicationKey), encoded)
            .putLong("revision:$publicationKey", revision).putLong(synchronizedAtKey(publicationKey), System.currentTimeMillis())
            .putBoolean(uncertainKey(publicationKey), false).putBoolean(pendingKey(publicationKey), false).apply()
        true
    }

    fun clear(publicationKey: String) = synchronized(preferences) {
        preferences.edit().remove(publicationKey).remove("revision:$publicationKey")
            .remove(pendingKey(publicationKey)).remove(uncertainKey(publicationKey)).remove(synchronizedKey(publicationKey)).remove(synchronizedAtKey(publicationKey)).apply()
    }

    override fun revision(publicationKey: String): Long = preferences.getLong("revision:$publicationKey", 0)

    override fun saveRevision(publicationKey: String, revision: Long) = synchronized(preferences) {
        if (revision > revision(publicationKey)) preferences.edit().putLong("revision:$publicationKey", revision).apply()
    }

    override fun lastSynchronizedAt(publicationKey: String): Long = preferences.getLong(synchronizedAtKey(publicationKey), 0)

    private fun pendingKey(publicationKey: String) = "pending:$publicationKey"
    private fun uncertainKey(publicationKey: String) = "uncertain:$publicationKey"
    private fun synchronizedKey(publicationKey: String) = "synchronized:$publicationKey"
    private fun synchronizedAtKey(publicationKey: String) = "synchronized_at:$publicationKey"
}

internal fun positionNeedsSynchronization(encoded: String, synchronized: String?): Boolean =
    synchronized == null || encoded != synchronized
