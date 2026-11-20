package net.kdhmedia.mochil.wiki

import kotlin.random.Random
import kotlin.system.measureNanoTime
import org.junit.Assert.assertEquals
import org.junit.Test

class WikiNaturalOrderingCostTest {
    @Test fun `prepared sibling keys preserve comparator ordering across mixed names`() {
        val names = listOf("Page 2", "Page 02", "page 002", "Page 10", "Page 0", "Page 00", "Page",
            "", "日本語 2", "日本語 10", "📚 4", "📚 03", "Page 99999999999999999999", "Page 100000000000000000000")
        repeat(20) { seed ->
            val entries = names.mapIndexed { i, name -> WikiEntry("page", "$i", "$i.md", name, "md") }.shuffled(Random(seed))
            assertEquals(previous(entries), orderedWikiEntries(entries))
        }
    }

    @Test fun compareLargeSiblingSortCosts() {
        if (System.getenv("MOCHIL_BENCHMARK_WIKI_ORDER") != "1") return
        val entries = (1..5_000).map { n -> WikiEntry("page", "$n", "Note $n.md", "Note $n", "md") }.shuffled(Random(1))
        val expected = previous(entries)
        assertEquals(expected, orderedWikiEntries(entries))
        repeat(5) { previous(entries); orderedWikiEntries(entries) }
        val oldTimes = mutableListOf<Long>()
        val newTimes = mutableListOf<Long>()
        repeat(11) { round ->
            val old = { oldTimes.add(measureNanoTime { assertEquals(expected, previous(entries)) }) }
            val new = { newTimes.add(measureNanoTime { assertEquals(expected, orderedWikiEntries(entries)) }) }
            if (round % 2 == 0) { old(); new() } else { new(); old() }
        }
        println("wiki-order 5000 siblings host JVM median_ms previous=${oldTimes.sorted()[5] / 1e6} prepared=${newTimes.sorted()[5] / 1e6}")
    }

    private fun previous(entries: List<WikiEntry>) = entries.sortedWith { a, b ->
        naturalWikiNameCompare(a.label, b.label).takeIf { it != 0 } ?: a.path.compareTo(b.path, ignoreCase = true)
    }
}
