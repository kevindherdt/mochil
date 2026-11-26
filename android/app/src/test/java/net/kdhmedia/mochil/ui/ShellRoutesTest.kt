package net.kdhmedia.mochil.ui

import net.kdhmedia.mochil.catalogue.LibrarySummary
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import androidx.compose.runtime.saveable.SaverScope
import org.junit.Test

class ShellRoutesTest {
    private val books = LibrarySummary("b", "Books", "books", 3)
    private val comics = LibrarySummary("c", "Comics", "comics", 1)
    private val vault = LibrarySummary("v", "Vault", "vault", 9)
    private val otherVault = LibrarySummary("w", "Work notes", "Vault", 2)

    @Test fun `the dock has the web's places in the web's order`() {
        assertEquals(listOf("library", "discover", "ai", "notes", "settings"), dockPlaces(chatAvailable = true).map { it.id })
        assertEquals(listOf("Library", "Discover", "AI", "Notes", "Settings"), dockPlaces(chatAvailable = true).map { it.label })
    }

    @Test fun `an account without the chat gets a four place dock`() {
        assertEquals(listOf("library", "discover", "notes", "settings"), dockPlaces(chatAvailable = false).map { it.id })
    }

    @Test fun `each place routes to its destination and the chat is a launcher`() {
        assertEquals(MobileShellDestination.Library, dockDestination("library"))
        assertEquals(MobileShellDestination.Discover, dockDestination("discover"))
        assertEquals(MobileShellDestination.Notes, dockDestination("notes"))
        assertEquals(MobileShellDestination.Settings, dockDestination("settings"))
        assertNull("the chat opens its own screen", dockDestination("ai"))
    }

    @Test fun `the active place follows the screen`() {
        assertEquals("library", dockActiveId(MobileShellDestination.Library, books))
        assertEquals("library", dockActiveId(MobileShellDestination.Library, null))
        assertEquals("discover", dockActiveId(MobileShellDestination.Discover, books))
        assertEquals("notes", dockActiveId(MobileShellDestination.Notes, null))
        assertEquals("settings", dockActiveId(MobileShellDestination.Settings, books))
    }

    @Test fun `Collections and Sources belong to Library`() {
        assertEquals("library", dockActiveId(MobileShellDestination.Collections, books))
        assertEquals("library", dockActiveId(MobileShellDestination.Sources, books))
    }

    @Test fun `a vault opened from the library menu lights Notes, not Library`() {
        assertEquals("notes", dockActiveId(MobileShellDestination.Library, vault))
        assertEquals("notes", dockActiveId(MobileShellDestination.Library, otherVault))
    }

    @Test fun `Notes opens the vault that is selected, otherwise the first one`() {
        assertEquals(vault, notesLibrary(listOf(books, vault, otherVault), selected = null))
        assertEquals(otherVault, notesLibrary(listOf(books, vault, otherVault), selected = otherVault))
        assertEquals(vault, notesLibrary(listOf(books, vault, otherVault), selected = books))
    }

    @Test fun `no vault means no Notes library`() {
        assertNull(notesLibrary(listOf(books, comics), selected = books))
        assertNull(notesLibrary(emptyList(), selected = null))
    }

    @Test fun `the Library tab keeps the library that is open`() {
        assertEquals(comics, libraryTabTarget(listOf(books, comics, vault), selected = comics, lastLibraryId = "b"))
    }

    @Test fun `the Library tab leaves a vault for the library you were last in`() {
        assertEquals(comics, libraryTabTarget(listOf(books, comics, vault), selected = vault, lastLibraryId = "c"))
    }

    @Test fun `the Library tab falls back to the first library that is not a vault`() {
        assertEquals(books, libraryTabTarget(listOf(vault, books, comics), selected = vault, lastLibraryId = null))
        assertEquals(books, libraryTabTarget(listOf(vault, books, comics), selected = null, lastLibraryId = "gone"))
    }

    @Test fun `the Library tab has nothing to open when every library is a vault`() {
        assertNull(libraryTabTarget(listOf(vault), selected = vault, lastLibraryId = null))
        assertNull(libraryTabTarget(emptyList(), selected = null, lastLibraryId = null))
    }

    @Test fun `Back from a vault needs a book library to go to`() {
        assertTrue(vaultBackAvailable(listOf(books, vault), selected = vault, lastLibraryId = null))
        assertFalse("a book library is selected", vaultBackAvailable(listOf(books, vault), selected = books, lastLibraryId = null))
        assertFalse("only vaults: Back leaves", vaultBackAvailable(listOf(vault, otherVault), selected = vault, lastLibraryId = null))
        assertFalse("nothing selected", vaultBackAvailable(listOf(books, vault), selected = null, lastLibraryId = null))
    }

    @Test fun `Notes is empty only when libraries have loaded and none is a vault`() {
        assertTrue(showsNotesEmpty(listOf(books, comics)))
        assertFalse("not loaded yet", showsNotesEmpty(emptyList()))
        assertFalse(showsNotesEmpty(listOf(books, vault)))
    }

    @Test fun `the saved destination is stored by name and an unknown one falls back to Library`() {
        val scope = object : SaverScope { override fun canBeSaved(value: Any) = true }
        assertEquals("Notes", with(MobileShellDestinationSaver) { scope.save(MobileShellDestination.Notes) })
        assertEquals(MobileShellDestination.Settings, MobileShellDestinationSaver.restore("Settings"))
        assertEquals("a destination from an older build", MobileShellDestination.Library, MobileShellDestinationSaver.restore("Home"))
    }

    @Test fun `the AI place is a launcher and the others are tabs`() {
        assertTrue(dockPlaces(chatAvailable = true).single { it.id == "ai" }.launcher)
        assertEquals(emptyList<String>(), dockPlaces(chatAvailable = true).filter { it.launcher && it.id != "ai" }.map { it.id })
    }

    @Test fun `the dock hides for the wide drawer and while the keyboard is up`() {
        assertTrue(dockVisible(expandedNavigation = false, keyboardVisible = false))
        assertFalse(dockVisible(expandedNavigation = false, keyboardVisible = true))
        assertFalse(dockVisible(expandedNavigation = true, keyboardVisible = false))
    }

    @Test fun `only the pages opened from a menu get the shell's title row, and no dropdown is left on the phone`() {
        assertEquals("Collections", shellPageTitle(MobileShellDestination.Collections))
        // Sources draws its own heading and back arrow, OPDS and WebDAV pages included.
        listOf(MobileShellDestination.Library, MobileShellDestination.Discover, MobileShellDestination.Notes,
            MobileShellDestination.Settings, MobileShellDestination.Sources)
            .forEach { assertNull("$it draws its own header", shellPageTitle(it)) }
    }
}
