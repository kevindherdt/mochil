package net.kdhmedia.mochil.wiki

import kotlin.system.measureNanoTime
import org.junit.Assert.assertEquals
import org.junit.Test

/** Opt-in host-JVM comparison. Timing is evidence, never a flaky pass/fail threshold. */
class WikiSearchBenchmarkTest {
    @Test fun compareRepeatedSearchWithPreviousSortingPolicy() {
        if (System.getenv("MOCHIL_BENCHMARK_WIKI") != "1") return
        val entries = (0 until 5_000).map { number ->
            WikiEntry("page", "$number", "Note ${5_000 - number}.md", "Note ${5_000 - number}", "md")
        }
        val ordered = orderedWikiEntries(entries)
        val index = WikiTreeIndex(entries)
        fun previous(query: String) = ordered.asSequence()
            .filterNot(WikiEntry::isFolder)
            .filter { it.label.contains(query, true) || it.path.contains(query, true) }
            .sortedBy { it.path.lowercase() }.toList()

        // Include the lazy first-search build separately, not hidden in warmup.
        val cold = measureNanoTime { assertEquals(previous("Note"), index.search("Note", emptySet())) }
        println("wiki-search host JVM: 5000 pages; cold comparison (both implementations) ${cold / 1_000_000.0} ms")
        for (query in listOf("Note", "Note 49", "missing")) {
            val expected = previous(query)
            assertEquals(expected, index.search(query, emptySet()))
            repeat(10) { previous(query); index.search(query, emptySet()) }
            fun sample(operation: () -> List<WikiEntry>): Double {
                val times = (0 until 9).map {
                    measureNanoTime { repeat(20) { assertEquals(expected.size, operation().size) } } / 20.0
                }.sorted()
                return times[times.size / 2] / 1_000_000.0
            }
            val oldMs = sample { previous(query) }
            val newMs = sample { index.search(query, emptySet()) }
            println("wiki-search query=$query matches=${expected.size} previous_ms=$oldMs cached_ms=$newMs ratio=${oldMs / newMs}")
        }
    }
}
