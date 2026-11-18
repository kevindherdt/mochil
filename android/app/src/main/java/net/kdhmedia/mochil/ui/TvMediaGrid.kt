package net.kdhmedia.mochil.ui

import androidx.compose.foundation.border
import androidx.compose.foundation.focusGroup
import androidx.compose.foundation.background
import androidx.compose.foundation.Image
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.draw.clip
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.GridItemSpan
import androidx.compose.foundation.lazy.grid.LazyGridScope
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.rememberLazyGridState
import androidx.compose.foundation.lazy.grid.LazyGridState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Check
import androidx.compose.material3.Icon
import androidx.compose.material3.Surface
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.Button
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.withFrameNanos
import kotlinx.coroutines.launch
import androidx.compose.runtime.setValue
import androidx.compose.runtime.snapshotFlow
import androidx.compose.ui.Alignment
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.draw.blur
import androidx.compose.ui.Modifier
import androidx.compose.ui.focus.focusRestorer
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.focus.focusProperties
import androidx.compose.ui.focus.onFocusChanged
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import coil3.compose.rememberAsyncImagePainter
import coil3.size.Size
import kotlinx.coroutines.flow.distinctUntilChanged

/** Width over height of the artwork a card shows; cards fill their grid cell and keep the artwork's own shape. */
internal enum class TvArtworkRatio(val aspect: Float) {
    Portrait(2f / 3f), Square(1f), Landscape(16f / 9f),
}

/** Lets the library screen hand focus to the first card once content has loaded. */
internal val LocalTvContentFocus = androidx.compose.runtime.staticCompositionLocalOf<FocusRequester?> { null }
private val LocalTvRememberedCardFocus = androidx.compose.runtime.staticCompositionLocalOf<FocusRequester?> { null }
private val LocalTvRestoringKey = androidx.compose.runtime.staticCompositionLocalOf<String?> { null }
private val LocalTvCardFocusChanged = androidx.compose.runtime.staticCompositionLocalOf<() -> Unit> { {} }

/** focusIndex maps a focus identity to its current grid slot, including any non-card header slots. */
@Composable
@OptIn(androidx.compose.foundation.ExperimentalFoundationApi::class)
internal fun TvMediaGrid(paging: TvPagingState? = null, minColumnWidth: Dp = 120.dp, positionKey: String = "default", positions: TvGridPositionStore? = null, focusIndex: (String) -> Int = { -1 }, content: LazyGridScope.() -> Unit) {
    val restored = positions?.get(positionKey)
    val gridState = remember(positionKey) { LazyGridState(restored?.index ?: 0, restored?.offset ?: 0) }
    DisposableEffect(positionKey, gridState, positions) {
        onDispose { positions?.put(positionKey, gridState.firstVisibleItemIndex, gridState.firstVisibleItemScrollOffset) }
    }
    LaunchedEffect(gridState, paging?.hasNextPage, paging?.loading) {
        val currentPaging = paging ?: return@LaunchedEffect
        snapshotFlow {
            val layout = gridState.layoutInfo
            layout.totalItemsCount to (layout.visibleItemsInfo.lastOrNull()?.index ?: -1)
        }.distinctUntilChanged().collect { (total, lastVisible) ->
            if (shouldLoadTvPage(total, lastVisible, currentPaging.hasNextPage, currentPaging.loading)) currentPaging.onLoadNextPage()
        }
    }
    val gridFocused = remember { mutableStateOf(false) }
    val rememberedCardFocus = remember(positionKey) { FocusRequester() }
    val scope = rememberCoroutineScope()
    var restoringKey by remember(positionKey) { mutableStateOf<String?>(null) }
    val focusChanges = remember { intArrayOf(0) }
    val restorations = remember { intArrayOf(0) }
    val currentFocusIndex by androidx.compose.runtime.rememberUpdatedState(focusIndex)
    CompositionLocalProvider(androidx.compose.foundation.gestures.LocalBringIntoViewSpec provides TvGridBringIntoView, LocalTvGridFocused provides gridFocused, LocalTvRememberedCardFocus provides rememberedCardFocus, LocalTvRestoringKey provides restoringKey, LocalTvCardFocusChanged provides { focusChanges[0]++ }) {
    LazyVerticalGrid(
        columns = GridCells.Adaptive(minColumnWidth),
        state = gridState,
        modifier = Modifier.fillMaxSize().then(LocalTvContentFocus.current?.let { Modifier.focusRequester(it) } ?: Modifier).onFocusChanged { gridFocused.value = it.hasFocus }.focusProperties { onEnter = {
            val key = positions?.focusedItem(positionKey)
            val index = key?.let(focusIndex) ?: -1
            if (key != null && index >= 0 && gridState.layoutInfo.visibleItemsInfo.none { it.index == index }) {
                restoringKey = key
                val initialFocusChanges = focusChanges[0]
                val restoration = ++restorations[0]
                scope.launch {
                    try {
                        withFrameNanos { }
                        if (gridFocused.value && focusChanges[0] <= initialFocusChanges + 1) {
                            val currentIndex = currentFocusIndex(key)
                            if (currentIndex >= 0) {
                                gridState.scrollToItem(currentIndex)
                                withFrameNanos { }
                                if (gridFocused.value && focusChanges[0] <= initialFocusChanges + 1) runCatching { rememberedCardFocus.requestFocus() }
                            }
                        }
                    } finally {
                        if (restorations[0] == restoration) restoringKey = null
                    }
                }
            } else runCatching { rememberedCardFocus.requestFocus() }
        } }.focusRestorer().focusGroup(),
        contentPadding = PaddingValues(start = 40.dp, end = 40.dp, top = 8.dp, bottom = 24.dp),
        horizontalArrangement = Arrangement.spacedBy(16.dp),
        verticalArrangement = Arrangement.spacedBy(16.dp),
    ) {
        content()
        if (paging?.error != null) item("paging-error", span = { GridItemSpan(maxLineSpan) }, contentType = "paging-error") {
            Column(Modifier.fillMaxWidth().padding(24.dp), horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.spacedBy(10.dp)) {
                Text(paging.error, color = MaterialTheme.colorScheme.onSurfaceVariant)
                // A control taking focus is always a newer choice, even if it is the first grid child focused on entry.
                Button(onClick = paging.onRetry, modifier = Modifier.onFocusChanged { if (it.isFocused) focusChanges[0] += 2 }.tvFocusBorder()) { Text("Retry") }
            }
        } else if (paging?.loading == true) item("paging-progress", contentType = "paging-progress") {
            AccessibleLoadingIndicator("Loading more items", Modifier.height(220.dp), size = 28.dp)
        }
    }
    }
}

/**
 * Focused cards grow a little, which pushes their bounds just past the viewport edge; the default behaviour then scrolls
 * to compensate and back again on the next move, so the grid seemed to wobble. Only a card that is really off screen scrolls.
 */
@OptIn(androidx.compose.foundation.ExperimentalFoundationApi::class)
private val TvGridBringIntoView = object : androidx.compose.foundation.gestures.BringIntoViewSpec {
    override fun calculateScrollDistance(offset: Float, size: Float, containerSize: Float): Float {
        val slack = size * .12f
        val trailing = offset + size
        return when {
            offset >= -slack && trailing <= containerSize + slack -> 0f
            offset < 0f -> offset
            else -> trailing - containerSize
        }
    }
}

internal data class TvGridPosition(val index: Int, val offset: Int, val focusedItemKey: String? = null)

internal data class TvGridFocusIdentity(
    val routeKey: String,
    val itemKey: String,
    val positions: TvGridPositionStore,
)

internal class TvGridPositionStore(private val maximumEntries: Int = 96) {
    init { require(maximumEntries > 0) }
    private val values = LinkedHashMap<String, TvGridPosition>(maximumEntries, 0.75f, true)
    fun get(key: String): TvGridPosition? = values[key]
    fun put(key: String, index: Int, offset: Int) {
        values[key] = TvGridPosition(index.coerceAtLeast(0), offset.coerceAtLeast(0), values[key]?.focusedItemKey)
        trim()
    }
    fun focus(key: String, itemKey: String) {
        val current = values[key] ?: TvGridPosition(0, 0)
        values[key] = current.copy(focusedItemKey = itemKey)
        trim()
    }
    fun hasFocus(key: String): Boolean = !values[key]?.focusedItemKey.isNullOrBlank()
    fun focusedItem(key: String): String? = values[key]?.focusedItemKey
    private fun trim() {
        while (values.size > maximumEntries) values.remove(values.entries.first().key)
    }
}

internal data class TvGridRouteKey(
    val libraryId: String,
    val browseMode: String,
    val layout: String,
    val hierarchy: String,
    val sort: String,
    val filter: String,
    val query: String,
    val bucket: String,
) {
    fun stableKey(): String = listOf(libraryId, browseMode, layout, hierarchy, sort, filter, query, bucket)
        .joinToString("|") { "${it.length}:$it" }
}

@Composable
internal fun TvWideMediaCard(
    title: String,
    subtitle: String,
    artwork: String,
    authorizationHeader: String?,
    onClick: () -> Unit,
    progress: Float = 0f,
    completed: Boolean = false,
    active: Boolean = false,
    onLongClick: (() -> Unit)? = null,
    focusIdentity: TvGridFocusIdentity? = null,
    onFocused: () -> Unit = {},
) {
    var focused by remember { mutableStateOf(false) }
    val rememberedCardFocus = LocalTvRememberedCardFocus.current
    val standaloneFocus = remember(focusIdentity?.routeKey, focusIdentity?.itemKey) { FocusRequester() }
    val restoringKey = LocalTvRestoringKey.current
    val onCardFocusChanged = LocalTvCardFocusChanged.current
    val isRemembered = focusIdentity?.let { (restoringKey ?: it.positions.focusedItem(it.routeKey)) == it.itemKey } == true
    if (rememberedCardFocus == null && isRemembered) {
        LaunchedEffect(focusIdentity.routeKey, focusIdentity.itemKey) { standaloneFocus.requestFocus() }
    }
    val shape = RoundedCornerShape(14.dp)
    Card(
        onClick = onClick,
        // Height follows the text: a fixed box clipped two-line titles once the 10-foot type scale
        // landed. min() keeps short cards from shrinking into the artwork.
        modifier = Modifier.fillMaxWidth().heightIn(min = 88.dp)
            .then(if (isRemembered && rememberedCardFocus != null) Modifier.focusRequester(rememberedCardFocus) else if (rememberedCardFocus == null && focusIdentity != null) Modifier.focusRequester(standaloneFocus) else Modifier)
            .tvRemoteLongPress(onLongClick != null) { onLongClick?.invoke() }
            .onFocusChanged {
                focused = it.isFocused
                if (it.isFocused) {
                    onCardFocusChanged()
                    focusIdentity?.let { identity -> identity.positions.focus(identity.routeKey, identity.itemKey) }
                    onFocused()
                }
            }
            // Same grow-and-settle + border as every other card (TvMediaCard); previously a raw,
            // unanimated .scale() that ignored reduce-motion and popped instead of easing in.
            .tvFocusVisual(focused, shape),
        shape = shape,
        colors = CardDefaults.cardColors(containerColor = when {
            focused -> MaterialTheme.colorScheme.primaryContainer.copy(alpha = .62f)
            active -> MaterialTheme.colorScheme.primaryContainer.copy(alpha = .3f)
            else -> MaterialTheme.colorScheme.surface
        }),
    ) {
        Row(Modifier.fillMaxSize(), verticalAlignment = Alignment.CenterVertically) {
            Box(Modifier.width(70.dp).height(70.dp)) {
                AuthenticatedArtwork(artwork, authorizationHeader, title, Modifier.fillMaxSize(), cornerRadius = 0.dp)
                if (progress > 0f || completed) LinearProgressIndicator(
                    progress = { if (completed) 1f else progress.coerceIn(0f, 1f) },
                    modifier = Modifier.fillMaxWidth().height(4.dp).align(Alignment.BottomCenter),
                )
            }
            Column(Modifier.weight(1f).padding(horizontal = 10.dp), verticalArrangement = Arrangement.spacedBy(2.dp)) {
                Text(title, style = MaterialTheme.typography.titleSmall, fontWeight = FontWeight.SemiBold, maxLines = 2, overflow = TextOverflow.Ellipsis)
                Text(subtitle, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant, maxLines = 1, overflow = TextOverflow.Ellipsis)
            }
        }
    }
}

/** True while something in the grid has focus, so the other cards can step back. */
internal val LocalTvGridFocused = androidx.compose.runtime.staticCompositionLocalOf { mutableStateOf(false) }

/** Matched shows and movies carry a tall TMDB poster; everything else on a wide tile is a wide still. */
internal fun tvIsPosterArtwork(url: String) = "metadata-artwork" in url && "backdrop=1" !in url

/** Art that fills a wide area: a wide still is cropped to fit, a tall poster sits whole on the right over a dimmed copy of itself. */
@Composable
internal fun TvWideArtwork(
    art: String,
    authorizationHeader: String?,
    title: String,
    modifier: Modifier = Modifier,
    posterEndPadding: androidx.compose.ui.unit.Dp = 10.dp,
    blurredBackdrop: Boolean = false,
) {
    if (!tvIsPosterArtwork(art)) return AuthenticatedArtwork(art, authorizationHeader, title, modifier, cornerRadius = 0.dp)
    BoxWithConstraints(modifier) {
        // One decode, drawn twice. Two AsyncImage widgets asked Coil for the same file at two different constraints,
        // so every poster card in the grid decoded its artwork twice — one copy only to be blurred away. The request
        // carries the card's own pixel size, so the painter is not re-resolved by the two draws either.
        val context = LocalContext.current
        val request = remember(art, authorizationHeader, constraints.maxWidth, constraints.maxHeight) {
            val bounded = constraints.maxWidth != androidx.compose.ui.unit.Constraints.Infinity &&
                constraints.maxHeight != androidx.compose.ui.unit.Constraints.Infinity
            authenticatedArtworkRequest(
                context, art, authorizationHeader,
                size = if (bounded) Size(constraints.maxWidth, constraints.maxHeight) else null,
            )
        }
        val painter = rememberAsyncImagePainter(request)
        Image(
            painter = painter,
            contentDescription = null,
            // The live blur is the focused card's alone: it is the one being looked at, and blurring every card in a
            // poster wall is per-frame work on the whole grid. Unfocused cards keep the same dimmed backdrop, sharp.
            modifier = Modifier.fillMaxSize().then(if (blurredBackdrop) Modifier.blur(24.dp) else Modifier)
                .background(Color.Black.copy(alpha = .35f)),
            contentScale = ContentScale.Crop,
        )
        Image(
            painter = painter,
            contentDescription = "Artwork for $title",
            modifier = Modifier.align(Alignment.CenterEnd).padding(end = posterEndPadding, top = 8.dp, bottom = 8.dp)
                .fillMaxHeight().aspectRatio(2f / 3f).clip(RoundedCornerShape(6.dp)),
            contentScale = ContentScale.Crop,
        )
    }
}

@Composable
internal fun TvMediaCard(
    title: String,
    subtitle: String,
    artwork: String,
    authorizationHeader: String?,
    ratio: TvArtworkRatio,
    onClick: () -> Unit,
    progress: Float = 0f,
    completed: Boolean = false,
    onLongClick: (() -> Unit)? = null,
    focusIdentity: TvGridFocusIdentity? = null,
    onFocused: () -> Unit = {},
) {
    var focused by remember { mutableStateOf(false) }
    val rememberedCardFocus = LocalTvRememberedCardFocus.current
    val restoringKey = LocalTvRestoringKey.current
    val onCardFocusChanged = LocalTvCardFocusChanged.current
    val shape = RoundedCornerShape(12.dp)
    val gridFocused = LocalTvGridFocused.current
    val overlay = ratio == TvArtworkRatio.Landscape
    // Wide stills carry their title on a shade over the art; covers already hold lettering, so their caption sits under them.
    Card(
        onClick = onClick,
        modifier = Modifier.fillMaxWidth()
            .then(if (focusIdentity != null && (restoringKey ?: focusIdentity.positions.focusedItem(focusIdentity.routeKey)) == focusIdentity.itemKey && rememberedCardFocus != null) Modifier.focusRequester(rememberedCardFocus) else Modifier)
            .tvRemoteLongPress(onLongClick != null) { onLongClick?.invoke() }
            .onFocusChanged {
                focused = it.isFocused
                if (it.isFocused) {
                    onCardFocusChanged()
                    focusIdentity?.let { identity -> identity.positions.focus(identity.routeKey, identity.itemKey) }
                    onFocused()
                }
            }
            .graphicsLayer { alpha = if (!focused && gridFocused.value) .7f else 1f }
            .tvFocusVisual(focused, shape, focusedScale = 1.06f),
        shape = shape,
        colors = CardDefaults.cardColors(containerColor = if (overlay) MaterialTheme.colorScheme.surface else if (focused) MaterialTheme.colorScheme.surfaceVariant else MaterialTheme.colorScheme.surface),
    ) {
        Column {
            // A hairline: the card fill sits about 1.1:1 above the page, so without an edge a
            // poster wall reads as captions floating on black.
            Box(Modifier.fillMaxWidth().aspectRatio(ratio.aspect)
                .border(1.dp, MaterialTheme.colorScheme.outline.copy(alpha = .30f), shape)) {
                TvWideArtwork(artwork, authorizationHeader, title, Modifier.fillMaxSize(), blurredBackdrop = focused)
                if (overlay) {
                    Box(Modifier.align(Alignment.BottomCenter).fillMaxWidth().fillMaxHeight(.55f).background(Brush.verticalGradient(listOf(Color.Transparent, Color.Black.copy(alpha = .85f)))))
                    Column(Modifier.align(Alignment.BottomStart).padding(start = 12.dp, end = 12.dp, bottom = if (progress > 0f || completed) 14.dp else 10.dp), verticalArrangement = Arrangement.spacedBy(1.dp)) {
                        Text(title, style = MaterialTheme.typography.titleSmall, fontWeight = FontWeight.SemiBold, color = Color.White, maxLines = 2, overflow = TextOverflow.Ellipsis)
                        if (subtitle.isNotBlank()) Text(subtitle, style = MaterialTheme.typography.bodySmall, color = Color.White.copy(alpha = .75f), maxLines = 1, overflow = TextOverflow.Ellipsis)
                    }
                }
                if (progress > 0f || completed) TvProgressBar(if (completed) 1f else progress, Modifier.align(Alignment.BottomCenter).fillMaxWidth())
                if (completed) Surface(Modifier.align(Alignment.TopEnd).padding(8.dp).size(26.dp), shape = androidx.compose.foundation.shape.CircleShape, color = Color.Black.copy(alpha = .55f)) {
                    Icon(Icons.Default.Check, contentDescription = "Finished", tint = Color.White, modifier = Modifier.padding(5.dp))
                }
            }
            if (!overlay) Column(Modifier.padding(horizontal = 10.dp, vertical = 8.dp), verticalArrangement = Arrangement.spacedBy(1.dp)) {
                Text(title, style = MaterialTheme.typography.titleSmall, fontWeight = FontWeight.SemiBold, maxLines = 1, overflow = TextOverflow.Ellipsis)
                Text(subtitle.ifBlank { " " }, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant, maxLines = 1, overflow = TextOverflow.Ellipsis)
            }
        }
    }
}

/** A slim rounded progress strip: accent fill over a pale track, readable on any artwork. */
@Composable
internal fun TvProgressBar(fraction: Float, modifier: Modifier = Modifier, height: Dp = 5.dp) {
    val shape = RoundedCornerShape(50)
    Box(modifier.height(height).background(Color.White.copy(alpha = .28f), shape).clip(shape)) {
        Box(Modifier.fillMaxHeight().fillMaxWidth(fraction.coerceIn(0f, 1f)).background(MaterialTheme.colorScheme.primary, shape))
    }
}

/** Frosted surface: a translucent dark pane with a hairline edge, so artwork behind it shows through softly. */
internal fun Modifier.tvGlass(shape: androidx.compose.ui.graphics.Shape, alpha: Float = .78f): Modifier =
    this.background(Color(0xFF14161D).copy(alpha = alpha), shape).border(1.dp, Color.White.copy(alpha = .10f), shape)
