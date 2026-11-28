package net.kdhmedia.mochil.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.runtime.getValue
import androidx.compose.runtime.produceState
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.semantics.stateDescription
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.delay
import net.kdhmedia.mochil.ui.theme.MochilChrome
import net.kdhmedia.mochil.ui.theme.MochilRadius
import net.kdhmedia.mochil.ui.theme.MochilSpace
import java.time.LocalTime
import java.time.format.DateTimeFormatter

/**
 * The corner cluster the TV sheets put over every screen. It is the clock and nothing else, deliberately: the sheets also
 * draw a "Connected" pill, but the app has no reachability signal threaded to this corner, and a pill that says Connected
 * while the server is down is worse than an empty corner. The same reasoning keeps a latency figure out of it.
 */
@Composable
internal fun TvStatusCluster(modifier: Modifier = Modifier) {
    val start = remember { nowText() }
    val clock by produceState(initialValue = start) {
        while (true) {
            delay(20_000)
            value = nowText()
        }
    }
    Text(
        clock,
        modifier,
        style = MaterialTheme.typography.labelLarge,
        color = MaterialTheme.colorScheme.onSurfaceVariant,
    )
}

private val clockFormat = DateTimeFormatter.ofPattern("HH:mm")
private fun nowText() = LocalTime.now().format(clockFormat)
