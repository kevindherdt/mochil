package net.kdhmedia.mochil.diagnostics

import java.io.File
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class PerformanceMetricSourceTest {
    @Test fun `foreground duration is not mislabeled as startup timing`() {
        val main = File("src/main/java/net/kdhmedia/mochil/MainActivity.kt").readText()
        val frame = File("src/main/java/net/kdhmedia/mochil/ui/AppPerformance.kt").readText()
        assertTrue(main.contains("\"foreground_session\""))
        assertFalse(main.contains("record(\"lifecycle\", \"app_foreground\""))
        assertTrue(frame.contains("withFrameNanos"))
        assertTrue(frame.contains("\"mobile_first_frame\""))
        assertTrue(frame.contains("\"tv_app_first_frame\""))
    }

    @Test fun `recording diagnostics does not synchronously commit every event`() {
        val source = File("src/main/java/net/kdhmedia/mochil/diagnostics/PerformanceDiagnostics.kt").readText()
        val record = source.substringAfter("fun record(").substringBefore("fun batch(")
        assertTrue(record.contains("persist(values)"))
        assertFalse(record.contains(".commit()"))
        assertTrue(source.contains("cachedEvents"))
    }
}
