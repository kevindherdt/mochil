package net.kdhmedia.mochil.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.combinedClickable
import androidx.compose.foundation.focusable
import androidx.compose.foundation.focusGroup
import androidx.compose.foundation.gestures.detectTapGestures
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.PlayArrow
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.runtime.snapshotFlow
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.focus.focusRestorer
import androidx.compose.ui.focus.onFocusChanged
import androidx.compose.ui.input.key.KeyEventType
import androidx.compose.ui.input.key.onPreviewKeyEvent
import androidx.compose.ui.input.key.type
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.layout.Layout
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.clearAndSetSemantics
import androidx.compose.ui.text.AnnotatedString
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.rememberTextMeasurer
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.Constraints
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import kotlinx.coroutines.flow.distinctUntilChanged
import net.kdhmedia.mochil.video.VideoItem
import net.kdhmedia.mochil.video.VideoShow
import net.kdhmedia.mochil.video.VideoState
import net.kdhmedia.mochil.video.VideoTitleDetails

/** What the big button on a show should do: pick up the first unfinished episode, else start from the top. */
internal fun tvShowPlayTarget(items: List<VideoItem>): VideoItem? =
    items.firstOrNull { !it.finished && it.resumeSeconds > 0 }
        ?: items.firstOrNull { !it.finished }
        ?: items.firstOrNull()

internal fun tvShowMeta(show: VideoShow): String = listOfNotNull(
    show.year.takeIf { it > 0 }?.toString(),
    tvCount(show.seasons.size, "season"),
    tvCount(show.count, "episode"),
).joinToString(" · ")

/** Season 0 is the extras folder, labelled the same way the phone labels it. */
internal fun tvSeasonLabel(season: Int): String = if (season == 0) "Specials" else "Season $season"

internal fun tvEpisodeLabel(item: VideoItem): String = listOfNotNull(
    item.episode.takeIf { it > 0 }?.let { "S${item.season} · E$it" },
    item.durationSeconds.takeIf { it > 0 }?.let { "${(it / 60).toInt().coerceAtLeast(1)} min" },
).joinToString(" · ")

/** The synopsis is supporting text: a step under bodyMedium, which is sized for a book page. */
internal fun tvDetailStoryStyle(base: TextStyle) = base.copy(fontSize = 16.sp, lineHeight = 21.sp)

/**
 * The heights a show page has to fit inside one band, chosen in a fixed order: three lines of story, then the largest
 * episode card, then the full cast row, and only then the year/genre line. Every input is a measured height from the
 * style the block is drawn with, never a guessed constant, so the choice matches what is placed.
 */
internal data class TvShowBudget(
    val storyLines: Int,
    val cardWidth: Dp,
    val compactCast: Boolean,
    val showFacts: Boolean,
    val showReadMore: Boolean = true,
)

internal fun tvShowDetailBudget(
    band: Dp,
    titleHeight: Dp,
    factsHeight: Dp,
    controlsHeight: Dp,
    storyHeight: (lines: Int, readMore: Boolean) -> Dp,
    railHeight: (cardWidth: Dp) -> Dp,
    castHeight: (compact: Boolean) -> Dp,
    gap: Dp,
    storyLinesShown: Int = 3,
    cardWidths: List<Dp> = tvShowCardWidths,
): TvShowBudget {
    fun search(facts: Dp, cast: (compact: Boolean) -> Dp, readMore: Boolean, widths: List<Dp> = cardWidths): TvShowBudget? {
        // The description outranks the year/genre line: it never drops below one line while there is a story.
        for (lines in if (storyLinesShown <= 0) listOf(0) else storyLinesShown downTo 1) {
            for (cardWidth in widths) {
                for (compact in listOf(false, true)) {
                    val children = 3 + (if (facts > 0.dp) 1 else 0) + (if (lines > 0) 1 else 0)
                    val total = titleHeight + facts + storyHeight(lines, readMore) + controlsHeight +
                        railHeight(cardWidth) + cast(compact) + gap * (children - 1)
                    if (total <= band) return TvShowBudget(lines, cardWidth, compact, facts > 0.dp, readMore)
                }
            }
        }
        return null
    }
    return search(factsHeight, castHeight, readMore = true)
        ?: search(0.dp, castHeight, readMore = true)
        // A window that cannot hold the cast as well still keeps title, description, controls and episodes whole: the
        // cast follows the rail and the page scrolls to it, the same way the film page hands its cast to the scroll.
        ?: search(factsHeight, { 0.dp }, readMore = true)
        ?: search(0.dp, { 0.dp }, readMore = true)
        // Last: a window this short keeps the header whole by giving up the Read more row and shrinking the cards below
        // the normal ladder, so the focused episode stays exactly where the viewer put it.
        ?: search(0.dp, { 0.dp }, readMore = false, widths = cardWidths + listOf(150.dp, 130.dp))
        ?: TvShowBudget(1, 130.dp, compactCast = false, showFacts = false)
}

/**
 * The episode card widths a show page may use, widest first. The layout walks this ladder down until the page fits.
 */
internal val tvShowCardWidths = listOf(270.dp, 250.dp, 230.dp, 210.dp, 190.dp, 170.dp, 150.dp)

/**
 * A show opened on TV. One viewport holds the title, the facts, the description, Play and the seasons, the episode
 * rail and the cast, the same shape as the film page, so a show does not feel like a different app.
 */
@Composable
internal fun TvShowDetail(
    show: VideoShow,
    state: VideoState,
    auth: String?,
    artwork: (String) -> String,
    onSeason: (Int) -> Unit,
    onPlay: (VideoItem) -> Unit,
    onSetWatched: (VideoItem, Boolean) -> Unit,
    onMore: () -> Unit,
    onFocus: (TvFocusDetail) -> Unit,
    onDetails: (VideoShow?, VideoItem?) -> Unit = { _, _ -> },
    modifier: Modifier = Modifier,
) {
    LaunchedEffect(show.sourceName) { onDetails(show, null) }
    val details = state.details["show:${show.sourceName}"]
    var focusedEpisodeId by remember(show.sourceName, state.selectedSeason) { mutableStateOf<String?>(null) }
    val focusedEpisode = state.items.firstOrNull { it.id == focusedEpisodeId }
    val railState = rememberLazyListState()
    val seasonsState = rememberLazyListState()
    val railFocus = remember { FocusRequester() }
    var synopsisOpen by remember(show.sourceName) { mutableStateOf(false) }
    val synopsisFocus = remember { FocusRequester() }
    val pageScroll = rememberScrollState()
    var railOffset by remember { mutableStateOf<Int?>(null) }
    LaunchedEffect(focusedEpisodeId) {
        // Only matters on a window too short for the whole page; on a normal TV nothing scrolls.
        if (focusedEpisodeId != null) railOffset?.let { pageScroll.animateScrollTo(it) }
    }
    // Keep the current season on screen wherever it was chosen (a remote press, or a season the ViewModel restored),
    // so the row never sits showing the wrong part of a long show.
    LaunchedEffect(show.sourceName, state.selectedSeason) {
        val index = show.seasons.indexOf(state.selectedSeason)
        if (index >= 0) seasonsState.animateScrollToItem(index)
    }
    val target = tvShowPlayTarget(state.items)
    val title = tvCleanTitle(show.name)
    // The focused episode's own synopsis replaces the show's in the same slot, so there is never a second paragraph.
    val story = focusedEpisode?.overview?.takeIf(String::isNotBlank) ?: show.overview
    val facts = details?.let { tvFactsLine(show.year, it) }.orEmpty().ifBlank { tvShowMeta(show) }
    TvBackdropEffect(show.backdropPath.takeIf(String::isNotBlank)?.let(artwork) ?: artwork(show.artworkPath))
    LaunchedEffect(railState, state.nextOffset, state.loadingMore) {
        if (state.nextOffset == null || state.loadingMore) return@LaunchedEffect
        snapshotFlow { railState.layoutInfo.let { it.totalItemsCount to (it.visibleItemsInfo.lastOrNull()?.index ?: -1) } }
            .distinctUntilChanged()
            .collect { (total, last) -> if (last >= total - 3) onMore() }
    }
    Box(modifier.fillMaxSize()) {
        BoxWithConstraints(Modifier.fillMaxSize().padding(start = 20.dp, end = 44.dp, top = 64.dp, bottom = 20.dp)) {
            val titleStyle = MaterialTheme.typography.headlineSmall.copy(fontSize = 26.sp, lineHeight = 30.sp)
            val storyStyle = tvDetailStoryStyle(MaterialTheme.typography.bodyMedium)
            val gap = 4.dp
            val band = maxHeight
            val density = LocalDensity.current
            val measurer = rememberTextMeasurer()
            fun lineHeight(style: TextStyle) = with(density) { style.lineHeight.toDp() }
            val storyLine = lineHeight(storyStyle)
            val readMoreLine = lineHeight(MaterialTheme.typography.labelMedium)
            // The synopsis keeps its Read more row when it overflows, so that row is charged before the lines are set.
            val storyLinesNeeded = if (story.isBlank()) 0 else with(density) {
                measurer.measure(
                    AnnotatedString(story), style = storyStyle, maxLines = Int.MAX_VALUE,
                    constraints = Constraints(maxWidth = maxWidth.roundToPx()),
                ).lineCount
            }
            fun storyBlockHeight(lines: Int, readMore: Boolean): Dp = when {
                lines <= 0 -> 0.dp
                readMore && storyLinesNeeded > lines -> storyLine * lines + 4.dp + readMoreLine + 2.dp
                else -> storyLine * lines + 4.dp
            }
            val castHeading = lineHeight(MaterialTheme.typography.titleSmall)
            val castName = lineHeight(MaterialTheme.typography.labelMedium)
            val castCharacter = lineHeight(MaterialTheme.typography.labelSmall)
            val castHeight: (Boolean) -> Dp = { compact ->
                val portrait = if (compact) TvCastPortraitCompact else TvCastPortraitFull
                castHeading + 4.dp + portrait + 4.dp + castName + (if (compact) 0.dp else castCharacter + 2.dp) + 14.dp
            }
            val budget = tvShowDetailBudget(
                // One dp per block of slack for the borders and paddings a style height cannot see.
                band = (maxHeight - 12.dp).coerceAtLeast(0.dp),
                titleHeight = with(density) {
                    measurer.measure(
                        AnnotatedString(title), style = titleStyle, maxLines = 2,
                        constraints = Constraints(maxWidth = maxWidth.roundToPx()),
                    ).size.height.toDp()
                },
                factsHeight = if (facts.isBlank()) 0.dp else lineHeight(MaterialTheme.typography.bodyMedium) + 6.dp,
                // Play and the seasons keep a 48dp remote target whatever the type scale.
                controlsHeight = 48.dp,
                storyHeight = ::storyBlockHeight,
                railHeight = { cardWidth -> cardWidth * 9 / 16 + 12.dp },
                castHeight = castHeight,
                gap = gap,
                storyLinesShown = if (story.isBlank()) 0 else 3,
            )
            Column(Modifier.fillMaxSize().verticalScroll(pageScroll).focusGroup()) {
                TvShowFold(
                    band = band,
                    cast = { VideoCastRow(details, artwork, auth, compact = budget.compactCast, focusUp = railFocus) },
                    above = {
                        Column(verticalArrangement = Arrangement.spacedBy(gap)) {
                            Text(
                                title,
                                style = titleStyle,
                                fontWeight = FontWeight.SemiBold,
                                color = MaterialTheme.colorScheme.onSurface,
                                maxLines = 2,
                                overflow = TextOverflow.Ellipsis,
                            )
                            if (budget.showFacts) TvTitleFacts(facts, details)
                            TvSynopsis(
                                story = story,
                                maxLines = budget.storyLines,
                                style = storyStyle,
                                // Measured up front, not inferred after layout: the block only becomes a focus stop when
                                // there is more to read than the page can show.
                                expandable = budget.showReadMore && storyLinesNeeded > budget.storyLines,
                                onOpen = { synopsisOpen = true },
                                readMoreFocus = synopsisFocus,
                            )
                            Row(Modifier.padding(start = 8.dp), horizontalArrangement = Arrangement.spacedBy(10.dp), verticalAlignment = Alignment.CenterVertically) {
                                target?.let { next ->
                                    TvPlayPill(
                                        label = if (next.resumeSeconds > 0) "Resume \u00b7 S${next.season}E${next.episode}" else if (state.items.any { it.finished }) "Next \u00b7 S${next.season}E${next.episode}" else "Play",
                                        onClick = { onPlay(next) },
                                        modifier = Modifier.then(LocalTvContentFocus.current?.let { Modifier.focusRequester(it) } ?: Modifier),
                                        compact = true,
                                    )
                                }
                                // A lazy row, not a plain row: on a 20-season show a plain row laid the later pills out past
                                // the screen edge, where the remote could still land on them but nothing scrolled, so those
                                // seasons were unreachable. A LazyRow keeps its items scrollable and follows the focus.
                                if (show.seasons.size > 1) LazyRow(
                                    Modifier.weight(1f),
                                    state = seasonsState,
                                    contentPadding = PaddingValues(end = 24.dp),
                                    horizontalArrangement = Arrangement.spacedBy(10.dp),
                                ) {
                                    items(show.seasons, key = { "season-$it" }) { season ->
                                        TvPill(tvSeasonLabel(season), selected = state.selectedSeason == season, onClick = { onSeason(season) }, compact = true)
                                    }
                                }
                            }
                            LazyRow(
                                Modifier.fillMaxWidth().onFocusChanged {
                                    if (!it.hasFocus) {
                                        if (focusedEpisodeId != null) railOffset = pageScroll.value
                                        focusedEpisodeId = null
                                    }
                                }.focusRequester(railFocus).focusRestorer().focusGroup(),
                                state = railState,
                                contentPadding = PaddingValues(start = 8.dp, end = 24.dp, top = 6.dp, bottom = 6.dp),
                                horizontalArrangement = Arrangement.spacedBy(14.dp),
                            ) {
                                items(state.items, key = VideoItem::id, contentType = { "tv-episode" }) { episode ->
                                    var menu by remember(episode.id) { mutableStateOf(false) }
                                    Box(Modifier.width(budget.cardWidth)) {
                                        TvMediaCard(
                                            tvCleanTitle(episode.displayTitle), tvEpisodeLabel(episode), artwork(episode.artworkPath), auth, TvArtworkRatio.Landscape,
                                            { onPlay(episode) },
                                            progress = episode.progressFraction, completed = episode.finished,
                                            onLongClick = { menu = true },
                                        ) { focusedEpisodeId = episode.id; onFocus(TvFocusDetail(episode.displayTitle, tvEpisodeLabel(episode))) }
                                        TvContextDropdownMenu(expanded = menu, onDismissRequest = { menu = false }) {
                                            DropdownMenuItem(
                                                text = { Text(if (episode.finished) "Mark as unwatched" else "Mark as watched") },
                                                onClick = { menu = false; onSetWatched(episode, !episode.finished) },
                                                modifier = Modifier.tvFocusBorder(),
                                            )
                                        }
                                    }
                                }
                            }
                        }
                    },
                )
            }
        }
        if (synopsisOpen) {
            // An in-screen overlay rather than a Dialog: on a TV the focus system belongs to one window, so Back,
            // D-pad and the return to Read more all behave like any other screen.
            androidx.activity.compose.BackHandler { synopsisOpen = false }
            TvSynopsisOverlay(
                title = title,
                story = story,
                onDismiss = {
                    synopsisOpen = false
                    runCatching { synopsisFocus.requestFocus() }
                },
            )
        }
    }
}

/**
 * Keeps Cast atomic at the fold the way the film page does: when the heading and the portraits fit under the episode
 * rail they follow it directly; when they do not, the whole section starts after the first viewport and the page scrolls
 * to it as one block, rather than leaving a heading with its portraits cut by the bottom of the screen.
 */
@Composable
internal fun TvShowFold(band: Dp, above: @Composable () -> Unit, cast: @Composable () -> Unit) {
    Layout(content = { Box { above() }; Box { cast() } }) { children, constraints ->
        val loose = constraints.copy(minWidth = 0, minHeight = 0)
        val head = children[0].measure(loose)
        val actors = children[1].measure(loose)
        val viewport = band.roundToPx()
        val next = head.height + 12.dp.roundToPx()
        val castTop = if (next + actors.height <= viewport) next else viewport
        layout(constraints.maxWidth, castTop + actors.height) {
            head.placeRelative(0, 0)
            actors.placeRelative(0, castTop)
        }
    }
}

@Composable
internal fun TvPlayPill(label: String, onClick: () -> Unit, modifier: Modifier = Modifier, compact: Boolean = false) {
    if (compact) {
        TvCompactPill(label, onClick, modifier, play = true)
        return
    }
    var focused by remember { mutableStateOf(false) }
    Row(
        modifier.height(48.dp)
            .onFocusChanged { focused = it.isFocused }
            .tvFocusVisual(focused, CircleShape, focusedScale = 1.06f)
            .background(MaterialTheme.colorScheme.primary, CircleShape)
            .combinedClickable(interactionSource = remember { MutableInteractionSource() }, indication = null, role = Role.Button, onClick = onClick)
            .padding(horizontal = 22.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        Icon(Icons.Default.PlayArrow, null, Modifier.size(26.dp), tint = MaterialTheme.colorScheme.onPrimary)
        Text(label, style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.SemiBold, color = MaterialTheme.colorScheme.onPrimary, maxLines = 1)
    }
}

/** A choice chip: filled when it is the current one, outlined by the focus ring when the remote is on it. */
@Composable
internal fun TvPill(label: String, selected: Boolean, onClick: () -> Unit, modifier: Modifier = Modifier, compact: Boolean = false) {
    if (compact) {
        TvCompactPill(label, onClick, modifier, selected = selected)
        return
    }
    var focused by remember { mutableStateOf(false) }
    val shape = CircleShape
    Box(
        modifier.height(40.dp)
            .onFocusChanged { focused = it.isFocused }
            .tvFocusVisual(focused, shape, focusedScale = 1.05f)
            .background(if (selected) MaterialTheme.colorScheme.primaryContainer else MaterialTheme.colorScheme.surface, shape)
            .combinedClickable(interactionSource = remember { MutableInteractionSource() }, indication = null, role = Role.Tab, onClick = onClick)
            .padding(horizontal = 20.dp),
        contentAlignment = Alignment.Center,
    ) {
        Text(label, style = MaterialTheme.typography.titleSmall, fontWeight = if (selected) FontWeight.SemiBold else FontWeight.Medium, color = if (selected) MaterialTheme.colorScheme.onPrimaryContainer else MaterialTheme.colorScheme.onSurface, maxLines = 1)
    }
}

@Composable
private fun TvCompactPill(label: String, onClick: () -> Unit, modifier: Modifier = Modifier, play: Boolean = false, selected: Boolean = false, tab: Boolean = true) {
    var focused by remember { mutableStateOf(false) }
    Box(
        // 48dp stays the remote target; only the visible pill inside it shrinks.
        modifier.heightIn(min = 48.dp).widthIn(min = 48.dp).onFocusChanged { focused = it.isFocused }
            .combinedClickable(interactionSource = remember { MutableInteractionSource() }, indication = null, role = if (play || !tab) Role.Button else Role.Tab, onClick = onClick),
        contentAlignment = Alignment.Center,
    ) {
        Row(
            Modifier.testTag("compact-$label").tvFocusVisual(focused, CircleShape, focusedScale = 1.05f)
                .background(if (play) MaterialTheme.colorScheme.primary else if (selected) MaterialTheme.colorScheme.primaryContainer else MaterialTheme.colorScheme.surface, CircleShape)
                .heightIn(min = 32.dp).padding(horizontal = 12.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(5.dp),
        ) {
            if (play) Icon(Icons.Default.PlayArrow, null, Modifier.size(18.dp), tint = MaterialTheme.colorScheme.onPrimary)
            Text(label, style = MaterialTheme.typography.titleSmall.copy(fontSize = 16.sp), fontWeight = FontWeight.SemiBold,
                color = if (play) MaterialTheme.colorScheme.onPrimary else if (selected) MaterialTheme.colorScheme.onPrimaryContainer else MaterialTheme.colorScheme.onSurface,
                maxLines = 1)
        }
    }
}

internal fun tvMovieMeta(item: VideoItem): String = listOfNotNull(
    item.year.takeIf { it > 0 }?.toString(),
    item.durationSeconds.takeIf { it > 0 }?.let { seconds ->
        val minutes = (seconds / 60).toInt().coerceAtLeast(1)
        if (minutes >= 60) "${minutes / 60} h ${minutes % 60} min" else "$minutes min"
    },
    "Watched".takeIf { item.finished },
).joinToString(" · ")

/** A movie opened on TV: its backdrop behind a compact title and story, with Play (or Resume) and a watched toggle. */
@Composable
internal fun TvMovieDetail(
    item: VideoItem,
    auth: String?,
    art: String,
    backdrop: String,
    onPlay: () -> Unit,
    onSetWatched: (Boolean) -> Unit,
    modifier: Modifier = Modifier,
    details: VideoTitleDetails? = null,
    onDetails: () -> Unit = {},
    artwork: (String) -> String = { it },
) {
    LaunchedEffect(item.id) { onDetails() }
    TvBackdropEffect(backdrop.ifBlank { art })
    val pageScroll = rememberScrollState()
    var controlsFocused by remember { mutableStateOf(false) }
    LaunchedEffect(controlsFocused) { if (controlsFocused) pageScroll.animateScrollTo(0) }
    var synopsisOpen by remember(item.id) { mutableStateOf(false) }
    val storyFocus = remember { FocusRequester() }
    val title = tvCleanTitle(item.displayTitle)
    Box(modifier.fillMaxSize()) {
    BoxWithConstraints(Modifier.fillMaxSize().padding(start = 20.dp, end = 44.dp, top = 64.dp, bottom = 20.dp)) {
        val viewportHeight = maxHeight
        val showPoster = maxWidth >= 700.dp && maxHeight >= 300.dp && art.isNotBlank()
        Column(Modifier.fillMaxSize().verticalScroll(pageScroll)) {
            TvDetailPage(viewportHeight, hero = {
                Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(28.dp)) {
                    if (showPoster) AuthenticatedArtwork(art, auth, title, Modifier.width(170.dp).aspectRatio(2f / 3f), cornerRadius = 10.dp)
                    TvDetailHero(
                        title = title,
                        story = item.overview,
                        modifier = Modifier.weight(1f),
                        // A synopsis longer than the hero's three lines opens the whole text, as on the show page.
                        onStoryOpen = { synopsisOpen = true },
                        storyFocus = storyFocus,
                        facts = { TvTitleFacts(tvFactsLine(item.year, details).ifBlank { tvMovieMeta(item) }, details) },
                        controls = {
                            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                                if (item.progressFraction > 0f && !item.finished) TvProgressBar(item.progressFraction, Modifier.width(280.dp), 5.dp)
                                Row(Modifier.onFocusChanged { controlsFocused = it.hasFocus }, horizontalArrangement = Arrangement.spacedBy(12.dp), verticalAlignment = Alignment.CenterVertically) {
                                    TvPlayPill(
                                        label = if (item.resumeSeconds > 0 && !item.finished) "Resume" else "Play",
                                        onClick = onPlay,
                                        modifier = Modifier.then(LocalTvContentFocus.current?.let { Modifier.focusRequester(it) } ?: Modifier),
                                        compact = true,
                                    )
                                    TvCompactPill(
                                        label = if (item.finished) "Mark unwatched" else "Mark watched",
                                        selected = item.finished,
                                        tab = false,
                                        onClick = { onSetWatched(!item.finished) },
                                    )
                                }
                            }
                        },
                    )
                }
            }, cast = { VideoCastRow(details, artwork, auth) })
        }
    }
        if (synopsisOpen) {
            androidx.activity.compose.BackHandler { synopsisOpen = false }
            TvSynopsisOverlay(
                title = title,
                story = item.overview,
                onDismiss = {
                    synopsisOpen = false
                    runCatching { storyFocus.requestFocus() }
                },
            )
        }
    }
}
