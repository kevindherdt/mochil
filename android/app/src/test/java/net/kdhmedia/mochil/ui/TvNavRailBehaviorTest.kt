package net.kdhmedia.mochil.ui

import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.material3.Button
import androidx.compose.material3.Text
import androidx.compose.ui.Modifier
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.test.getUnclippedBoundsInRoot
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithTag
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performSemanticsAction
import androidx.compose.ui.test.performKeyInput
import androidx.compose.ui.test.performClick
import androidx.compose.ui.test.assertIsFocused
import androidx.compose.ui.input.key.Key
import net.kdhmedia.mochil.catalogue.LibrarySummary
import androidx.compose.runtime.mutableStateOf
import androidx.compose.ui.test.onNodeWithContentDescription
import androidx.compose.ui.semantics.SemanticsActions
import androidx.compose.ui.unit.dp
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35])
class TvNavRailBehaviorTest {
    @get:Rule val compose = createComposeRule()

    @Test fun collapsedRailIsCompactAndExpandsOnFocus() {
        val requester = FocusRequester()
        compose.setContent {
            Box(Modifier.fillMaxSize()) {
                Box(Modifier.padding(start = TvNavRailWidth).testTag("content")) {
                    Button(onClick = {}) { Text("Content") }
                }
                TvNavRail(emptyList(), TV_RAIL_HOME, {}, {}, {}, {}, requester, Modifier.testTag("rail"))
            }
        }
        compose.onNodeWithText("Content").performSemanticsAction(SemanticsActions.RequestFocus)
        val collapsedBounds = compose.onNodeWithTag("rail").getUnclippedBoundsInRoot()
        val collapsed = collapsedBounds.right - collapsedBounds.left
        // 48dp on the resting rail: the sheets put it near 44dp at ten feet, and a 20dp glyph needs the air.
        assertTrue("Collapsed rail must be 48dp: $collapsed", collapsed == 48.dp)
        val home = compose.onNodeWithContentDescription("Home").getUnclippedBoundsInRoot()
        val settings = compose.onNodeWithContentDescription("Settings").getUnclippedBoundsInRoot()
        // The glyph is decorative now: the destination's name is the row's description, so a remote reads "Home"
        // whether or not the rail has opened. The row is what Carries the name.
        val named = compose.onNodeWithContentDescription("Home")
        named.assertExists()
        assertTrue("Collapsed row must be 36dp: $home", home.bottom - home.top == 36.dp)
        assertTrue("Collapsed destinations should form a short floating group: $home, $settings", settings.bottom - home.top <= 110.dp)
        assertTrue("Collapsed group should float away from the top: $home", home.top >= 80.dp)
        val contentBefore = compose.onNodeWithTag("content").getUnclippedBoundsInRoot()
        compose.runOnIdle { requester.requestFocus() }
        val expandedBounds = compose.onNodeWithTag("rail").getUnclippedBoundsInRoot()
        val expanded = expandedBounds.right - expandedBounds.left
        assertTrue("Expanded rail must remain 224dp: $expanded", expanded == 224.dp)
        val expandedHome = compose.onNodeWithText("Home").getUnclippedBoundsInRoot()
        assertTrue("Expanded rows must be 40dp: $expandedHome", expandedHome.bottom - expandedHome.top == 40.dp)
        val contentAfter = compose.onNodeWithTag("content").getUnclippedBoundsInRoot()
        assertTrue("Content must not move when rail opens", contentBefore == contentAfter)
        assertTrue("Open rail overlays content", expandedBounds.right > contentAfter.left)
        compose.onNodeWithText("Content").performSemanticsAction(SemanticsActions.RequestFocus)
        val after = compose.onNodeWithTag("rail").getUnclippedBoundsInRoot()
        assertTrue("Rail must collapse after focus leaves", after.right - after.left == 48.dp)
    }

    @Test fun selectedLibraryRegainsFocusAfterReturningFromContent() {
        val requester = FocusRequester()
        val selected = mutableStateOf(TV_RAIL_HOME)
        val library = LibrarySummary("books", "Books", "books", 0)
        compose.setContent {
            Box(Modifier.fillMaxSize()) {
                Box(Modifier.padding(start = TvNavRailWidth)) { Button(onClick = {}) { Text("Content") } }
                TvNavRail(listOf(library), selected.value, { selected.value = TV_RAIL_HOME },
                    { selected.value = tvRailLibraryKey(it.id) }, { selected.value = TV_RAIL_SETTINGS }, {}, requester)
            }
        }
        compose.runOnIdle { requester.requestFocus() }
        compose.onNodeWithText("Home").performKeyInput { keyDown(Key.DirectionDown); keyUp(Key.DirectionDown) }
        compose.onNodeWithText("Books").assertIsFocused()
        compose.onNodeWithText("Books").performKeyInput { keyDown(Key.DirectionCenter); keyUp(Key.DirectionCenter) }
        compose.runOnIdle { assertTrue(selected.value == tvRailLibraryKey("books")) }
        compose.onNodeWithText("Content").performSemanticsAction(SemanticsActions.RequestFocus)
        compose.onNodeWithContentDescription("Books").assertExists()
        compose.runOnIdle { requester.requestFocus() }
        compose.onNodeWithText("Books").assertIsFocused()
    }

    @Test fun selectedEntryAndBackRetainExistingNavigation() {
        val requester = FocusRequester()
        var settings = 0
        var backs = 0
        compose.setContent {
            TvNavRail(emptyList(), TV_RAIL_HOME, {}, {}, { settings++ }, { backs++ }, requester)
        }
        compose.runOnIdle { requester.requestFocus() }
        compose.onNodeWithText("Home").assertIsFocused()
        compose.onNodeWithText("Settings").performClick()
        compose.runOnIdle { assertTrue("Settings should navigate", settings == 1) }
        compose.onNodeWithText("Home").performKeyInput { keyDown(Key.Back); keyUp(Key.Back) }
        compose.runOnIdle { assertTrue("Back should invoke shell navigation once", backs == 1) }
    }

    @Test fun manyLibrariesRemainReachableOnShortScreen() {
        val requester = FocusRequester()
        val libraries = (1..12).map { LibrarySummary("$it", "Library $it", "books", 0) }
        var chosen = ""
        compose.setContent {
            Box(Modifier.size(640.dp, 300.dp)) {
                TvNavRail(libraries, TV_RAIL_HOME, {}, { chosen = it.id }, {}, {}, requester)
            }
        }
        compose.runOnIdle { requester.requestFocus() }
        for (index in 0..11) {
            val current = if (index == 0) "Home" else "Library $index"
            compose.onNodeWithText(current).performKeyInput { keyDown(Key.DirectionDown); keyUp(Key.DirectionDown) }
            compose.onNodeWithText("Library ${index + 1}").assertIsFocused()
        }
        compose.onNodeWithText("Library 12").performKeyInput { keyDown(Key.DirectionCenter); keyUp(Key.DirectionCenter) }
        compose.runOnIdle { assertTrue("Last library must remain selectable", chosen == "12") }
        compose.onNodeWithText("Library 12").performKeyInput { keyDown(Key.DirectionDown); keyUp(Key.DirectionDown) }
        compose.onNodeWithText("Settings").assertIsFocused()
    }

    @Test fun changingSelectedLibraryWhileContentFocusedRestoresOffscreenSelection() {
        val requester = FocusRequester()
        val selected = mutableStateOf(TV_RAIL_HOME)
        val libraries = (1..12).map { LibrarySummary("$it", "Library $it", "books", 0) }
        compose.setContent {
            Box(Modifier.size(640.dp, 300.dp)) {
                Button(onClick = {}) { Text("Content") }
                TvNavRail(libraries, selected.value, {}, {}, {}, {}, requester)
            }
        }
        compose.onNodeWithText("Content").performSemanticsAction(SemanticsActions.RequestFocus)
        compose.runOnIdle { selected.value = tvRailLibraryKey("12") }
        compose.onNodeWithText("Content").assertIsFocused()
        compose.runOnIdle { requester.requestFocus() }
        compose.onNodeWithText("Library 12").assertIsFocused()
    }

    @Test fun reorderedLibrariesRestoreSelectedDestinationWithoutStealingContentFocus() {
        val requester = FocusRequester()
        val all = (1..12).map { LibrarySummary("$it", "Library $it", "books", 0) }
        val libraries = mutableStateOf(all.reversed())
        compose.setContent {
            Box(Modifier.size(640.dp, 300.dp)) {
                Button(onClick = {}) { Text("Content") }
                TvNavRail(libraries.value, tvRailLibraryKey("12"), {}, {}, {}, {}, requester)
            }
        }
        compose.onNodeWithText("Content").performSemanticsAction(SemanticsActions.RequestFocus)
        compose.runOnIdle { libraries.value = all }
        compose.onNodeWithText("Content").assertIsFocused()
        compose.runOnIdle { requester.requestFocus() }
        compose.onNodeWithText("Library 12").assertIsFocused()
    }

    @Test fun missingSelectionFallsBackAndLaterRestoresWhenLibraryAppears() {
        val requester = FocusRequester()
        val all = (1..12).map { LibrarySummary("$it", "Library $it", "books", 0) }
        val libraries = mutableStateOf(all.dropLast(1))
        compose.setContent {
            Box(Modifier.size(640.dp, 300.dp)) {
                Button(onClick = {}) { Text("Content") }
                TvNavRail(libraries.value, tvRailLibraryKey("12"), {}, {}, {}, {}, requester)
            }
        }
        compose.runOnIdle { requester.requestFocus() }
        compose.onNodeWithText("Home").assertIsFocused()
        compose.onNodeWithText("Content").performSemanticsAction(SemanticsActions.RequestFocus)
        compose.runOnIdle { libraries.value = all }
        compose.onNodeWithText("Content").assertIsFocused()
        compose.runOnIdle { requester.requestFocus() }
        compose.onNodeWithText("Library 12").assertIsFocused()
    }

    @Test fun latestSelectionWinsAfterRapidChangesAndReturnFromRail() {
        val requester = FocusRequester()
        val selected = mutableStateOf(TV_RAIL_HOME)
        val libraries = (1..20).map { LibrarySummary("$it", "Library $it", "books", 0) }
        compose.setContent {
            Box(Modifier.size(640.dp, 300.dp)) {
                Button(onClick = {}) { Text("Content") }
                TvNavRail(libraries, selected.value, {}, {}, {}, {}, requester)
            }
        }
        compose.runOnIdle { requester.requestFocus() }
        compose.onNodeWithText("Home").assertIsFocused()
        compose.runOnIdle { selected.value = tvRailLibraryKey("20") }
        compose.onNodeWithText("Home").assertIsFocused()
        compose.onNodeWithText("Content").performSemanticsAction(SemanticsActions.RequestFocus)
        compose.runOnIdle {
            selected.value = tvRailLibraryKey("1")
            selected.value = tvRailLibraryKey("15")
        }
        compose.onNodeWithText("Content").assertIsFocused()
        compose.runOnIdle { requester.requestFocus() }
        compose.onNodeWithText("Library 15").assertIsFocused()
    }

    @Test fun offscreenSelectedLibraryIsFocusedOnEntry() {
        val requester = FocusRequester()
        val libraries = (1..12).map { LibrarySummary("$it", "Library $it", "books", 0) }
        compose.setContent {
            Box(Modifier.size(640.dp, 300.dp)) {
                TvNavRail(libraries, tvRailLibraryKey("12"), {}, {}, {}, {}, requester)
            }
        }
        compose.runOnIdle { requester.requestFocus() }
        compose.onNodeWithText("Library 12").assertIsFocused()
    }
}
