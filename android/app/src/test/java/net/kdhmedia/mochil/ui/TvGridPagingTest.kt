package net.kdhmedia.mochil.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.unit.dp
import net.kdhmedia.mochil.ui.theme.AppPalette
import net.kdhmedia.mochil.ui.theme.AppThemeMode
import net.kdhmedia.mochil.ui.theme.AppThemeSelection
import net.kdhmedia.mochil.ui.theme.MochilTheme
import org.junit.Assert.assertEquals
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config

/**
 * The grid's paging effect is keyed on the grid alone, and remembers the total it last asked about. It used to be keyed on
 * `loading`, so every completed page restarted the collector, whose fresh snapshot re-emitted the pair it had already acted
 * on: a page that filled the viewport asked for the next one immediately, and a server answering an empty page with a cursor
 * looped forever. This mounts the real grid and counts the requests.
 */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35], qualifiers = "w960dp-h540dp-mdpi")
class TvGridPagingTest {
    @get:Rule val compose = createComposeRule()

    @Test
    fun aPageThatFillsTheViewportAsksOnceAndDoesNotAskAgainWhenItCompletes() {
        var requests = 0
        compose.setContent {
            // An explicit selection: the theme's own fallback reads device features Robolectric does not answer.
            MochilTheme(AppThemeSelection(AppPalette.Tide, AppThemeMode.Dark), glow = false, cinema = true) {
                var loading by remember { mutableStateOf(false) }
                val rows = remember { (0 until 8).toList() }
                Box(Modifier.fillMaxSize()) {
                    TvMediaGrid(
                        paging = TvPagingState(
                            hasNextPage = true,
                            loading = loading,
                            onLoadNextPage = {
                                requests++
                                loading = true
                            },
                            onRetry = {},
                            error = null,
                        ),
                        minColumnWidth = 200.dp,
                        positionKey = "test",
                    ) {
                        items(rows, key = { it }) { index ->
                            Box(Modifier.height(120.dp).background(Color.DarkGray))
                        }
                    }
                }
                // The page "arrives": the item count does not grow (the server said nothing new) and loading clears. A grid
                // that re-asked on that restart would ask here, and again on every completion after it.
                androidx.compose.runtime.LaunchedEffect(requests) {
                    if (requests > 0) loading = false
                }
            }
        }
        compose.waitForIdle()
        compose.mainClock.advanceTimeBy(500)
        compose.waitForIdle()
        assertEquals("the grid asked for a page more than once without new items", 1, requests)
    }
}
