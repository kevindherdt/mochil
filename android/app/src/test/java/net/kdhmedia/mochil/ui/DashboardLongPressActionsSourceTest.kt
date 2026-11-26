package net.kdhmedia.mochil.ui

import java.nio.file.Files
import java.nio.file.Paths
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class DashboardLongPressActionsSourceTest {
    private val source = Files.readString(Paths.get("src/main/java/net/kdhmedia/mochil/ui/DashboardScreen.kt"))

    @Test fun `home removal stays behind a long press, never a visible button`() {
        // Discover's rails are cards now: the gesture is still a long press, labelled "Remove … from Discover".
        assertTrue(source.contains("from Discover"))
        assertTrue(source.contains("onLongClickLabel"))
        assertTrue(source.contains("onDismissReading") && source.contains("onDismissWatching") && source.contains("onDismissListening"))
        assertFalse(source.contains("Button(onClick = onDismiss"))
        assertFalse(source.contains("TextButton(onClick = onDismiss"))
        assertFalse(source.contains("IconButton(onClick = onDismiss"))
    }
}
