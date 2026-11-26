package net.kdhmedia.mochil.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.width
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithTag
import androidx.compose.ui.unit.dp
import net.kdhmedia.mochil.catalogue.BookSummary
import net.kdhmedia.mochil.testing.Goldens
import net.kdhmedia.mochil.ui.theme.AppPalette
import net.kdhmedia.mochil.ui.theme.AppThemeMode
import net.kdhmedia.mochil.ui.theme.AppThemeSelection
import net.kdhmedia.mochil.ui.theme.MochilTheme
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import org.robolectric.annotation.GraphicsMode

/**
 * The three card families side by side at a phone's width, so a change to one can be seen against the others: books (kind
 * glyph, Vol. plate, headphones, bar or tick), video (the probe strip), music (square, no chrome).
 */
@RunWith(RobolectricTestRunner::class)
@GraphicsMode(GraphicsMode.Mode.NATIVE)
@Config(sdk = [35], qualifiers = "w390dp-h844dp-xxhdpi")
class CardGoldenTest {
    @get:Rule val compose = createComposeRule()

    @Composable
    private fun Host(content: @Composable () -> Unit) {
        MochilTheme(AppThemeSelection(AppPalette.Tide, AppThemeMode.Dark), glow = false) {
            Surface(Modifier.width(390.dp).testTag("host"), color = MaterialTheme.colorScheme.background) {
                Row(Modifier.fillMaxWidth().background(MaterialTheme.colorScheme.background), horizontalArrangement = Arrangement.spacedBy(8.dp)) { content() }
            }
        }
    }

    /** One grid cell's width at a 390dp phone: two across with an 8dp gap. */
    @Composable
    private fun Cell(content: @Composable () -> Unit) { Box(Modifier.width(171.dp)) { content() } }

    /** A synthetic cover, so the marks sit on something with contrast and the golden shows the marks, not the theme. */
    @Composable
    private fun cover(light: Boolean = false): @Composable (Modifier) -> Unit = { modifier ->
        androidx.compose.foundation.layout.Box(modifier.background(if (light) androidx.compose.ui.graphics.Color(0xFFD8CBB8) else androidx.compose.ui.graphics.Color(0xFF3A3A3A)))
    }

    private fun book(title: String, series: String = "", index: Double? = null, fraction: Double? = null, finished: Boolean = false, readAlong: Boolean = false) =
        BookSummary("id-$title", "books", title, listOf("William Gibson"), series, index, "epub", "application/epub+zip", "en", "", 1, readingFraction = fraction, finished = finished, readAlong = readAlong)

    @Test fun `golden book cards Tide dark`() {
        compose.setContent {
            Host {
                // Two cards in the row a grid draws: each is as wide as its cell, not as wide as the screen.
                Cell { BookCard(book("Neuromancer", "Sprawl", 3.0, fraction = .42, readAlong = true), "books", null, null, {}, {}, art = cover()) }
                Cell { BookCard(book("Count Zero", finished = true), "books", null, null, {}, {}, art = cover()) }
            }
        }
        Goldens.assertMatches(compose.onNodeWithTag("host"), "cards-books-tide-dark")
    }

    @Test fun `golden book cards Ember light`() {
        compose.setContent {
            MochilTheme(AppThemeSelection(AppPalette.Ember, AppThemeMode.Light), glow = false) {
                Surface(Modifier.width(390.dp).testTag("host"), color = MaterialTheme.colorScheme.background) {
                    Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        Cell { BookCard(book("Neuromancer", "Sprawl", 3.0, fraction = .42, readAlong = true), "books", null, null, {}, {}, art = cover(light = true)) }
                        Cell { BookCard(book("Count Zero", finished = true), "books", null, null, {}, {}, art = cover(light = true)) }
                    }
                }
            }
        }
        Goldens.assertMatches(compose.onNodeWithTag("host"), "cards-books-ember-light")
    }

    @Test fun `golden book cards at the largest font`() {
        compose.setContent {
            androidx.compose.runtime.CompositionLocalProvider(
                androidx.compose.ui.platform.LocalDensity provides androidx.compose.ui.unit.Density(androidx.compose.ui.platform.LocalDensity.current.density, 2f),
            ) {
                Host {
                    Cell { BookCard(book("Neuromancer", "Sprawl", 3.0, fraction = .42, readAlong = true), "books", null, null, {}, {}, art = cover()) }
                }
            }
        }
        Goldens.assertMatches(compose.onNodeWithTag("host"), "cards-books-tide-dark-font2")
    }

    @Test fun `golden video card with its probe strip`() {
        compose.setContent {
            Host {
                Cell {
                VideoCard(
                    art = { Box(it.background(MaterialTheme.colorScheme.surfaceVariant)) },
                    aspect = 2f / 3f, title = "Blade Runner", caption = VideoCaption("Movie · 1982 · 118 min", .35f, false), formatLabel = "1080p · Direct Play",
                )
                }
            }
        }
        Goldens.assertMatches(compose.onNodeWithTag("host"), "cards-video-tide-dark")
    }

    @Test fun `golden music album card`() {
        compose.setContent {
            Host {
                Cell {
                MusicAlbumCard(title = "Selected Ambient Works", subtitle = "Aphex Twin · 13 tracks", art = { Box(Modifier.fillMaxSize().background(MaterialTheme.colorScheme.surfaceVariant)) }, onClick = {})
                }
            }
        }
        Goldens.assertMatches(compose.onNodeWithTag("host"), "cards-music-tide-dark")
    }
}
