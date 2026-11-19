package net.kdhmedia.mochil.reader.comic

import androidx.compose.foundation.MutatePriority
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.pager.rememberPagerState
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.*
import kotlinx.coroutines.flow.emptyFlow
import org.junit.Assert.*
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35])
class ComicRemoteRequestBehaviorTest {
    @get:Rule val compose = createComposeRule()
    private val source = object : ComicPageSource {
        override val pageCount = 9
        override fun page(index: Int) = byteArrayOf()
        override fun close() {}
    }

    @Test fun pagedInitialRequestNeverPublishesSavedPage() {
        var request by mutableStateOf<Int?>(7)
        val observed = mutableListOf<Int>()
        compose.setContent {
            Box(Modifier.width(300.dp).height(500.dp)) {
                PagedComic(source, "test", 2, request, { request = null }, false, PageFit.Fit, emptyFlow(), {}) { observed.add(it) }
            }
        }
        compose.waitForIdle()
        assertEquals(listOf(7), observed)
    }

    @Test fun pagedInterruptedRequestRetriesWithoutMovement() {
        var request by mutableStateOf<Int?>(null)
        var consumed = 0
        val observed = mutableListOf<Int>()
        lateinit var scope: CoroutineScope
        lateinit var blocker: Job
        compose.setContent {
            val pager = rememberPagerState(initialPage = 2, pageCount = { source.pageCount })
            scope = rememberCoroutineScope()
            Box(Modifier.width(300.dp).height(500.dp)) {
                PagedComic(source, "test", 2, request, { consumed++; request = null }, false, PageFit.Fit, emptyFlow(), {}, pager) { observed.add(it) }
            }
            LaunchedEffect(pager) { blocker = scope.launch { pager.scroll(MutatePriority.PreventUserInput) { awaitCancellation() } } }
        }
        compose.waitForIdle()
        compose.runOnIdle { observed.clear(); request = 7 }
        compose.waitForIdle()
        assertEquals(0, consumed)
        assertTrue(observed.isEmpty())
        compose.runOnIdle { blocker.cancel() }
        compose.waitUntil(5000) { consumed == 1 }
        compose.waitForIdle()
        assertEquals(listOf(7), observed)
    }

    @Test fun replacingBlockedRequestOnlyAcknowledgesLatest() {
        var request by mutableStateOf<Int?>(null)
        var consumed = 0
        val observed = mutableListOf<Int>()
        lateinit var scope: CoroutineScope
        lateinit var blocker: Job
        compose.setContent {
            val list = rememberLazyListState(initialFirstVisibleItemIndex = 2)
            scope = rememberCoroutineScope()
            Box(Modifier.width(300.dp).height(500.dp)) {
                WebtoonComic(source, "test", 2, request, { consumed++; request = null }, emptyFlow(), {}, { observed.add(it) }, list) { _, _ -> Box(Modifier.height(600.dp)) }
            }
            LaunchedEffect(list) { blocker = scope.launch { list.scroll(MutatePriority.PreventUserInput) { awaitCancellation() } } }
        }
        compose.waitForIdle()
        compose.runOnIdle { observed.clear(); request = 4 }
        compose.waitForIdle()
        compose.runOnIdle { request = 7 }
        compose.waitForIdle()
        assertEquals(0, consumed)
        compose.runOnIdle { blocker.cancel() }
        compose.waitUntil(5000) { consumed == 1 }
        compose.waitForIdle()
        assertEquals(listOf(7), observed)
    }

    @Test fun webtoonInterruptedRequestWaitsAndRetriesWithoutMovement() {
        var request by mutableStateOf<Int?>(null)
        var consumed = 0
        val observed = mutableListOf<Int>()
        lateinit var scope: CoroutineScope
        lateinit var blocker: Job
        compose.setContent {
            val list = rememberLazyListState(initialFirstVisibleItemIndex = 2)
            scope = rememberCoroutineScope()
            Box(Modifier.width(300.dp).height(500.dp)) {
                WebtoonComic(source, "test", 2, request, { consumed++; request = null }, emptyFlow(), {}, { observed.add(it) }, list) { _, _ -> Box(Modifier.height(600.dp)) }
            }
            LaunchedEffect(list) { blocker = scope.launch { list.scroll(MutatePriority.PreventUserInput) { awaitCancellation() } } }
        }
        compose.waitForIdle()
        compose.runOnIdle { observed.clear(); request = 7 }
        compose.waitForIdle()
        assertEquals("cancelled scroll must not acknowledge", 0, consumed)
        assertTrue("pending request must suppress stale progress", observed.isEmpty())
        compose.runOnIdle { blocker.cancel() }
        compose.waitUntil(5000) { consumed == 1 }
        compose.waitForIdle()
        assertEquals(listOf(7), observed)
    }
}
