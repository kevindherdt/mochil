package net.kdhmedia.mochil.ui

import java.nio.file.Files
import java.nio.file.Paths
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class DashboardLongPressActionsSourceTest {
    private val source = Files.readString(Paths.get("src/main/java/net/kdhmedia/mochil/ui/DashboardScreen.kt"))

    @Test fun `home removal stays behind the established long press menu`() {
        assertTrue(source.contains("onLongClick = { menuExpanded = true }"))
        assertTrue(source.contains("DropdownMenuItem(text = { Text(\"Remove from Home\") }"))
        assertFalse(source.contains("Button(onClick = onDismiss"))
        assertFalse(source.contains("TextButton(onClick = onDismiss"))
        assertFalse(source.contains("IconButton(onClick = onDismiss"))
    }
}
