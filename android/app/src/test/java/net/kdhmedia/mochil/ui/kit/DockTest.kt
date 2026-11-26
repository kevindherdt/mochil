package net.kdhmedia.mochil.ui.kit

import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.width
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.test.assertHeightIsAtLeast
import androidx.compose.ui.test.assertHeightIsEqualTo
import androidx.compose.ui.test.assertIsNotSelected
import androidx.compose.ui.test.assertIsSelected
import androidx.compose.ui.test.assertWidthIsAtLeast
import androidx.compose.ui.semantics.SemanticsActions
import androidx.compose.ui.text.TextLayoutResult
import androidx.compose.ui.test.getUnclippedBoundsInRoot
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithTag
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.compose.ui.unit.Density
import androidx.compose.ui.unit.dp
import net.kdhmedia.mochil.testing.Goldens
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

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35], qualifiers = "w390dp-h844dp-xxhdpi")
@GraphicsMode(GraphicsMode.Mode.NATIVE)
class DockTest {
    @get:Rule val compose = createComposeRule()

    private val places = listOf(
        DockPlace("library", "Library", LucideIcons.withStroke("book-open", 1.8f)),
        DockPlace("discover", "Discover", LucideIcons.withStroke("compass", 1.8f)),
        DockPlace("ai", "AI", LucideIcons.withStroke("sparkles", 1.8f)),
        DockPlace("notes", "Notes", LucideIcons.withStroke("notebook-text", 1.8f)),
        DockPlace("settings", "Settings", LucideIcons.withStroke("settings", 1.8f)),
    )

    @Composable
    private fun Host(
        activeId: String?,
        palette: AppPalette = AppPalette.Tide,
        mode: AppThemeMode = AppThemeMode.Dark,
        fontScale: Float = 1f,
        onSelect: (DockPlace) -> Unit = {},
        list: List<DockPlace> = places,
    ) {
        val density = LocalDensity.current
        CompositionLocalProvider(LocalDensity provides Density(density.density, fontScale)) {
            MochilTheme(AppThemeSelection(palette, mode), glow = false) {
                Box(Modifier.width(390.dp).testTag("host")) { Dock(list, activeId, onSelect) }
            }
        }
    }

    @Test fun `shows every place in order`() {
        compose.setContent { Host("library") }
        val lefts = places.map { compose.onNodeWithText(it.label).getUnclippedBoundsInRoot().left }
        assertEquals(lefts.sorted(), lefts)
    }

    @Test fun `only the active place is selected`() {
        compose.setContent { Host("notes") }
        compose.onNodeWithText("Notes").assertIsSelected()
        listOf("Library", "Discover", "AI", "Settings").forEach { compose.onNodeWithText(it).assertIsNotSelected() }
    }

    @Test fun `no place is selected when the screen is not one of them`() {
        compose.setContent { Host(null) }
        places.forEach { compose.onNodeWithText(it.label).assertIsNotSelected() }
    }

    @Test fun `tapping a place reports it`() {
        var picked: String? = null
        compose.setContent { Host("library", onSelect = { picked = it.id }) }
        compose.onNodeWithText("Discover").performClick()
        assertEquals("discover", picked)
    }

    @Test fun `a four place dock still spans the width`() {
        compose.setContent { Host("library", list = places.filterNot { it.id == "ai" }) }
        val library = compose.onNodeWithText("Library").getUnclippedBoundsInRoot()
        val settings = compose.onNodeWithText("Settings").getUnclippedBoundsInRoot()
        assertTrue("Library starts at the left edge", library.left < 390.dp / 4)
        assertTrue("Settings ends at the right edge", settings.right > 390.dp * 3 / 4)
    }

    @Test fun `the dock is the token height and every place is a touch target`() {
        compose.setContent { Host("library") }
        compose.onNodeWithTag("host").assertHeightIsEqualTo(MochilChrome.dockHeightPhone)
        places.forEach {
            compose.onNodeWithText(it.label).assertHeightIsAtLeast(MochilChrome.touchMin)
            compose.onNodeWithText(it.label).assertWidthIsAtLeast(MochilChrome.touchMin)
        }
    }

    @Test fun `at the largest font scale the labels still fit inside the dock`() {
        compose.setContent { Host("library", fontScale = 2f) }
        val dock = compose.onNodeWithTag("host").getUnclippedBoundsInRoot()
        compose.onNodeWithTag("host").assertHeightIsAtLeast(MochilChrome.dockHeightPhone)
        places.forEach {
            val label = compose.onNodeWithText(it.label).getUnclippedBoundsInRoot()
            assertTrue("${it.label} stays inside the dock", label.top >= dock.top && label.bottom <= dock.bottom)
        }
    }

    @Test fun `at the largest font scale no label is cut short`() {
        compose.setContent { Host("library", fontScale = 2f) }
        places.forEach {
            val results = mutableListOf<TextLayoutResult>()
            val node = compose.onNodeWithText(it.label, useUnmergedTree = true).fetchSemanticsNode()
            node.config[SemanticsActions.GetTextLayoutResult].action?.invoke(results)
            val layout = results.single()
            assertTrue("${it.label} is not ellipsized", layout.lineCount == 1 && !layout.isLineEllipsized(0))
        }
    }

    @Test fun `golden Tide dark`() {
        compose.setContent { Host("discover", AppPalette.Tide, AppThemeMode.Dark) }
        Goldens.assertMatches(compose.onNodeWithTag("host"), "dock-tide-dark")
    }

    @Test fun `golden Ember light`() {
        compose.setContent { Host("library", AppPalette.Ember, AppThemeMode.Light) }
        Goldens.assertMatches(compose.onNodeWithTag("host"), "dock-ember-light")
    }

    @Test fun `golden E-ink light`() {
        compose.setContent { Host("settings", AppPalette.EInk, AppThemeMode.Light) }
        Goldens.assertMatches(compose.onNodeWithTag("host"), "dock-eink-light")
    }

    @Test fun `golden largest font scale`() {
        compose.setContent { Host("library", AppPalette.Tide, AppThemeMode.Dark, fontScale = 2f) }
        Goldens.assertMatches(compose.onNodeWithTag("host"), "dock-tide-dark-font2")
    }
}
