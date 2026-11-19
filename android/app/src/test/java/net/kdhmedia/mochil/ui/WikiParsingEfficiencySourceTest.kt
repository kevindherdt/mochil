package net.kdhmedia.mochil.ui

import java.nio.file.Files
import java.nio.file.Paths
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class WikiParsingEfficiencySourceTest {
    @Test
    fun pageParsingRunsOnceOffTheMainThreadAndFeedsTheReader() {
        val screen = Files.readString(Paths.get("src/main/java/net/kdhmedia/mochil/ui/WikiScreen.kt"))
        val view = Files.readString(Paths.get("src/main/java/net/kdhmedia/mochil/ui/WikiMarkdownView.kt"))
        assertTrue(screen.contains("withContext(Dispatchers.Default)"))
        assertTrue(screen.contains("PerformanceDiagnostics.measure(\"reader\", \"format_wiki_page\")"))
        assertTrue(screen.contains("parseWikiMarkdownDocument(page.content) { context.ensureActive() }"))
        assertTrue(screen.contains("blocks = document!!.blocks"))
        assertFalse(view.contains("parseWikiMarkdown(content)"))
    }

    @Test
    fun backlinksAreVirtualizedInsteadOfComposedInsideOneLazyItem() {
        val view = Files.readString(Paths.get("src/main/java/net/kdhmedia/mochil/ui/WikiMarkdownView.kt"))
        assertTrue(view.contains("items(\n                items = backlinks"))
        assertFalse(view.contains("backlinks.forEach"))
    }

    @Test
    fun vaultTreeSortingAndFilteringStayOffTheUiThread() {
        val screen = Files.readString(Paths.get("src/main/java/net/kdhmedia/mochil/ui/WikiScreen.kt"))
        assertTrue(screen.contains("PerformanceDiagnostics.measure(\"reader\", \"index_wiki_tree\")"))
        assertTrue(screen.contains("PerformanceDiagnostics.measure(\"reader\", \"format_wiki_tree\")"))
        assertTrue(screen.contains("else treeIndex?.search(query, state.expanded).orEmpty()"))
    }
}
