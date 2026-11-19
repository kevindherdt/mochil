package net.kdhmedia.mochil.ui

import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.focus.onFocusChanged
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.semantics.selected
import androidx.compose.ui.unit.dp

/** Single-line TV jump rail; only server-confirmed buckets are rendered. */
@Composable
internal fun TvAlphabetGrid(buckets: List<String>, selected: String, onSelect: (String) -> Unit) {
    val values = buckets.distinct()
    val state = rememberLazyListState()
    LaunchedEffect(selected, values) { values.indexOf(selected).takeIf { it >= 0 }?.let { state.animateScrollToItem(it) } }
    LazyRow(
        state = state,
        modifier = Modifier.fillMaxWidth().height(48.dp),
        horizontalArrangement = Arrangement.spacedBy(4.dp),
        contentPadding = PaddingValues(horizontal = 64.dp),
    ) {
        items(values, key = { it }) { bucket -> TvAlphabetButton(bucket, bucket == selected) { onSelect(bucket) } }
    }
}

@Composable
private fun TvAlphabetButton(label: String, selected: Boolean, onClick: () -> Unit) {
    var focused by remember { mutableStateOf(false) }
    val shape = RoundedCornerShape(8.dp)
    Card(
        onClick = onClick,
        modifier = Modifier.size(48.dp).semantics { this.selected = selected }.onFocusChanged { focused = it.isFocused }
            .tvFocusVisual(focused, shape, selected, focusedScale = 1f),
        shape = shape,
        colors = CardDefaults.cardColors(containerColor = if (selected || focused) MaterialTheme.colorScheme.primaryContainer else Color.Transparent),
    ) { Box(Modifier.size(48.dp), contentAlignment = Alignment.Center) { Text(label, style = MaterialTheme.typography.labelLarge) } }
}
