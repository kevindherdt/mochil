package net.kdhmedia.mochil.ui

import androidx.compose.material3.MaterialTheme
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.onAllNodesWithText
import androidx.compose.ui.test.performClick
import net.kdhmedia.mochil.playback.PlaybackQuality
import net.kdhmedia.mochil.playback.PlaybackQualityPreferences
import org.junit.Assert.assertEquals
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.RuntimeEnvironment
import org.robolectric.annotation.Config

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35])
class PlaybackQualitySettingsBehaviorTest {
    @get:Rule val compose = createComposeRule()

    @Test fun mobileRetainsItsExistingLabels() {
        compose.setContent { MaterialTheme { PlaybackQualitySettings() } }
        compose.onNodeWithText("Wi-Fi").assertIsDisplayed()
        compose.onNodeWithText("Mobile data").assertIsDisplayed()
    }

    @Test fun tvLabelsMeteredPolicyAndSavesBothExistingPreferences() {
        val context = RuntimeEnvironment.getApplication()
        context.getSharedPreferences("playback_quality", 0).edit().clear().commit()
        compose.setContent { MaterialTheme { PlaybackQualitySettings(television = true) } }
        compose.onNodeWithText("Unmetered (usually Wi-Fi or Ethernet)").assertIsDisplayed()
        compose.onNodeWithText("Metered (including cellular)").assertIsDisplayed()
        compose.onAllNodesWithText("Low")[0].performClick()
        compose.onAllNodesWithText("Medium")[1].performClick()
        compose.runOnIdle {
            val saved = PlaybackQualityPreferences(context).load()
            assertEquals(PlaybackQuality.Low, saved.wifi)
            assertEquals(PlaybackQuality.Medium, saved.mobile)
        }
    }
}
