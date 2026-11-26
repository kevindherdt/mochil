package net.kdhmedia.mochil.ui

import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import net.kdhmedia.mochil.catalogue.BookStatusFilter
import net.kdhmedia.mochil.catalogue.LibrarySummary
import net.kdhmedia.mochil.ui.kit.ChipRow
import net.kdhmedia.mochil.ui.kit.PillItem
import net.kdhmedia.mochil.ui.kit.PillRow
import net.kdhmedia.mochil.ui.theme.MochilChrome

/** The reading-status chips: which one is chosen and what a tap does. Present only where the view has a list to filter. */
internal data class StatusChips(val active: BookStatusFilter, val onSelect: (BookStatusFilter) -> Unit)

/**
 * The Library header on the phone (web/src/features/library/screens/LibraryScreen.tsx, ScreenHeader): the libraries as pills,
 * and under them the status chips at the left and the screen's own tools (search, view) at the right. It replaces the old
 * drop-down switcher; the dock already holds Notes, so a notes vault is not a shelf and has no pill.
 *
 * The row of tools is kept for every library whose screen puts tools there, empty or not: the screen below hands its tools up a frame
 * after it first draws, and a row that appeared then would push the covers down under the reader's thumb.
 * Both rows are 44dp targets, so the band is 88dp where the web's is 80. The covers start a little lower than the web's 96 until
 * step 4 takes the cover grid's own 20dp padding down to the header's 16.
 */
@Composable
internal fun LibraryHeader(
    libraries: List<LibrarySummary>,
    selected: LibrarySummary?,
    onSelect: (LibrarySummary) -> Unit,
    statusChips: StatusChips?,
    actions: List<HeaderAction>?,
    offlineMessage: String?,
    onRetry: () -> Unit,
    modifier: Modifier = Modifier,
    switcher: Boolean = true,
    /** False for a shelf whose screen keeps its own controls (music), so the band is not a blank row above them. */
    toolsRow: Boolean = true,
) {
    Column(modifier.fillMaxWidth()) {
        if (switcher && selected != null && !isVault(selected)) {
            val shelves = remember(libraries) { libraries.filterNot(::isVault) }
            val items = remember(shelves) { shelves.map { PillItem(it.id, it.name, libraryGlyph(it.kind, it.id)) } }
            PillRow(items, selected.id, { id -> shelves.firstOrNull { it.id == id }?.let(onSelect) })
            if (toolsRow) Row(Modifier.fillMaxWidth().heightIn(min = MochilChrome.touchMin), verticalAlignment = Alignment.CenterVertically) {
                Box(Modifier.weight(1f)) { if (statusChips != null) StatusChipRow(statusChips) }
                actions?.let { HeaderActionButtons(it) }
            }
        }
        if (offlineMessage != null) Row(verticalAlignment = Alignment.CenterVertically) {
            Text(offlineMessage, Modifier.weight(1f), style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.primary)
            TextButton(onClick = onRetry) { Text("Retry") }
        }
    }
}

/** The chips on their own, for the wide layout, which has no pill row. */
@Composable
internal fun StatusChipRow(chips: StatusChips, modifier: Modifier = Modifier) {
    val items = remember { BookStatusFilter.entries.map { PillItem(it.name, it.label) } }
    ChipRow(items, chips.active.name, { id -> chips.onSelect(BookStatusFilter.valueOf(id)) }, modifier)
}
