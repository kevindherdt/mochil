package net.kdhmedia.mochil.ui

import androidx.compose.material3.MaterialTheme
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithText
import net.kdhmedia.mochil.video.*
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35], qualifiers = "w400dp-h800dp")
class MobileVideoCastTest {
    @get:Rule val compose = createComposeRule()
    @Test fun selectedShowDisplaysCast() {
        val show = VideoShow("Show", "Show", "id", 0, emptyList(), "")
        val details = VideoTitleDetails(cast = listOf(VideoCastMember("Actor One", "Hero")))
        compose.setContent { MaterialTheme {
            VideoScreen(VideoState(selectedShow = show, details = mapOf("show:Show" to details)), { it }, null,
                {}, {}, {}, {}, {}, {}, {}, {}, { _, _ -> }, {}, {}, {})
        } }
        compose.onNodeWithText("Actor One").assertIsDisplayed()
        compose.onNodeWithText("Hero").assertIsDisplayed()
    }
}
