package net.kdhmedia.mochil.wiki

import kotlin.random.Random
import org.junit.Assert.assertEquals
import org.junit.Test

class WikiTreeParentLinksTest {
    @Test fun `compact links match all ancestor filtering for randomized expansion`() {
        val entries = buildList {
            for (root in 1..8) {
                add(WikiEntry("folder", path = "Root $root"))
                for (child in 1..8) {
                    val path = "Root $root/Child $child"
                    add(WikiEntry("folder", path = path))
                    repeat(5) { page -> add(WikiEntry("page", "$root-$child-$page", "$path/Page $page.md", format = "md")) }
                }
            }
            add(WikiEntry("page", "root", "Root.md", format = "md"))
        }
        val index = WikiTreeIndex(entries.shuffled(Random(3)))
        val ordered = orderedWikiEntries(entries)
        val random = Random(4)
        repeat(100) {
            val expanded = entries.filter { it.isFolder && random.nextBoolean() }.mapTo(hashSetOf()) { it.path }
            val expected = ordered.filter { entry ->
                entry.path.indices.filter { entry.path[it] == '/' }.all { entry.path.substring(0, it) in expanded }
            }
            assertEquals(expected, index.visible(expanded))
        }
    }

    @Test fun `deep vault retains parent links and respects a collapsed middle ancestor`() {
        val folders = (1..2000).map { depth -> WikiEntry("folder", path = List(depth) { "a" }.joinToString("/")) }
        val leaf = WikiEntry("page", "leaf", "${folders.last().path}/Note.md", format = "md")
        val index = WikiTreeIndex(folders + leaf)
        val expanded = folders.mapTo(hashSetOf()) { it.path }
        assertEquals(folders + leaf, index.visible(expanded))
        expanded.remove(folders[999].path)
        assertEquals(folders.take(1000), index.visible(expanded))
        assertEquals(listOf(leaf), index.search("Note", emptySet()))
    }
}
