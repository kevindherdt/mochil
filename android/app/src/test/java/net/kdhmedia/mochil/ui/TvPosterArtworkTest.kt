package net.kdhmedia.mochil.ui

import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class TvPosterArtworkTest {
    @Test
    fun aMatchedPosterIsTallButItsBackdropAndVideoFramesAreWide() {
        assertTrue(tvIsPosterArtwork("https://h/api/android/v1/video-metadata-artwork/movies/abc"))
        assertFalse(tvIsPosterArtwork("https://h/api/android/v1/video-metadata-artwork/movies/abc?backdrop=1"))
        assertFalse(tvIsPosterArtwork("https://h/api/android/v1/video-show-metadata-artwork/anime?backdrop=1&show=Baki"))
        assertFalse(tvIsPosterArtwork("https://h/api/android/v1/video-artwork/movies/abc?v=1"))
    }
}
