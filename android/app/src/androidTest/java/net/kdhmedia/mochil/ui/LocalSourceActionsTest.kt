package net.kdhmedia.mochil.ui

import androidx.compose.material3.MaterialTheme
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import org.junit.Assert.assertEquals
import org.junit.Rule
import org.junit.Test

class LocalSourceActionsTest {
    @get:Rule val compose = createComposeRule()

    @Test fun menuStartsClosedAndEachSourceActionRemainsAvailable() {
        val invoked = mutableListOf<String>()
        compose.setContent {
            MaterialTheme {
                LocalSourceActions(
                    onServer = { invoked.add("server") },
                    onFolder = { invoked.add("folder") },
                    onOpds = { invoked.add("opds") },
                    onWebDav = { invoked.add("webdav") },
                )
            }
        }
        val actions = listOf(
            "Connect Mochil server" to "server",
            "Add local media folder" to "folder",
            "OPDS catalogues" to "opds",
            "WebDAV libraries" to "webdav",
        )
        actions.forEach { (label, expected) ->
            compose.onNodeWithText(label).assertDoesNotExist()
            compose.onNodeWithText("Manage sources").assertIsDisplayed().performClick()
            compose.onNodeWithText(label).assertIsDisplayed().performClick()
            compose.runOnIdle { assertEquals(expected, invoked.last()) }
            compose.onNodeWithText(label).assertDoesNotExist()
        }
    }
}
