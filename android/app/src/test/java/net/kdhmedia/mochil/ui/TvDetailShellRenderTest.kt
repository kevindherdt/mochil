package net.kdhmedia.mochil.ui

import androidx.compose.foundation.layout.*
import androidx.compose.material3.Surface
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.asAndroidBitmap
import androidx.compose.ui.input.key.Key
import androidx.compose.ui.test.*
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.unit.dp
import net.kdhmedia.mochil.audio.MusicState
import net.kdhmedia.mochil.catalogue.LibraryState
import net.kdhmedia.mochil.catalogue.LibrarySummary
import net.kdhmedia.mochil.podcast.PodcastState
import net.kdhmedia.mochil.ui.theme.MochilTheme
import net.kdhmedia.mochil.video.*
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import org.robolectric.annotation.GraphicsMode
import java.io.File

/** Native Skia/fonts: legacy Robolectric fonts cannot exercise real wrapping. */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35], qualifiers = "w960dp-h540dp-mdpi")
@GraphicsMode(GraphicsMode.Mode.NATIVE)
class TvDetailShellRenderTest {
    @get:Rule val compose = createComposeRule()
    private val title = "Baki Hanma: The World's Strongest\nThe Arizona State Prison Saga"
    private val episode = VideoItem("ep", "tv", "Episode One", "Baki", 1, 1, 1440.0, "", "stream", "", 0.0, emptyList(), overview = "To gain the skills he needs to surpass his powerful father, Baki enters Arizona State Prison to take on the notorious inmate known as Mr. Unchained.\nA second line of story.\nA third line.")
    private val details = VideoTitleDetails(genres = listOf("Animation", "Action & Adventure", "Science Fiction", "Drama"), runtimeMinutes = 24, cast = listOf(VideoCastMember("Actor One", "Hero")))

    @Composable private fun Shell(fontScale: Float = 1.6f, dock: Boolean = true, content: @Composable () -> Unit) {
        MochilTheme(selection = net.kdhmedia.mochil.ui.theme.AppThemeSelection(), cinema = true, glow = false) {
            val input = androidx.compose.ui.platform.LocalInputModeManager.current
            androidx.compose.runtime.LaunchedEffect(Unit) { input.requestInputMode(androidx.compose.ui.input.InputMode.Keyboard) }
            Surface {
            TvContentFrame(showRail = true, playbackDockVisible = dock) {
                androidx.compose.runtime.CompositionLocalProvider(androidx.compose.ui.platform.LocalDensity provides androidx.compose.ui.unit.Density(1f, fontScale)) { content() }
            }
        } }
    }
    @Composable private fun Library(video: VideoState) {
        val lib = LibrarySummary(video.libraryId, "Videos", video.libraryKind, 1)
        TvLibraryScreen(LibraryState(selected = lib, libraries = listOf(lib), initialLoading = false), MusicState(), video, PodcastState(), null,
            TvLibraryArtwork({ "" }, { "" }, { "" }, { "" }, { "" }, { path -> if (path.isBlank()) "" else "https://example.invalid/$path.jpg" }, { "" }, { "" }),
            TvLibraryActions(TvNavigationActions({}, {}, {}, {})), "", TvGridPositionStore())
    }
    private fun shot(name: String) {
        val dir = File("build/reports/tv-layout").apply { mkdirs() }
        File(dir, "$name.png").outputStream().use { compose.onRoot().captureToImage().asAndroidBitmap().compress(android.graphics.Bitmap.CompressFormat.PNG, 100, it) }
    }
    @Test fun homeShowsListeningArtworkBelowWatchingWithoutOversizedHero() = checkHome(1f, "standard", true)
    @Test @Config(qualifiers = "w800dp-h450dp-mdpi") fun homeCompactLargeTypeNavigation() = checkHome(1.6f, "compact-large", false)
    @Test @Config(qualifiers = "w1280dp-h720dp-mdpi") fun homeTallWindow() = checkHome(1f, "tall", true)
    private fun checkHome(font: Float, name: String, requireInitialArtwork: Boolean) {
        val listening = net.kdhmedia.mochil.dashboard.ContinueListening(net.kdhmedia.mochil.audio.AudioTrack("a", "music", "Listening fixture", "Artist", "Album", 3600.0), 60.0, 3600.0)
        val watching = episode.copy(title = "005 - The Cotton Drifting Chapter Part 1 Jealousy and Another Extremely Long Episode Title", resumeSeconds = 120.0)
        compose.setContent { Shell(fontScale = font, dock = false) {
            TvHomeScreen(net.kdhmedia.mochil.dashboard.DashboardState(watching = listOf(watching), listening = listOf(listening), loading = false), { "https://example.invalid/listening.jpg" }, { "https://example.invalid/watching.jpg" }, authorizationHeader = null, onListen = {}, onWatch = {}, onDismissListening = {}, onDismissWatching = {}, onRetry = {}, focusRail = {})
        } }
        compose.waitForIdle()
        compose.onNode(hasContentDescription("Artwork for ${watching.displayTitle}") and hasClickAction()).performSemanticsAction(androidx.compose.ui.semantics.SemanticsActions.RequestFocus) { it() }
        shot("home-$name-initial")
        val art = compose.onNodeWithContentDescription("Artwork for Listening fixture", useUnmergedTree = true).getUnclippedBoundsInRoot()
        if (requireInitialArtwork) assertTrue("Listening has only a stranded heading: $art", art.top <= (org.robolectric.RuntimeEnvironment.getApplication().resources.configuration.screenHeightDp - 88).dp)
        compose.onNode(isFocused()).performKeyInput { pressKey(Key.DirectionDown) }
        compose.onNode(hasContentDescription("Artwork for Listening fixture") and hasClickAction()).assertIsFocused()
        shot("home-$name-listening")
        val viewport = compose.onNode(SemanticsMatcher.keyIsDefined(androidx.compose.ui.semantics.SemanticsProperties.VerticalScrollAxisRange)).getUnclippedBoundsInRoot()
        val portrait = compose.onNodeWithContentDescription("Artwork for Listening fixture", useUnmergedTree = true).getUnclippedBoundsInRoot()
        assertTrue("Focused listening tile clipped by Home viewport: $portrait / $viewport", portrait.top >= viewport.top && portrait.bottom <= viewport.bottom)
    }

    @Test @Config(qualifiers = "w800dp-h450dp-mdpi") fun movieTitleAndCompactControlsSurviveFocus() = checkMovie(1.6f, true, "compact-large")
    @Test fun movieStandardWindow() = checkMovie(1f, false, "standard")
    @Test @Config(qualifiers = "w1280dp-h720dp-mdpi") fun movieTallLargeType() = checkMovie(1.6f, false, "tall-large")
    private fun checkMovie(font: Float, dock: Boolean, name: String) {
        val movie = episode.copy(id = "movie", libraryId = "movies", title = title, show = "", season = 0, episode = 0, resumeSeconds = 120.0, artworkPath = "poster")
        compose.setContent { Shell(font, dock) { Library(VideoState(libraryId = "movies", libraryKind = "movies", items = listOf(movie), details = mapOf("item:movie" to details))) } }
        compose.waitForIdle()
        compose.mainClock.advanceTimeBy(1000)
        compose.onNode(hasText(title) and hasClickAction()).performSemanticsAction(androidx.compose.ui.semantics.SemanticsActions.RequestFocus) { it() }
        compose.onNode(hasText(title) and hasClickAction()).performKeyInput { pressKey(Key.Enter) }
        compose.onNodeWithText("Resume").performSemanticsAction(androidx.compose.ui.semantics.SemanticsActions.RequestFocus) { it() }
        shot("movie-$name-focused")
        val full = compose.onNodeWithText(title).getUnclippedBoundsInRoot()
        assertTrue("Movie title clipped on focus: $full", full.top >= 64.dp)
        if (name != "compact-large") {
            val poster = compose.onNodeWithContentDescription("Artwork for $title").getUnclippedBoundsInRoot()
            assertTrue("Poster clipped on focus: $poster", poster.top >= 64.dp)
        }
        compose.onNodeWithText("Resume").performKeyInput { pressKey(Key.DirectionDown) }
        compose.onNodeWithText("Actor One").assertIsFocused().performKeyInput { pressKey(Key.DirectionUp) }
        compose.onNodeWithText("Resume").assertIsFocused()
        shot("movie-$name-return")
        val returnedTitle = compose.onNodeWithText(title).getUnclippedBoundsInRoot()
        assertTrue("Movie title clipped after cast return: $returnedTitle", returnedTitle.top >= 64.dp)
        val target = compose.onNodeWithText("Resume").getUnclippedBoundsInRoot()
        val textLayout = mutableListOf<androidx.compose.ui.text.TextLayoutResult>()
        compose.onNodeWithText("Resume", useUnmergedTree = true).performSemanticsAction(androidx.compose.ui.semantics.SemanticsActions.GetTextLayoutResult) { it(textLayout) }
        assertTrue("Oversized movie action typography: ${textLayout.single().layoutInput.style.fontSize}", textLayout.single().layoutInput.style.fontSize.value <= 16f)
        compose.onNodeWithText("Mark watched").assert(SemanticsMatcher.expectValue(androidx.compose.ui.semantics.SemanticsProperties.Role, androidx.compose.ui.semantics.Role.Button))
        val visual = compose.onNodeWithTag("compact-Resume", useUnmergedTree = true).getUnclippedBoundsInRoot()
        assertTrue("Keep 48dp action target: $target", target.bottom - target.top >= 48.dp)
        assertTrue("Shrink visual not target: $visual / $target", visual.bottom - visual.top < target.bottom - target.top)
    }

    @Test @Config(qualifiers = "w800dp-h450dp-mdpi") fun showTitleRemainsFullyVisibleAfterEpisodeFocus() = checkShow(1.6f, true, "compact-large-dock")
    @Test @Config(qualifiers = "w800dp-h450dp-mdpi") fun showCompactNormalType() = checkShow(1f, false, "compact")
    @Test fun showStandardWindow() = checkShow(1f, false, "standard")
    @Test @Config(qualifiers = "w1280dp-h720dp-mdpi") fun showTallLargeType() = checkShow(1.6f, false, "tall-large")
    @Test fun shortShowTitleDoesNotStrandCastHeading() = checkShow(1f, false, "short-title", "Baki Hanma")
    private fun checkShow(font: Float, dock: Boolean, name: String, heroTitle: String = title) {
        val show = VideoShow(heroTitle, "Baki", "show", 1, listOf(1, 2), "", year = 2021, overview = episode.overview)
        compose.setContent { Shell(font, dock) { Library(VideoState(libraryId = "tv", libraryKind = "tv", selectedShow = show, selectedSeason = 1, items = listOf(episode), details = mapOf("show:Baki" to details))) } }
        compose.waitForIdle()
        compose.mainClock.advanceTimeBy(1000)
        compose.onNodeWithText("Play").performSemanticsAction(androidx.compose.ui.semantics.SemanticsActions.RequestFocus) { it() }
        shot("show-$name-initial")
        compose.onNodeWithText("Play").assertIsFocused()
        compose.onNodeWithText("Play").performKeyInput { pressKey(Key.DirectionDown) }
        shot("show-$name-episode")
        compose.onNodeWithText("Episode One").assertIsFocused()
        val visible = compose.onNodeWithText(heroTitle).fetchSemanticsNode().boundsInRoot
        val full = compose.onNodeWithText(heroTitle).getUnclippedBoundsInRoot()
        assertTrue("Title clipped by detail viewport: full=$full visible=$visible", full.top >= 64.dp && visible.height >= (full.bottom - full.top).value - 1f)
        val safeBottom = (org.robolectric.RuntimeEnvironment.getApplication().resources.configuration.screenHeightDp - (if (dock) 88 else 24) - 20).dp
        val story = compose.onNodeWithText(episode.overview).getUnclippedBoundsInRoot()
        assertTrue("Story clipped on episode focus: $story", story.top >= 64.dp && story.bottom <= safeBottom)
        val card = compose.onNodeWithText("Episode One").getUnclippedBoundsInRoot()
        assertTrue("Episode not fully visible: $card", card.top >= 64.dp && card.bottom <= safeBottom)
        val cast = compose.onNodeWithText("Cast").getUnclippedBoundsInRoot()
        val initialPortrait = compose.onNodeWithTag("cast-portrait-Actor One", useUnmergedTree = true).getUnclippedBoundsInRoot()
        // Cast is atomic at the fold, like the film page: when the band has room the heading and the portraits are both
        // inside it; when it has not, the whole section starts after the first viewport and the reveal scroll brings it
        // in as one block. A heading whose portraits are cut by the bottom of the screen is the defect.
        val castInBand = initialPortrait.bottom <= safeBottom
        assertTrue(
            "Stranded Cast heading: $cast / $initialPortrait",
            if (castInBand) cast.top >= 64.dp else cast.top >= safeBottom,
        )
        compose.onNodeWithText("Episode One").performKeyInput { pressKey(Key.DirectionDown) }
        compose.onNodeWithText("Actor One").assertIsFocused()
        shot("show-$name-cast")
        val portrait = compose.onNodeWithTag("cast-portrait-Actor One", useUnmergedTree = true).getUnclippedBoundsInRoot()
        assertTrue("Cast portrait clipped: $portrait", portrait.top >= 64.dp && portrait.bottom <= safeBottom)
        assertTrue("Cast heading lost on reveal", compose.onNodeWithText("Cast").getUnclippedBoundsInRoot().top >= 64.dp)
        compose.onNodeWithText("Actor One").performKeyInput { pressKey(Key.DirectionUp) }
        compose.onNodeWithText("Episode One").assertIsFocused()
        assertTrue("Title not restored", compose.onNodeWithText(heroTitle).getUnclippedBoundsInRoot().top >= 64.dp)
        compose.onNodeWithText("Episode One").performKeyInput { pressKey(Key.DirectionUp) }
        val control = compose.onNode(isFocused()).fetchSemanticsNode().config[androidx.compose.ui.semantics.SemanticsProperties.Text].first().text
        assertTrue("Up from episodes must return to controls: $control", control == "Play" || control.startsWith("Season "))
    }
}
