package net.kdhmedia.mochil.audio

import java.nio.file.Files
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Rule
import org.junit.Test
import org.junit.rules.TemporaryFolder

class OfflineAudioLocalFileTest {
    @get:Rule val temporary = TemporaryFolder()

    @Test fun `only direct local files are eligible for playback`() {
        val root = temporary.newFolder("music")
        val saved = root.resolve("song.flac").apply { createNewFile() }
        assertEquals(saved, offlineAudioLocalFile(root, saved.path))
        assertNull(offlineAudioLocalFile(root, root.resolve("missing.flac").path))
        assertNull(offlineAudioLocalFile(root, root.path, includeMissing = true))
        val child = root.resolve("folder").apply { mkdir() }
        assertNull(offlineAudioLocalFile(root, child.path, includeMissing = true))
    }

    @Test fun `removal can tolerate missing files but not external paths`() {
        val root = temporary.newFolder("music")
        val missing = root.resolve("missing.flac")
        assertEquals(missing, offlineAudioLocalFile(root, missing.path, includeMissing = true))
        val outside = temporary.newFile("private.txt")
        assertNull(offlineAudioLocalFile(root, outside.path, includeMissing = true))
        assertNull(offlineAudioLocalFile(root, root.resolve("../private.txt").path, includeMissing = true))
        assertNull(offlineAudioLocalFile(root, root.resolve("../absent.txt").path, includeMissing = true))
    }

    @Test fun `symlink outside download directory is rejected`() {
        val root = temporary.newFolder("music")
        val outside = temporary.newFile("private.txt")
        val link = root.resolve("song.flac")
        Files.createSymbolicLink(link.toPath(), outside.toPath())
        assertNull(offlineAudioLocalFile(root, link.path))
        assertNull(offlineAudioLocalFile(root, link.path, includeMissing = true))
    }
}
