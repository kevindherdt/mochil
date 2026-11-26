package net.kdhmedia.mochil.testing

import android.graphics.Bitmap
import android.graphics.BitmapFactory
import androidx.compose.ui.graphics.asAndroidBitmap
import androidx.compose.ui.test.SemanticsNodeInteraction
import androidx.compose.ui.test.captureToImage
import java.io.File
import kotlin.math.abs

/**
 * Screenshot goldens without a plugin: render a node, compare it to `src/test/screenshots/<name>.png`.
 *
 *  - A normal run verifies. A missing or different golden fails, and the actual image is written to
 *    `build/goldens/<name>.actual.png` so the difference can be looked at rather than guessed at.
 *  - `scripts/android.sh … -PrecordGoldens=true` rewrites the goldens. Look at the new images before committing them.
 *
 * Needs `@GraphicsMode(GraphicsMode.Mode.NATIVE)` on the test class: legacy graphics mode draws no real text.
 */
internal object Goldens {
    private val record = System.getProperty("mochil.goldens") == "record"
    private val dir = File("src/test/screenshots")
    private val actualDir = File("build/goldens")

    /** A channel may differ by this much (anti-aliasing), on at most [MAX_DIFFERING] of the pixels. */
    private const val CHANNEL_TOLERANCE = 6
    private const val MAX_DIFFERING = 0.002

    fun assertMatches(node: SemanticsNodeInteraction, name: String) {
        val actual = node.captureToImage().asAndroidBitmap()
        val golden = File(dir, "$name.png")
        if (record) {
            dir.mkdirs()
            golden.outputStream().use { actual.compress(Bitmap.CompressFormat.PNG, 100, it) }
            return
        }
        if (!golden.isFile) fail(name, actual, "no golden at ${golden.path}; record it with -PrecordGoldens=true and look at it")
        val expected = BitmapFactory.decodeFile(golden.path) ?: fail(name, actual, "${golden.path} is not a readable PNG")
        if (expected.width != actual.width || expected.height != actual.height) {
            fail(name, actual, "size ${actual.width}x${actual.height}, golden is ${expected.width}x${expected.height}")
        }
        val a = IntArray(actual.width * actual.height).also { actual.getPixels(it, 0, actual.width, 0, 0, actual.width, actual.height) }
        val e = IntArray(a.size).also { expected.getPixels(it, 0, expected.width, 0, 0, expected.width, expected.height) }
        val differing = a.indices.count { i -> channelDelta(a[i], e[i]) > CHANNEL_TOLERANCE }
        if (differing > a.size * MAX_DIFFERING) fail(name, actual, "$differing of ${a.size} pixels differ from ${golden.path}")
    }

    private fun channelDelta(x: Int, y: Int): Int = maxOf(
        abs((x shr 24 and 0xFF) - (y shr 24 and 0xFF)),
        abs((x shr 16 and 0xFF) - (y shr 16 and 0xFF)),
        abs((x shr 8 and 0xFF) - (y shr 8 and 0xFF)),
        abs((x and 0xFF) - (y and 0xFF)),
    )

    private fun fail(name: String, actual: Bitmap, why: String): Nothing {
        actualDir.mkdirs()
        File(actualDir, "$name.actual.png").outputStream().use { actual.compress(Bitmap.CompressFormat.PNG, 100, it) }
        throw AssertionError("Golden '$name': $why (actual: build/goldens/$name.actual.png)")
    }
}
