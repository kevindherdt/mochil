package net.kdhmedia.mochil.catalogue

import android.app.Application
import android.content.Context
import androidx.test.core.app.ApplicationProvider
import net.kdhmedia.mochil.audio.MusicLayout
import net.kdhmedia.mochil.audio.MusicViewModel
import net.kdhmedia.mochil.podcast.PodcastLayout
import net.kdhmedia.mochil.podcast.PodcastViewModel
import net.kdhmedia.mochil.podcast.PodcastViewModelCredentialStoreShadow
import net.kdhmedia.mochil.video.VideoBrowsePreferences
import net.kdhmedia.mochil.video.VideoLayout
import org.junit.Assert.assertEquals
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.Shadows.shadowOf
import org.robolectric.annotation.Config

/** Covers/List is ONE saved choice for books, music, podcasts and video. */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35], shadows = [PodcastViewModelCredentialStoreShadow::class])
class SharedBrowseLayoutTest {
    private val app: Application get() = ApplicationProvider.getApplicationContext()
    private fun prefs(name: String) = app.getSharedPreferences(name, Context.MODE_PRIVATE)

    @Test fun nothingSavedMeansCoversEverywhere() {
        assertEquals(PublicationLayout.Covers, LibraryNavigationStore(app).booksLayout())
        assertEquals(PublicationLayout.Covers, LibraryNavigationStore(app).seriesLayout())
        assertEquals(VideoLayout.Covers, VideoBrowsePreferences(app).layout("movies"))
        assertEquals(MusicLayout.Covers, MusicViewModel(app).state.value.layout)
        assertEquals(PodcastLayout.Covers, PodcastViewModel(app).state.value.layout)
    }

    @Test fun anOldSavedMusicListChoiceBecomesTheChoiceForEveryKind() {
        prefs("music_view").edit().putString("layout", "List").commit()
        assertEquals(PublicationLayout.List, LibraryNavigationStore(app).booksLayout())
        assertEquals(PublicationLayout.List, LibraryNavigationStore(app).seriesLayout())
        assertEquals(VideoLayout.List, VideoBrowsePreferences(app).layout("movies"))
        assertEquals(VideoLayout.List, VideoBrowsePreferences(app).layout("shows"))
        assertEquals(PodcastLayout.List, PodcastViewModel(app).state.value.layout)
        assertEquals(MusicLayout.List, MusicViewModel(app).state.value.layout)
    }

    @Test fun anOldSavedVideoListChoiceForOneKindWinsEverywhere() {
        prefs("video_browse").edit().putString("layout_shows", "List").commit()
        assertEquals(PublicationLayout.List, LibraryNavigationStore(app).seriesLayout())
        assertEquals(VideoLayout.List, VideoBrowsePreferences(app).layout("movies"))
    }

    @Test fun anOldSavedBooksOnlyListChoiceWinsEverywhere() {
        prefs("library_navigation").edit().putString("books_layout", "List").commit()
        assertEquals(PublicationLayout.List, LibraryNavigationStore(app).seriesLayout())
        assertEquals(PodcastLayout.List, PodcastViewModel(app).state.value.layout)
    }

    @Test fun aLaterChoiceIsNotOverriddenByStaleOldValues() {
        prefs("music_view").edit().putString("layout", "List").commit()
        LibraryNavigationStore(app).booksLayout() // first read migrates
        LibraryNavigationStore(app).setLayout(PublicationGrouping.Books, PublicationLayout.Covers)
        assertEquals(PublicationLayout.Covers, LibraryNavigationStore(app).seriesLayout())
        assertEquals(VideoLayout.Covers, VideoBrowsePreferences(app).layout("movies"))
        assertEquals(MusicLayout.Covers, MusicViewModel(app).state.value.layout)
    }

    @Test fun choosingInOneKindIsSeenByTheOthersImmediately() {
        val music = MusicViewModel(app)
        val podcasts = PodcastViewModel(app)
        podcasts.setLayout(PodcastLayout.List)
        shadowOf(android.os.Looper.getMainLooper()).idle()
        assertEquals(MusicLayout.List, music.state.value.layout)
        music.setLayout(MusicLayout.Covers)
        shadowOf(android.os.Looper.getMainLooper()).idle()
        assertEquals(PodcastLayout.Covers, podcasts.state.value.layout)
        assertEquals(VideoLayout.Covers, VideoBrowsePreferences(app).layout("shows"))
    }
}
