package net.kdhmedia.mochil.ui

import androidx.compose.foundation.layout.Column
import androidx.compose.material3.Button
import androidx.compose.material3.Text
import androidx.compose.runtime.mutableStateOf
import androidx.compose.ui.test.assertIsFocused
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performSemanticsAction
import androidx.compose.ui.test.performScrollToIndex
import androidx.compose.ui.test.hasScrollToIndexAction
import androidx.compose.ui.semantics.SemanticsActions
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35])
class TvGridFocusBehaviorTest {
    @get:Rule val compose = createComposeRule()

    @Test fun gridEntryRestoresOffscreenRememberedCardWithoutPriorScroll() {
        val positions = TvGridPositionStore().apply { focus("shelf", "30") }
        val requester = androidx.compose.ui.focus.FocusRequester()
        compose.setContent {
            Column {
                Button(onClick = {}) { Text("Search") }
                androidx.compose.runtime.CompositionLocalProvider(LocalTvContentFocus provides requester) {
                    TvMediaGrid(positionKey = "shelf", positions = positions, focusIndex = { it.toIntOrNull() ?: -1 }) {
                        items(40, key = { "$it" }) { index ->
                            TvWideMediaCard("Book $index", "", "", null, {}, focusIdentity = TvGridFocusIdentity("shelf", "$index", positions))
                        }
                    }
                }
            }
        }
        compose.onNodeWithText("Search").performSemanticsAction(SemanticsActions.RequestFocus)
        compose.runOnIdle { requester.requestFocus() }
        compose.onNodeWithText("Book 30").assertIsFocused()
    }

    @Test fun retryFocusWinsWhileOffscreenRestorationIsPending() {
        val positions = TvGridPositionStore().apply { put("shelf", 39, 0); focus("shelf", "0") }
        val requester = androidx.compose.ui.focus.FocusRequester()
        compose.setContent {
            Column {
                Button(onClick = {}) { Text("Search") }
                androidx.compose.runtime.CompositionLocalProvider(LocalTvContentFocus provides requester) {
                    TvMediaGrid(paging = TvPagingState(false, false, {}, error = "Failed", onRetry = {}), positionKey = "shelf", positions = positions, focusIndex = { it.toIntOrNull() ?: -1 }) {
                        items(40, key = { "$it" }) { index ->
                            TvWideMediaCard("Book $index", "", "", null, {}, focusIdentity = TvGridFocusIdentity("shelf", "$index", positions))
                        }
                    }
                }
            }
        }
        compose.onNodeWithText("Search").performSemanticsAction(SemanticsActions.RequestFocus)
        compose.mainClock.autoAdvance = false
        compose.runOnIdle { requester.requestFocus() }
        compose.onNodeWithText("Retry").performSemanticsAction(SemanticsActions.RequestFocus)
        compose.mainClock.advanceTimeBy(1000)
        compose.onNodeWithText("Retry").assertIsFocused()
    }

    @Test fun cancelledRestorationDoesNotOverrideLaterRememberedSelection() {
        val positions = TvGridPositionStore().apply { focus("shelf", "30") }
        val requester = androidx.compose.ui.focus.FocusRequester()
        var calls = 0
        val cancel = mutableStateOf(true)
        compose.setContent {
            Column {
                Button(onClick = {}) { Text("Search") }
                androidx.compose.runtime.CompositionLocalProvider(LocalTvContentFocus provides requester) {
                    TvMediaGrid(positionKey = "shelf", positions = positions, focusIndex = {
                        if (!cancel.value) -1 else if (++calls == 1) 30 else throw kotlinx.coroutines.CancellationException("cancel restoration")
                    }) {
                        items(40, key = { "$it" }) { index ->
                            TvWideMediaCard("Book $index", "", "", null, {}, focusIdentity = TvGridFocusIdentity("shelf", "$index", positions))
                        }
                    }
                }
            }
        }
        compose.onNodeWithText("Search").performSemanticsAction(SemanticsActions.RequestFocus)
        compose.runOnIdle { requester.requestFocus() }
        compose.waitForIdle()
        compose.onNodeWithText("Book 1").performSemanticsAction(SemanticsActions.RequestFocus)
        compose.onNodeWithText("Search").performSemanticsAction(SemanticsActions.RequestFocus)
        compose.runOnIdle { cancel.value = false }
        compose.runOnIdle { requester.requestFocus() }
        compose.onNodeWithText("Book 1").assertIsFocused()
    }

    @Test fun newerSelectionWinsWhileOffscreenRestorationIsPending() {
        val positions = TvGridPositionStore().apply { focus("shelf", "30") }
        val requester = androidx.compose.ui.focus.FocusRequester()
        compose.setContent {
            Column {
                Button(onClick = {}) { Text("Search") }
                androidx.compose.runtime.CompositionLocalProvider(LocalTvContentFocus provides requester) {
                    TvMediaGrid(positionKey = "shelf", positions = positions, focusIndex = { it.toIntOrNull() ?: -1 }) {
                        items(40, key = { "$it" }) { index ->
                            TvWideMediaCard("Book $index", "", "", null, {}, focusIdentity = TvGridFocusIdentity("shelf", "$index", positions))
                        }
                    }
                }
            }
        }
        compose.onNodeWithText("Search").performSemanticsAction(SemanticsActions.RequestFocus)
        compose.mainClock.autoAdvance = false
        compose.runOnIdle { requester.requestFocus() }
        compose.onNodeWithText("Book 1").performSemanticsAction(SemanticsActions.RequestFocus)
        compose.mainClock.advanceTimeBy(1000)
        compose.onNodeWithText("Book 1").assertIsFocused()
    }

    @Test fun pendingRestorationUsesReorderedItemPosition() {
        val positions = TvGridPositionStore().apply { focus("shelf", "30") }
        val requester = androidx.compose.ui.focus.FocusRequester()
        val order = mutableStateOf((0 until 40).map(Int::toString))
        compose.setContent {
            val items = order.value
            Column {
                Button(onClick = {}) { Text("Search") }
                androidx.compose.runtime.CompositionLocalProvider(LocalTvContentFocus provides requester) {
                    TvMediaGrid(positionKey = "shelf", positions = positions, focusIndex = items::indexOf) {
                        items(items.size, key = { items[it] }) { index ->
                            val id = items[index]
                            TvWideMediaCard("Book $id", "", "", null, {}, focusIdentity = TvGridFocusIdentity("shelf", id, positions))
                        }
                    }
                }
            }
        }
        compose.onNodeWithText("Search").performSemanticsAction(SemanticsActions.RequestFocus)
        compose.mainClock.autoAdvance = false
        compose.runOnIdle { requester.requestFocus(); order.value = order.value.filter { it != "30" } + "30" }
        compose.mainClock.advanceTimeBy(1000)
        compose.onNodeWithText("Book 30").assertIsFocused()
    }

    @Test fun leaveAndReenterCancelsEarlierRestoration() {
        val positions = TvGridPositionStore().apply { focus("shelf", "30") }
        val requester = androidx.compose.ui.focus.FocusRequester()
        compose.setContent {
            Column {
                Button(onClick = {}) { Text("Search") }
                androidx.compose.runtime.CompositionLocalProvider(LocalTvContentFocus provides requester) {
                    TvMediaGrid(positionKey = "shelf", positions = positions, focusIndex = { it.toIntOrNull() ?: -1 }) {
                        items(40, key = { "$it" }) { index ->
                            TvWideMediaCard("Book $index", "", "", null, {}, focusIdentity = TvGridFocusIdentity("shelf", "$index", positions))
                        }
                    }
                }
            }
        }
        compose.onNodeWithText("Search").performSemanticsAction(SemanticsActions.RequestFocus)
        compose.mainClock.autoAdvance = false
        compose.runOnIdle { requester.requestFocus() }
        compose.onNodeWithText("Search").performSemanticsAction(SemanticsActions.RequestFocus)
        compose.runOnIdle { requester.requestFocus() }
        compose.mainClock.advanceTimeBy(1000)
        compose.onNodeWithText("Book 0").assertIsFocused()
    }

    @Test fun offscreenRememberedCardDoesNotStealFocusWhenScrolledIntoView() {
        val positions = TvGridPositionStore().apply { focus("shelf", "30") }
        val requester = androidx.compose.ui.focus.FocusRequester()
        compose.setContent {
            Column {
                Button(onClick = {}) { Text("Search") }
                androidx.compose.runtime.CompositionLocalProvider(LocalTvContentFocus provides requester) {
                    TvMediaGrid(positionKey = "shelf", positions = positions) {
                        items(40) { index ->
                            TvWideMediaCard("Book $index", "", "", null, {}, focusIdentity = TvGridFocusIdentity("shelf", "$index", positions))
                        }
                    }
                }
            }
        }
        compose.onNodeWithText("Search").performSemanticsAction(SemanticsActions.RequestFocus)
        compose.onNode(hasScrollToIndexAction()).performScrollToIndex(30)
        compose.onNodeWithText("Search").assertIsFocused()
        compose.runOnIdle { requester.requestFocus() }
        compose.onNodeWithText("Book 30").assertIsFocused()
    }

    @Test fun explicitGridEntryRestoresRememberedCardAfterRecreation() {
        val visible = mutableStateOf(true)
        val positions = TvGridPositionStore().apply { focus("shelf", "second") }
        val requester = androidx.compose.ui.focus.FocusRequester()
        compose.setContent {
            Column {
                Button(onClick = {}) { Text("Search") }
                if (visible.value) androidx.compose.runtime.CompositionLocalProvider(LocalTvContentFocus provides requester) {
                    TvMediaGrid(positionKey = "shelf", positions = positions) {
                        item { TvWideMediaCard("First", "", "", null, {}, focusIdentity = TvGridFocusIdentity("shelf", "first", positions)) }
                        item { TvWideMediaCard("Second", "", "", null, {}, focusIdentity = TvGridFocusIdentity("shelf", "second", positions)) }
                    }
                }
            }
        }
        compose.onNodeWithText("Search").performSemanticsAction(SemanticsActions.RequestFocus)
        compose.runOnIdle { visible.value = false }
        compose.runOnIdle { visible.value = true }
        compose.onNodeWithText("Search").assertIsFocused()
        compose.runOnIdle { requester.requestFocus() }
        compose.onNodeWithText("Second").assertIsFocused()
    }

    @Test fun reorderedItemsRestoreByIdentityNotOldIndex() {
        val positions = TvGridPositionStore().apply { put("shelf", 0, 0); focus("shelf", "30") }
        val requester = androidx.compose.ui.focus.FocusRequester()
        val items = (0 until 40).filter { it != 30 }.map(Int::toString) + "30"
        compose.setContent {
            Column {
                Button(onClick = {}) { Text("Search") }
                androidx.compose.runtime.CompositionLocalProvider(LocalTvContentFocus provides requester) {
                    TvMediaGrid(positionKey = "shelf", positions = positions, focusIndex = items::indexOf) {
                        items(items.size, key = { items[it] }) { index ->
                            val id = items[index]
                            TvWideMediaCard("Book $id", "", "", null, {}, focusIdentity = TvGridFocusIdentity("shelf", id, positions))
                        }
                    }
                }
            }
        }
        compose.onNodeWithText("Search").performSemanticsAction(SemanticsActions.RequestFocus)
        compose.runOnIdle { requester.requestFocus() }
        compose.onNodeWithText("Book 30").assertIsFocused()
    }

    @Test fun missingRememberedItemFallsBackToAvailableCard() {
        val positions = TvGridPositionStore().apply { focus("shelf", "missing") }
        val requester = androidx.compose.ui.focus.FocusRequester()
        compose.setContent {
            Column {
                Button(onClick = {}) { Text("Search") }
                androidx.compose.runtime.CompositionLocalProvider(LocalTvContentFocus provides requester) {
                    TvMediaGrid(positionKey = "shelf", positions = positions, focusIndex = { -1 }) {
                        item { TvWideMediaCard("Available", "", "", null, {}, focusIdentity = TvGridFocusIdentity("shelf", "available", positions)) }
                    }
                }
            }
        }
        compose.onNodeWithText("Search").performSemanticsAction(SemanticsActions.RequestFocus)
        compose.runOnIdle { requester.requestFocus() }
        compose.onNodeWithText("Available").assertIsFocused()
    }

    @Test fun standaloneWideCardRestoresRememberedFocus() {
        val visible = mutableStateOf(false)
        val positions = TvGridPositionStore().apply { focus("queue", "second") }
        compose.setContent {
            Column {
                Button(onClick = {}) { Text("Other") }
                if (visible.value) {
                    TvWideMediaCard("First", "", "", null, {}, focusIdentity = TvGridFocusIdentity("queue", "first", positions))
                    TvWideMediaCard("Second", "", "", null, {}, focusIdentity = TvGridFocusIdentity("queue", "second", positions))
                }
            }
        }
        compose.onNodeWithText("Other").performSemanticsAction(SemanticsActions.RequestFocus)
        compose.runOnIdle { visible.value = true }
        compose.onNodeWithText("Second").assertIsFocused()
    }

    @Test fun returningCardDoesNotStealFocusFromSearch() = returningCardDoesNotStealFocus(false)

    @Test fun returningWideCardDoesNotStealFocusFromSearch() = returningCardDoesNotStealFocus(true)

    private fun returningCardDoesNotStealFocus(wide: Boolean) {
        val visible = mutableStateOf(false)
        val positions = TvGridPositionStore().apply { focus("shelf", "book") }
        compose.setContent {
            Column {
                Button(onClick = {}) { Text("Search") }
                TvMediaGrid(positionKey = "shelf", positions = positions) {
                    if (visible.value) item {
                        val identity = TvGridFocusIdentity("shelf", "book", positions)
                        if (wide) TvWideMediaCard("Book", "", "", null, {}, focusIdentity = identity)
                        else TvMediaCard("Book", "", "", null, TvArtworkRatio.Portrait, {}, focusIdentity = identity)
                    }
                }
            }
        }
        compose.onNodeWithText("Search").performSemanticsAction(SemanticsActions.RequestFocus)
        compose.onNodeWithText("Search").assertIsFocused()
        compose.runOnIdle { visible.value = true }
        compose.onNodeWithText("Search").assertIsFocused()
    }
}
