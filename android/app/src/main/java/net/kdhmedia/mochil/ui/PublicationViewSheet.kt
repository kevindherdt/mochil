package net.kdhmedia.mochil.ui

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ViewList
import androidx.compose.material.icons.filled.GridView
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilterChip
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import net.kdhmedia.mochil.catalogue.LibraryState
import net.kdhmedia.mochil.catalogue.PublicationGrouping
import net.kdhmedia.mochil.catalogue.PublicationLayout
import net.kdhmedia.mochil.catalogue.currentLayout

/** The ways books can be grouped; "None" is the plain list of every book, the only one with an order of its own. */
internal val publicationGroupChoices = listOf(PublicationGrouping.Series, PublicationGrouping.Authors, PublicationGrouping.Books)
internal val publicationOrderChoices = listOf("" to "Title", "author" to "Author", "recent" to "Recently added")

internal fun publicationGroupLabel(grouping: PublicationGrouping) = when (grouping) {
    PublicationGrouping.Series -> "Series"
    PublicationGrouping.Authors -> "Authors"
    else -> "None"
}

/** Recently added is the same list as the plain list ordered by date, so both count as "None" being on. */
internal fun publicationGroupSelected(current: PublicationGrouping, choice: PublicationGrouping) =
    current == choice || (choice == PublicationGrouping.Books && current == PublicationGrouping.Recent)

/**
 * One small sheet for how a library looks: how to group the books, how to order them when they are not grouped, whether to keep only
 * the ones being read, and covers or list. Every choice is a chip, so the current state reads at a glance. Series keep their own
 * order inside a group (by number, then title) and authors theirs, so there is nothing more to order there.
 */
@OptIn(ExperimentalMaterial3Api::class, ExperimentalLayoutApi::class)
@Composable
internal fun PublicationViewSheet(
    state: LibraryState,
    onGrouping: (PublicationGrouping) -> Unit,
    onSort: (String) -> Unit,
    onLayout: (PublicationLayout) -> Unit,
    onClose: () -> Unit,
) {
    val covers = state.currentLayout() == PublicationLayout.Covers
    val inProgress = state.grouping == PublicationGrouping.Continue
    val plain = state.grouping == PublicationGrouping.Books || state.grouping == PublicationGrouping.Recent
    ModalBottomSheet(onDismissRequest = onClose) {
        Column(Modifier.fillMaxWidth().navigationBarsPadding().padding(horizontal = 20.dp, vertical = 4.dp).padding(bottom = 16.dp), verticalArrangement = Arrangement.spacedBy(10.dp)) {
            SheetLabel("Group by")
            FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                publicationGroupChoices.forEach { choice ->
                    FilterChip(
                        selected = !inProgress && publicationGroupSelected(state.grouping, choice),
                        onClick = { onGrouping(choice); if (choice != PublicationGrouping.Books) onClose() },
                        label = { Text(publicationGroupLabel(choice)) },
                    )
                }
            }
            if (plain) {
                SheetLabel("Order by")
                FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    publicationOrderChoices.forEach { (key, label) ->
                        FilterChip(
                            selected = (state.grouping == PublicationGrouping.Books && state.sort == key) || (state.grouping == PublicationGrouping.Recent && key == "recent"),
                            onClick = { onSort(key); onClose() },
                            label = { Text(label) },
                        )
                    }
                }
            }
            SheetLabel("Show")
            FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                FilterChip(
                    selected = inProgress,
                    onClick = { onGrouping(if (inProgress) PublicationGrouping.Series else PublicationGrouping.Continue); onClose() },
                    label = { Text("Only books in progress") },
                )
            }
            SheetLabel("Layout")
            FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                FilterChip(selected = covers, onClick = { onLayout(PublicationLayout.Covers); onClose() }, label = { Text("Covers") }, leadingIcon = { Icon(Icons.Default.GridView, null, Modifier.size(18.dp)) })
                FilterChip(selected = !covers, onClick = { onLayout(PublicationLayout.List); onClose() }, label = { Text("List") }, leadingIcon = { Icon(Icons.AutoMirrored.Filled.ViewList, null, Modifier.size(18.dp)) })
            }
        }
    }
}

@Composable
private fun SheetLabel(text: String) {
    Text(text, Modifier.padding(top = 6.dp), style = MaterialTheme.typography.labelLarge, color = MaterialTheme.colorScheme.onSurfaceVariant)
}
