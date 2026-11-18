package net.kdhmedia.mochil.ui

import net.kdhmedia.mochil.video.VideoItem
import net.kdhmedia.mochil.video.VideoState
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * A movie's detail has to survive a rotation: the activity declares no configChanges, so the composition that held
 * the selected item is thrown away and only a saveable key comes back. These tests cover the derivation that key
 * drives — the item itself is the ViewModel's copy, looked up again after recreation.
 */
class VideoMovieSelectionTest {
    private fun movie(id: String, libraryId: String = "movies", resume: Double = 0.0) = VideoItem(
        id = id, libraryId = libraryId, title = id, show = "", season = 0, episode = 0,
        durationSeconds = 7_200.0, artworkPath = "", streamPath = "s", hlsPath = "", resumeSeconds = resume,
        subtitles = emptyList(),
    )

    @Test fun `the saved key resolves the movie again after recreation`() {
        val item = movie("heat")
        val key = videoMovieSelectionKey(item)
        val recreated = VideoState(items = listOf(item))
        assertEquals("heat", videoSelectedMovie(recreated.items, key)?.id)
    }

    @Test fun `the recreated detail draws the shelf copy, not the one captured before rotation`() {
        val key = videoMovieSelectionKey(movie("heat", resume = 0.0))
        val refreshed = VideoState(items = listOf(movie("heat", resume = 900.0)))
        assertEquals(900.0, videoSelectedMovie(refreshed.items, key)?.resumeSeconds ?: 0.0, 0.0)
    }

    @Test fun `the key is the catalogue identity, so two libraries can hold the same id`() {
        val films = videoMovieSelectionKey(movie("heat", libraryId = "movies"))
        val documentaries = videoMovieSelectionKey(movie("heat", libraryId = "docs"))
        assertNotEquals(films, documentaries)
        assertNull("a key from another library must not open this shelf's movie", videoSelectedMovie(listOf(movie("heat", libraryId = "movies")), documentaries))
    }

    @Test fun `no selection and a movie that left the shelf both open nothing`() {
        assertNull(videoSelectedMovie(listOf(movie("heat")), null))
        assertNull(videoSelectedMovie(listOf(movie("heat")), videoMovieSelectionKey(movie("gone"))))
        assertNull(videoSelectedMovie(emptyList(), videoMovieSelectionKey(movie("heat"))))
    }

    /**
     * The item cannot be saved — it is a data class of the shelf — so only its key is. A plain `remember` there is the
     * bug: the composition is discarded on rotation and the detail came back to the grid.
     */
    @Test fun `only the selection key is saved across recreation`() {
        val source = java.nio.file.Files.readString(java.nio.file.Paths.get("src/main/java/net/kdhmedia/mochil/ui/VideoScreen.kt"))
        assertTrue(source.contains("var selectedMovieKey by rememberSaveable"))
        assertFalse(source.contains("var selectedMovie by remember("))
    }
}
