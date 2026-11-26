package net.kdhmedia.mochil.ui

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.width
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.assertIsSelected
import androidx.compose.ui.test.getUnclippedBoundsInRoot
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithContentDescription
import androidx.compose.ui.test.onNodeWithTag
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.compose.ui.unit.Density
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.height
import net.kdhmedia.mochil.catalogue.BookStatusFilter
import net.kdhmedia.mochil.catalogue.LibrarySummary
import net.kdhmedia.mochil.testing.Goldens
import net.kdhmedia.mochil.ui.kit.LucideIcons
import net.kdhmedia.mochil.ui.theme.AppPalette
import net.kdhmedia.mochil.ui.theme.AppThemeMode
import net.kdhmedia.mochil.ui.theme.AppThemeSelection
import net.kdhmedia.mochil.ui.theme.MochilChrome
import net.kdhmedia.mochil.ui.theme.MochilTheme
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import org.robolectric.annotation.GraphicsMode

/**
 * The Library header on the phone: the libraries as pills, the status chips and the view / search buttons under them. On the web
 * (390px) the band is 80px tall with the covers starting at 96. Here every tappable row is a 44dp target, so two rows are 88dp;
 * the cover grid's own padding (taken down to the header's gutter in step 4) brings the covers into line.
 */
@RunWith(RobolectricTestRunner::class)
@GraphicsMode(GraphicsMode.Mode.NATIVE)
@Config(sdk = [35], qualifiers = "w390dp-h844dp-xxhdpi")
class LibraryHeaderTest {
    @get:Rule val compose = createComposeRule()

    private val books = LibrarySummary("books", "Books", "books", 40)
    private val novels = LibrarySummary("lightnovels", "Light Novels", "books", 12)
    private val anime = LibrarySummary("anime", "Anime", "video", 9)
    private val music = LibrarySummary("music", "Music", "music", 500)
    private val vault = LibrarySummary("notes", "Notes vault", "vault", 48)
    private val all = listOf(books, novels, anime, music, vault)

    private val actions = listOf(
        HeaderAction(LucideIcons.Search, "Search and filter", {}),
        HeaderAction(LucideIcons.Layers, "View options", {}),
    )

    @Composable
    private fun Host(palette: AppPalette = AppPalette.Tide, mode: AppThemeMode = AppThemeMode.Dark, content: @Composable () -> Unit) {
        MochilTheme(AppThemeSelection(palette, mode), glow = false) {
            Surface(Modifier.width(390.dp), color = MaterialTheme.colorScheme.background) { Column(Modifier.testTag("host")) { content() } }
        }
    }

    @Composable
    private fun Header(
        selected: LibrarySummary? = books,
        libraries: List<LibrarySummary> = all,
        chips: StatusChips? = StatusChips(BookStatusFilter.All) {},
        actions: List<HeaderAction>? = this.actions,
        offline: String? = null,
        onSelect: (LibrarySummary) -> Unit = {},
        onRetry: () -> Unit = {},
    ) = LibraryHeader(libraries, selected, onSelect, chips, actions, offline, onRetry, Modifier.testTag("header"))

    @Test fun `every library is a pill, the open one is chosen, and a tap opens another`() {
        var opened: LibrarySummary? = null
        compose.setContent { Host { Header(onSelect = { opened = it }) } }
        compose.onNodeWithText("Books").assertIsSelected()
        compose.onNodeWithText("Anime").performClick()
        assertEquals(anime, opened)
    }

    @Test fun `a notes vault is not a shelf to browse, so it has no pill`() {
        compose.setContent { Host { Header() } }
        compose.onNodeWithText("Notes vault").assertDoesNotExist()
    }

    @Test fun `with a vault open there is nothing to switch and no row of tools, even when the screen offers tools`() {
        compose.setContent { Host { Header(selected = vault, chips = null) } }
        compose.onNodeWithText("Books").assertDoesNotExist()
        compose.onNodeWithContentDescription("View options").assertDoesNotExist()
        assertEquals("an empty header costs no height", 0f, compose.onNodeWithTag("header").getUnclippedBoundsInRoot().height.value, 0.5f)
    }

    @Test fun `the status chips show the chosen one and report a tap`() {
        var chosen: BookStatusFilter? = null
        compose.setContent { Host { Header(chips = StatusChips(BookStatusFilter.InProgress) { chosen = it }) } }
        compose.onNodeWithText("In Progress").assertIsSelected()
        compose.onNodeWithText("Unread").performClick()
        assertEquals(BookStatusFilter.Unread, chosen)
    }

    @Test fun `a screen that offers no chips still gets its tools on the same row`() {
        compose.setContent { Host { Header(chips = null) } }
        compose.onNodeWithText("In Progress").assertDoesNotExist()
        compose.onNodeWithContentDescription("View options").assertIsDisplayed()
    }

    @Test fun `the screen's own buttons sit at the right of the chip row`() {
        var taps = 0
        compose.setContent { Host { Header(actions = listOf(HeaderAction(LucideIcons.Layers, "View options", { taps++ }))) } }
        compose.onNodeWithContentDescription("View options").performClick()
        assertEquals(1, taps)
        val chip = compose.onNodeWithText("Unread").getUnclippedBoundsInRoot()
        val button = compose.onNodeWithContentDescription("View options").getUnclippedBoundsInRoot()
        assertTrue("the buttons are to the right of the chips", button.left >= chip.right)
        assertEquals("on the same row", chip.top.value.toInt() / 10, button.top.value.toInt() / 10)
    }

    @Test fun `two tappable rows make the band 88dp, which is the web's 80 plus the cost of 44dp targets`() {
        compose.setContent { Host { Header() } }
        val bounds = compose.onNodeWithTag("header").getUnclippedBoundsInRoot()
        assertEquals(2 * MochilChrome.touchMin.value, bounds.height.value, 0.5f)
    }

    @Test fun `the row of tools keeps its height when a screen has nothing to put in it yet`() {
        compose.setContent { Host { Header(chips = null, actions = null) } }
        val bounds = compose.onNodeWithTag("header").getUnclippedBoundsInRoot()
        assertEquals("a library that has no tools yet must not make the covers jump when they arrive", 2 * MochilChrome.touchMin.value, bounds.height.value, 0.5f)
    }

    @Test fun `an offline library says so and offers the retry`() {
        var retries = 0
        compose.setContent { Host { Header(chips = null, offline = "Offline · local downloads only", onRetry = { retries++ }) } }
        compose.onNodeWithText("Offline · local downloads only").assertIsDisplayed()
        compose.onNodeWithText("Retry").performClick()
        assertEquals(1, retries)
    }

    @Test fun `at the largest font the band keeps its two rows and the labels are drawn`() {
        compose.setContent {
            CompositionLocalProvider(LocalDensity provides Density(LocalDensity.current.density, 2f)) { Host { Header() } }
        }
        val bounds = compose.onNodeWithTag("header").getUnclippedBoundsInRoot()
        assertEquals("two 44dp rows", 2 * MochilChrome.touchMin.value, bounds.height.value, 0.5f)
        // The labels are still drawn whole at 2x; the chip row scrolls sideways rather than clipping them.
        compose.onNodeWithText("All").assertIsDisplayed()
        compose.onNodeWithText("In Progress").assertExists()
        compose.onNodeWithText("Unread").assertExists()
    }

    @Test fun `the open library is brought into view when it is far along the row`() {
        val many = (1..14).map { LibrarySummary("lib$it", "Shelf number $it", "books", 1) }
        compose.setContent { Host { Header(selected = many.last(), libraries = many) } }
        compose.waitForIdle()
        compose.onNodeWithText("Shelf number 14").assertIsDisplayed()
    }

    @Test fun `golden Books Tide dark`() {
        compose.setContent { Host { Header(chips = StatusChips(BookStatusFilter.All) {}) } }
        Goldens.assertMatches(compose.onNodeWithTag("host"), "library-header-books-tide-dark")
    }

    @Test fun `golden Books in progress Ember light`() {
        compose.setContent { Host(AppPalette.Ember, AppThemeMode.Light) { Header(chips = StatusChips(BookStatusFilter.InProgress) {}) } }
        Goldens.assertMatches(compose.onNodeWithTag("host"), "library-header-books-ember-light")
    }

    @Test fun `golden a video shelf has pills and tools but no chips`() {
        compose.setContent { Host { Header(selected = anime, chips = null) } }
        Goldens.assertMatches(compose.onNodeWithTag("host"), "library-header-video-tide-dark")
    }

    @Test fun `golden Books at the largest font`() {
        compose.setContent {
            CompositionLocalProvider(LocalDensity provides Density(LocalDensity.current.density, 2f)) { Host { Header() } }
        }
        Goldens.assertMatches(compose.onNodeWithTag("host"), "library-header-books-tide-dark-font2")
    }

    @Test fun `golden a row of many libraries fades at the edge that has more`() {
        val many = listOf(books, novels, anime, music,
            LibrarySummary("podcasts", "Podcasts", "podcasts", 3), LibrarySummary("audiobooks", "Audiobooks", "audiobooks", 5),
            LibrarySummary("manga", "Manga", "manga", 7), LibrarySummary("documents", "Documents", "documents", 2))
        compose.setContent { Host { Header(libraries = many) } }
        Goldens.assertMatches(compose.onNodeWithTag("host"), "library-header-many-tide-dark")
    }

    @Test fun `golden offline`() {
        compose.setContent { Host { Header(chips = null, offline = "Offline · local downloads only") } }
        Goldens.assertMatches(compose.onNodeWithTag("host"), "library-header-offline-tide-dark")
    }
}
