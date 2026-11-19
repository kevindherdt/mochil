package net.kdhmedia.mochil.update

import org.json.JSONObject
import org.junit.Assert.assertEquals
import org.junit.Test

class ReleaseNotesTest {
    @Test fun `release notes read the current fixes field`() {
        assertEquals(listOf("Focus is visible."), JSONObject("""{"fixes":["Focus is visible."]}""").releaseFixes())
    }

    @Test fun `release notes retain compatibility with the old fixed field`() {
        assertEquals(listOf("Legacy fix."), JSONObject("""{"fixed":["Legacy fix."]}""").releaseFixes())
    }

    @Test fun `current fixes take precedence when both fields exist`() {
        assertEquals(listOf("Current."), JSONObject("""{"fixes":["Current."],"fixed":["Old."]}""").releaseFixes())
    }
}
