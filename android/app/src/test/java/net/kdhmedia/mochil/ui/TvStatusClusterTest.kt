package net.kdhmedia.mochil.ui

import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.test.hasStateDescription
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithText
import net.kdhmedia.mochil.ui.theme.AppPalette
import net.kdhmedia.mochil.ui.theme.AppThemeMode
import net.kdhmedia.mochil.ui.theme.AppThemeSelection
import net.kdhmedia.mochil.ui.theme.MochilTheme
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import java.time.LocalTime
import java.time.format.DateTimeFormatter

/**
 * The corner cluster shows a real clock and nothing else. The sheets draw a "Connected" pill beside it, but no reachability
 * signal reaches this corner, and a pill that says Connected while the server is down is worse than an empty corner.
 */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35], qualifiers = "w960dp-h540dp-xhdpi")
class TvStatusClusterTest {
    @get:Rule val compose = createComposeRule()

    @Composable
    private fun Host() = MochilTheme(AppThemeSelection(AppPalette.Tide, AppThemeMode.Dark), glow = false, cinema = true) { TvStatusCluster() }

    @Test
    fun itPrintsTheTimeAndClaimsNothingElse() {
        compose.setContent { Host() }
        val expected = LocalTime.now().format(DateTimeFormatter.ofPattern("HH:mm"))
        compose.onNodeWithText(expected).assertExists()
        // No connection claim: the app cannot know one here, and the whole point of removing the pill was that it asserted
        // a state nothing measured.
        compose.onNodeWithText("Connected").assertDoesNotExist()
        compose.onNode(hasStateDescription("Server connected")).assertDoesNotExist()
    }
}
