package net.kdhmedia.mochil.ui

import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.width
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.input.key.Key
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.assertIsFocused
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performKeyInput
import androidx.compose.ui.test.pressKey
import androidx.compose.ui.test.performSemanticsAction
import androidx.compose.ui.semantics.SemanticsActions
import androidx.compose.ui.unit.dp
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35])
class TvSettingsCompactBehaviorTest {
    @get:Rule val compose = createComposeRule()

    @Test fun allCategoriesFitShortViewportAndRemainRemoteNavigable() {
        var selected by mutableStateOf(TvSettingsCategory.Appearance)
        compose.setContent {
            MaterialTheme {
                Box(Modifier.width(220.dp).height(350.dp)) {
                    TvSettingsCategoryList(selected, updatesEnabled = true) { selected = it }
                }
            }
        }

        compose.onNodeWithText("Diagnostics").assertIsDisplayed()
        compose.onNodeWithText("Appearance").performSemanticsAction(SemanticsActions.RequestFocus)
        compose.onNodeWithText("Appearance").assertIsFocused().performKeyInput { pressKey(Key.DirectionDown) }
        compose.onNodeWithText("Libraries").assertIsFocused()
        compose.runOnIdle { check(selected == TvSettingsCategory.Libraries) }
    }
}
