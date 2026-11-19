package net.kdhmedia.mochil.reader

import android.os.Build

internal fun currentReaderDeviceName(): String = readerDeviceName(Build.MANUFACTURER, Build.MODEL)

internal fun readerDeviceName(manufacturer: String, model: String): String {
    val maker = manufacturer.trim()
    val product = model.trim()
    val combined = when {
        product.isBlank() -> maker
        maker.isBlank() || product.startsWith(maker, ignoreCase = true) -> product
        else -> "$maker $product"
    }
    return combined.ifBlank { "Android device" }.take(120)
}
