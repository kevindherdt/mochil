package net.kdhmedia.mochil.ui

import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.test.assertIsFocused
import androidx.compose.ui.test.getUnclippedBoundsInRoot
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performKeyInput
import androidx.compose.ui.test.performSemanticsAction
import androidx.compose.ui.test.pressKey
import androidx.compose.ui.input.key.Key
import androidx.compose.ui.semantics.SemanticsActions
import androidx.compose.ui.unit.Density
import androidx.compose.ui.unit.dp
import net.kdhmedia.mochil.ui.theme.AppPalette
import net.kdhmedia.mochil.ui.theme.AppThemeMode
import net.kdhmedia.mochil.ui.theme.AppThemeSelection
import net.kdhmedia.mochil.ui.theme.MochilTheme
import net.kdhmedia.mochil.video.VideoCastMember
import net.kdhmedia.mochil.video.VideoItem
import net.kdhmedia.mochil.video.VideoShow
import net.kdhmedia.mochil.video.VideoState
import net.kdhmedia.mochil.video.VideoTitleDetails
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config

/**
 * Reported from the set twice: with an episode focused, the show title was cut by the top bar because
 * the page had to scroll to reveal the rail. The header now measures what the window can afford, so
 * the title and the focused card both stay inside the band at any system font scale.
 */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35], qualifiers = "w800dp-h450dp")
class TvShowHeaderFitTest {
    @get:Rule val compose = createComposeRule()

    private fun check(font: Float) {
        val show = VideoShow("Extracurricular", "Extracurricular", "show", 2020, listOf(1), "", overview = "Story ".repeat(30))
        val episodes = listOf(
            VideoItem("ep4", "tv", "Episode 4", "Extracurricular", 1, 4, 2640.0, "", "s", "", 0.0, emptyList(), overview = "An incident at school puts Gyuri in a tense exchange with her mother. ".repeat(2)),
            VideoItem("ep5", "tv", "Episode 5", "Extracurricular", 1, 5, 2820.0, "", "s", "", 0.0, emptyList(), overview = "Brief."),
        )
        val state = VideoState(libraryId = "tv", libraryKind = "tv", items = episodes, details = mapOf("show:Extracurricular" to VideoTitleDetails(cast = listOf(VideoCastMember("Actor One", "Hero")))))
        compose.setContent {
            CompositionLocalProvider(LocalDensity provides Density(1f, font)) {
                MochilTheme(selection = AppThemeSelection(AppPalette.Espresso, AppThemeMode.Dark), glow = false, cinema = true) {
                    Box(Modifier.fillMaxSize()) { TvShowDetail(show, state, null, { it }, {}, {}, { _, _ -> }, {}, {}) }
                }
            }
        }
        compose.onNodeWithText("Play").performSemanticsAction(SemanticsActions.RequestFocus)
        compose.onNodeWithText("Play").performKeyInput { pressKey(Key.DirectionDown) }
        compose.onNodeWithText("Episode 4").assertIsFocused()
        val title = compose.onNodeWithText("Extracurricular").getUnclippedBoundsInRoot()
        assertTrue("hero title cut at font scale $font: $title", title.top >= 64.dp && title.bottom <= 430.dp)
        val card = compose.onNodeWithText("Episode 4").getUnclippedBoundsInRoot()
        assertTrue("focused episode not fully visible at $font: $card", card.top >= 64.dp && card.bottom <= 430.dp)
        val cast = compose.onNodeWithText("Cast").getUnclippedBoundsInRoot()
        assertTrue("cast row pokes into the viewport at $font: $cast", cast.top >= 430.dp)
    }

    @Test fun normalType() = check(1f)
    @Test fun raisedType() = check(1.3f)
    @Test fun largeType() = check(1.5f)
    @Test @Config(sdk = [35], qualifiers = "w960dp-h540dp") fun tallerWindow() = check(1f)

    /** The movie page had the same dead inset and a title that scrolled away with it. */
    @Test fun movieKeepsTitleAndControls() {
        val movie = VideoItem("m", "movies", "The Witch: Part 1. The Subversion", "", 2018, 0, 7560.0, "", "s", "", 0.35, emptyList(), overview = "Ja-yoon is a high school student who struggles with memory loss after she endured some unknown trauma during her childhood. ".repeat(2))
        val details = VideoTitleDetails(cast = List(4) { VideoCastMember("Actor $it", "Role $it") })
        compose.setContent {
            MochilTheme(selection = AppThemeSelection(AppPalette.Espresso, AppThemeMode.Dark), glow = false, cinema = true) {
                Box(Modifier.fillMaxSize()) { TvMovieDetail(movie, null, "poster", "backdrop", {}, {}, details = details) }
            }
        }
        compose.onNodeWithText("Resume").performSemanticsAction(SemanticsActions.RequestFocus)
        val title = compose.onNodeWithText("The Witch", substring = true).getUnclippedBoundsInRoot()
        assertTrue("movie title cut: $title", title.top >= 64.dp && title.bottom <= 430.dp)
        val watched = compose.onNodeWithText("Mark watched").getUnclippedBoundsInRoot()
        assertTrue("movie controls pushed under the edge: $watched", watched.bottom <= 430.dp)
    }
}
