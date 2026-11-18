package net.kdhmedia.mochil.reader.epub

import java.io.File
import org.junit.Assert.assertTrue
import org.junit.Test

class NarrowStatisticsTest {
    @Test fun statisticColumnsReserveSpaceBetweenLabelAndValue() {
        val source = File("src/main/java/net/kdhmedia/mochil/reader/epub/EpubReadingProgressChrome.kt").readText()
        val row = source.substringAfter("private fun StatisticLine(").substringBefore("private val Theme.progressSurface")
        assertTrue("Statistic columns need a minimum gap", row.contains("Arrangement.spacedBy(16.dp)"))
        assertTrue("Both columns must share constrained width", Regex("Modifier\\.weight\\(1f\\)").findAll(row).count() == 2)
    }
}
