package net.kdhmedia.mochil.audio

import java.nio.file.Files
import java.nio.file.Paths
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class MusicQueueLazyLoadingSourceTest {
    @Test fun `persistent queue is not downloaded during app startup`() {
        val source = Files.readString(Paths.get("src/main/java/net/kdhmedia/mochil/audio/MusicQueueViewModel.kt"))
        assertFalse(source.contains("init { refresh() }"))
        assertTrue(source.contains("fun ensureLoaded()"))
        assertTrue(source.contains("state.value.loaded || state.value.loading"))
    }
}
