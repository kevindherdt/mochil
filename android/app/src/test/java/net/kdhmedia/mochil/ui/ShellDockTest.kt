package net.kdhmedia.mochil.ui

import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.SemanticsProperties
import androidx.compose.ui.test.SemanticsMatcher
import androidx.compose.ui.test.assert
import androidx.compose.ui.test.assertIsSelected
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import net.kdhmedia.mochil.catalogue.LibrarySummary
import net.kdhmedia.mochil.hermes.HermesChatEntry
import net.kdhmedia.mochil.ui.theme.AppThemeSelection
import net.kdhmedia.mochil.ui.theme.MochilTheme
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import org.robolectric.annotation.GraphicsMode

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35], qualifiers = "w390dp-h844dp-xxhdpi")
@GraphicsMode(GraphicsMode.Mode.NATIVE)
class ShellDockTest {
    @get:Rule val compose = createComposeRule()

    private val books = LibrarySummary("b", "Books", "books", 3)
    private val vault = LibrarySummary("v", "Vault", "vault", 9)

    private fun show(
        destination: MobileShellDestination,
        selected: LibrarySummary? = books,
        chat: HermesChatEntry? = null,
        onDestination: (MobileShellDestination) -> Unit = {},
    ) = compose.setContent {
        MochilTheme(AppThemeSelection(), glow = false) { ShellDock(destination, selected, chat, onDestination) }
    }

    @Test fun `without the chat the dock has no AI place`() {
        show(MobileShellDestination.Library)
        compose.onNodeWithText("AI").assertDoesNotExist()
        compose.onNodeWithText("Library").assertIsSelected()
    }

    @Test fun `a place opens its destination`() {
        var opened: MobileShellDestination? = null
        show(MobileShellDestination.Library) { opened = it }
        compose.onNodeWithText("Settings").performClick()
        assertEquals(MobileShellDestination.Settings, opened)
        compose.onNodeWithText("Notes").performClick()
        assertEquals(MobileShellDestination.Notes, opened)
    }

    @Test fun `the AI place starts the chat and leaves the screen alone`() {
        var opened: MobileShellDestination? = null
        var chatOpened = 0
        show(MobileShellDestination.Library, chat = HermesChatEntry("Lyana") { chatOpened++ }) { opened = it }
        compose.onNodeWithText("AI").performClick()
        assertEquals(1, chatOpened)
        assertNull(opened)
    }

    @Test fun `the AI place is announced as a button and the others as tabs`() {
        show(MobileShellDestination.Library, chat = HermesChatEntry("Lyana") {})
        compose.onNodeWithText("AI").assert(SemanticsMatcher.expectValue(SemanticsProperties.Role, Role.Button))
        compose.onNodeWithText("Library").assert(SemanticsMatcher.expectValue(SemanticsProperties.Role, Role.Tab))
    }

    @Test fun `an open vault lights Notes`() {
        show(MobileShellDestination.Library, selected = vault)
        compose.onNodeWithText("Notes").assertIsSelected()
    }
}
