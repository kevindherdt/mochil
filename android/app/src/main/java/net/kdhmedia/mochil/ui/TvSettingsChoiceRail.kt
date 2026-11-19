package net.kdhmedia.mochil.ui

import androidx.compose.foundation.focusGroup
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.FilterChip
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.focus.focusRestorer
import androidx.compose.ui.unit.dp

/** A bounded, focus-following option rail for ten-foot settings surfaces. */
@Composable
internal fun <T> TvSettingsChoiceRail(
    choices: List<T>,
    selected: (T) -> Boolean,
    label: (T) -> String,
    key: (T) -> Any,
    onSelect: (T) -> Unit,
    modifier: Modifier = Modifier,
) {
    LazyRow(
        modifier = modifier.focusRestorer().focusGroup(),
        horizontalArrangement = Arrangement.spacedBy(12.dp),
        contentPadding = PaddingValues(end = 28.dp),
    ) {
        items(choices, key = key) { choice ->
            FilterChip(
                selected = selected(choice),
                onClick = { onSelect(choice) },
                label = { Text(label(choice), maxLines = 1) },
                modifier = Modifier.heightIn(min = 48.dp).tvFocusBorder(),
            )
        }
    }
}
