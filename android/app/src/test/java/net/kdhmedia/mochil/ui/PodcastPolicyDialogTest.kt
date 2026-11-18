package net.kdhmedia.mochil.ui

import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.compose.ui.test.performScrollTo
import net.kdhmedia.mochil.podcast.PodcastDevicePolicy
import net.kdhmedia.mochil.podcast.PodcastPolicy
import net.kdhmedia.mochil.podcast.PodcastShow
import org.junit.Assert.assertEquals
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35])
class PodcastPolicyDialogTest {
    @get:Rule val compose = createComposeRule()

    private val show = PodcastShow(id = "feed", title = "Show", description = "", episodeCount = 1, policy = PodcastPolicy(autoDownload = true, retainDownloads = 5, deletePlayedAfterDays = 7))

    private fun dialog(device: PodcastDevicePolicy, saved: MutableList<Pair<PodcastPolicy, PodcastDevicePolicy>> = mutableListOf()) =
        compose.setContent { PodcastPolicyDialog(show, device, busy = false, dismiss = {}, save = { s, d -> saved += s to d }) }

    @Test fun deviceSectionStatesInOneSentenceWhatThisPhoneKeeps() {
        dialog(PodcastDevicePolicy(autoDownload = true, retainDownloads = 3, unmeteredOnly = true))
        compose.onNodeWithText("On this device").assertIsDisplayed()
        compose.onNodeWithText("On the server").assertExists()
        compose.onNodeWithText("Keeps the newest 3 episodes on this device, Wi-Fi only.").assertIsDisplayed()
    }

    @Test fun summaryFollowsTheStoredDeviceFields() {
        dialog(PodcastDevicePolicy(autoDownload = true, retainDownloads = 10, unmeteredOnly = false))
        compose.onNodeWithText("Keeps the newest 10 episodes on this device, on any network.").assertIsDisplayed()
    }

    @Test fun manualOnlyDeviceSaysSoAndNothingContradictsIt() {
        dialog(PodcastDevicePolicy(autoDownload = false))
        compose.onNodeWithText("Nothing downloads to this device by itself. You can still download episodes by hand.").assertIsDisplayed()
        compose.onNodeWithText("These settings manage the server's podcast cache. Device downloads remain under your manual control.").assertDoesNotExist()
    }

    @Test fun savingKeepsEveryStoredFieldAndWifiOnlyStillToggles() {
        val saved = mutableListOf<Pair<PodcastPolicy, PodcastDevicePolicy>>()
        dialog(PodcastDevicePolicy(autoDownload = true, retainDownloads = 3, unmeteredOnly = true), saved)
        compose.onNodeWithText("Wi-Fi only").performScrollTo().performClick()
        compose.onNodeWithText("Save").performClick()
        assertEquals(show.policy, saved.single().first)
        assertEquals(PodcastDevicePolicy(autoDownload = true, retainDownloads = 3, unmeteredOnly = false), saved.single().second)
    }
}
