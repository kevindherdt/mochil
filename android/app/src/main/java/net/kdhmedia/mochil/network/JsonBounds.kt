package net.kdhmedia.mochil.network

import java.io.IOException
import org.json.JSONArray

internal fun JSONArray.requireAtMost(maximum: Int, label: String): JSONArray {
    require(maximum >= 0)
    if (length() > maximum) throw IOException("$label returned too many items.")
    return this
}
