package net.kdhmedia.mochil.reader.comic

import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.MutatePriority
import androidx.compose.foundation.lazy.LazyListState
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.runtime.rememberCoroutineScope
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.awaitCancellation
import kotlinx.coroutines.launch
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.width
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.test.onNodeWithTag
import androidx.compose.ui.test.getUnclippedBoundsInRoot
import org.junit.Assert.assertTrue
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.flow.emptyFlow
import org.junit.Assert.assertEquals
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35])
class WebtoonPositionBehaviorTest {
    @get:Rule val compose = createComposeRule()
    private val source = object : ComicPageSource {
        override val pageCount = 9
        override fun page(index: Int) = byteArrayOf()
        override fun close() {}
    }

    @Test fun pendingRemoteRequestDoesNotPublishOldPosition() {
        val observed = mutableListOf<Int>()
        var consumed = false
        var request by mutableStateOf<Int?>(8)
        compose.setContent {
            Box(Modifier.width(300.dp).height(500.dp)) {
                WebtoonComic(source, "test", 4, request, { consumed = true; request = null }, emptyFlow(), {}, { observed.add(it) }) { _, placeholder ->
                    Box(placeholder)
                }
            }
        }
        compose.waitForIdle()
        assertEquals(true, consumed)
        assertEquals(listOf(8), observed)
    }

    @Test fun remoteRequestAfterReadingReportsOnlyDestination() {
        var request by mutableStateOf<Int?>(null)
        val observed = mutableListOf<Int>()
        var consumed = 0
        compose.setContent {
            Box(Modifier.width(300.dp).height(500.dp)) {
                WebtoonComic(source, "test", 3, request, { consumed++; request = null }, emptyFlow(), {}, { observed.add(it) }) { _, placeholder ->
                    Box(placeholder)
                }
            }
        }
        compose.waitForIdle()
        assertEquals(listOf(3), observed)
        compose.runOnIdle { observed.clear(); request = 8 }
        compose.waitForIdle()
        assertEquals(1, consumed)
        assertEquals(8, observed.last())
        assertEquals(true, observed.all { it == 8 })
    }

    @Test fun restoringLastShortPageSurvivesImageLoading() {
        var loaded by mutableStateOf(false)
        val observed = mutableListOf<Int>()
        compose.setContent {
            Box(Modifier.width(300.dp).height(500.dp)) {
                WebtoonComic(source, "test", 8, null, {}, emptyFlow(), {}, { observed.add(it) }) { _, placeholder ->
                    Box(if (loaded) Modifier.height(100.dp) else placeholder)
                }
            }
        }
        compose.waitForIdle()
        assertEquals(8, observed.last())
        compose.runOnIdle { loaded = true }
        compose.waitForIdle()
        assertEquals("loading short pages must not move saved progress", 8, observed.last())
    }
    @Test fun shortIntermediatePagesAreContinuous() {
        compose.setContent {
            Box(Modifier.width(300.dp).height(500.dp)) {
                WebtoonComic(source, "test", 0, null, {}, emptyFlow(), {}, {}) { page, _ ->
                    Box(Modifier.height(100.dp).testTag("page$page"))
                }
            }
        }
        compose.waitForIdle()
        val first = compose.onNodeWithTag("page0").getUnclippedBoundsInRoot()
        val second = compose.onNodeWithTag("page1").getUnclippedBoundsInRoot()
        assertTrue("short pages must touch: $first then $second", kotlin.math.abs((second.top - first.bottom).value) < 1f)
    }

    @Test fun interruptedRequestDoesNotStopProgressReporting() {
        var request by mutableStateOf<Int?>(null)
        val observed = mutableListOf<Int>()
        lateinit var list: LazyListState
        lateinit var scope: CoroutineScope
        lateinit var blocker: Job
        compose.setContent {
            list = rememberLazyListState()
            scope = rememberCoroutineScope()
            Box(Modifier.width(300.dp).height(500.dp)) {
                WebtoonComic(source, "test", 0, request, { request = null }, emptyFlow(), {}, { observed.add(it) }, list) { _, _ ->
                    Box(Modifier.height(600.dp))
                }
            }
        }
        compose.waitForIdle()
        compose.runOnIdle {
            blocker = scope.launch { list.scroll(MutatePriority.PreventUserInput) { awaitCancellation() } }
        }
        compose.waitForIdle()
        compose.runOnIdle { request = 4 }
        compose.waitForIdle()
        compose.runOnIdle { blocker.cancel() }
        compose.waitUntil(5000) { request == null }
        compose.runOnIdle { scope.launch { list.scrollToItem(2) } }
        compose.waitForIdle()
        assertEquals("progress must continue after a scroll request is interrupted", 2, observed.last())
    }

}
