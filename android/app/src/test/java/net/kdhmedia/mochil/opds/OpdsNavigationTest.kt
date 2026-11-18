package net.kdhmedia.mochil.opds

import android.os.Looper
import androidx.test.core.app.ApplicationProvider
import java.util.concurrent.TimeUnit
import kotlinx.coroutines.CompletableDeferred
import net.kdhmedia.mochil.source.SourceItem
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.Shadows.shadowOf
import org.robolectric.annotation.Config

/**
 * On a slow OPDS feed the first request used to swallow everything the viewer did next: the folder row, the search
 * button and the Back arrow were all discarded while `loading` was true, and Back still reported that it had handled
 * the press, so the arrow looked dead. The newest navigation is the one that must own the screen.
 */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35])
class OpdsNavigationTest {
    private fun item(id: String, title: String, navigation: String? = null) =
        SourceItem(id = id, libraryId = "opds:test", title = title, artworkUrl = null, navigationUrl = navigation)

    /** Robolectric's main looper has to be pumped for the ViewModel's main-dispatched state writes to land. */
    private fun settle(model: OpdsViewModel, until: () -> Boolean) {
        val deadline = System.nanoTime() + TimeUnit.SECONDS.toNanos(5)
        while (!until() && System.nanoTime() < deadline) {
            shadowOf(Looper.getMainLooper()).idle()
            Thread.sleep(10)
        }
        shadowOf(Looper.getMainLooper()).idle()
    }

    private fun modelWith(loader: suspend (OpdsConnection, String?, String?) -> OpdsPage): Pair<OpdsViewModel, String> {
        val context = ApplicationProvider.getApplicationContext<android.app.Application>()
        // The real store is encrypted with the device keystore, so the connection list comes through the read seam.
        val feed = OpdsConnection("test", "Test feed", "https://example.test/opds")
        return OpdsViewModel(context, loader, { listOf(feed) }) to feed.id
    }

    @Test
    fun searchTappedWhileAPageIsLoadingReplacesThePageItWasWaitingFor() {
        val heldPage = CompletableDeferred<OpdsPage>()
        val (model, id) = modelWith { _, url, query ->
            if (query == null) heldPage.await() // the first page never arrives on its own
            else OpdsPage("Results", listOf(item("hit", "Search hit")), null, false)
        }
        model.select(id)
        settle(model) { model.state.value.loading }

        model.search("dragon")
        settle(model) { !model.state.value.loading && model.state.value.items.any { it.id == "hit" } }

        assertEquals("the search must own the screen", listOf("hit"), model.state.value.items.map { it.id })
        assertNull("a replaced load must not leave an error behind", model.state.value.message)
        // The abandoned page arriving later must not overwrite the search.
        heldPage.complete(OpdsPage("Root", listOf(item("root", "Root folder")), null, false))
        settle(model) { true }
        assertEquals(listOf("hit"), model.state.value.items.map { it.id })
    }

    @Test
    fun backWhileAPageIsLoadingReturnsToThePreviousPage() {
        val heldFolder = CompletableDeferred<OpdsPage>()
        val (model, id) = modelWith { _, url, _ ->
            if (url == null) OpdsPage("Root", listOf(item("folder", "Folder", "https://example.test/opds/folder")), null, false)
            else heldFolder.await()
        }
        model.select(id)
        settle(model) { model.state.value.items.any { it.id == "folder" } }

        model.openFolder(model.state.value.items.single())
        settle(model) { model.state.value.loading }

        assertTrue("Back must report that it acted", model.back())
        settle(model) { !model.state.value.loading && model.state.value.items.any { it.id == "folder" } }

        assertEquals("Back must go back, not wait for the folder it left", listOf("folder"), model.state.value.items.map { it.id })
        assertEquals("Root", model.state.value.pageTitle)
    }
}
