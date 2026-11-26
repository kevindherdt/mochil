package net.kdhmedia.mochil.ui

import androidx.compose.foundation.layout.width
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithTag
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.unit.dp
import net.kdhmedia.mochil.testing.Goldens
import net.kdhmedia.mochil.ui.theme.AppPalette
import net.kdhmedia.mochil.ui.theme.AppThemeMode
import net.kdhmedia.mochil.ui.theme.AppThemeSelection
import net.kdhmedia.mochil.ui.theme.MochilTheme
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import org.robolectric.annotation.GraphicsMode

/** A Discover rail is a heading and a sideways row of the shelf's own cards; an empty rail is not drawn at all. */
@RunWith(RobolectricTestRunner::class)
@GraphicsMode(GraphicsMode.Mode.NATIVE)
@Config(sdk = [35], qualifiers = "w390dp-h844dp-xxhdpi")
class DashboardRailTest {
    @get:Rule val compose = createComposeRule()

    @Composable
    private fun Host(content: @Composable () -> Unit) {
        MochilTheme(AppThemeSelection(AppPalette.Tide, AppThemeMode.Dark), glow = false) {
            Surface(Modifier.width(390.dp).testTag("host"), color = MaterialTheme.colorScheme.background) { content() }
        }
    }

    @Test fun `an empty rail draws nothing at all`() {
        compose.setContent { Host { DashboardRail("Continue reading", emptyList<String>(), { it to it }) { Text(it) } } }
        compose.onNodeWithText("Continue reading").assertDoesNotExist()
    }

    @Test fun `a rail draws its heading and every card`() {
        compose.setContent { Host { DashboardRail("Continue reading", listOf("one", "two", "three"), { it to it }) { Text("card $it") } } }
        compose.onNodeWithText("Continue reading").assertIsDisplayed()
        compose.onNodeWithText("card one").assertIsDisplayed()
        compose.onNodeWithText("card two").assertIsDisplayed()
    }

    @Test fun `golden a rail of two covers`() {
        compose.setContent {
            Host { DashboardRail("Continue reading", listOf("one", "two"), { it to it }) { Text("card $it") } }
        }
        Goldens.assertMatches(compose.onNodeWithTag("host"), "discover-rail-tide-dark")
    }
}
