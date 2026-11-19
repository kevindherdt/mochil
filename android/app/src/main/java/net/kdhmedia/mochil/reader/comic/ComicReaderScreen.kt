package net.kdhmedia.mochil.reader.comic

import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.distinctUntilChanged
import kotlinx.coroutines.flow.filterNotNull
import kotlinx.coroutines.launch
import kotlinx.coroutines.supervisorScope
import kotlinx.coroutines.currentCoroutineContext
import kotlinx.coroutines.isActive
import androidx.compose.runtime.rememberUpdatedState
import kotlinx.coroutines.withContext
import kotlinx.coroutines.flow.Flow
import androidx.compose.ui.input.key.KeyEventType
import androidx.compose.ui.input.key.onPreviewKeyEvent
import androidx.compose.ui.input.key.type
import kotlinx.coroutines.flow.emptyFlow
import net.kdhmedia.mochil.reader.ReaderPageTurn
import net.kdhmedia.mochil.reader.FixedLayoutLocalProgressStore
import net.kdhmedia.mochil.reader.FixedLayoutReaderKind
import androidx.compose.foundation.background
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.foundation.gestures.detectTapGestures
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.pager.HorizontalPager
import androidx.compose.foundation.pager.rememberPagerState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.ArrowDropDown
import net.kdhmedia.mochil.ui.MotionVisibility
import net.kdhmedia.mochil.ui.theme.LocalReduceMotion
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Button
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.produceState
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.runtime.snapshotFlow
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import coil3.compose.AsyncImage
import coil3.request.CachePolicy
import coil3.request.ImageRequest

private fun directionKey(progressKey: String) = "right_to_left:$progressKey"

/**
 * Manga reads right-to-left and every other comic left-to-right, as the web does. A direction the reader
 * saved for this book wins, then the one saved before directions were per book; with no library kind
 * (a local, offline or WebDAV file) the old right-to-left default stays.
 */
internal fun comicReadsRightToLeft(preferences: android.content.SharedPreferences, progressKey: String, libraryKind: String?): Boolean {
    if (preferences.contains(directionKey(progressKey))) return preferences.getBoolean(directionKey(progressKey), true)
    if (preferences.contains("right_to_left")) return preferences.getBoolean("right_to_left", true)
    return libraryKind == null || libraryKind.trim().equals("manga", ignoreCase = true)
}

private enum class ComicMode { Paged, Webtoon }
internal enum class PageFit { Fit, Width }

@Composable
fun ComicReaderScreen(source: ComicSource, title: String, progressKey: String, requestedPage: Int? = null, onRequestedPageConsumed: () -> Unit = {}, onReady: (Int, Int) -> Unit = { _, _ -> }, onPage: (Int) -> Unit = {}, onClose: () -> Unit, onNextChapter: (() -> Unit)? = null, pageTurns: Flow<ReaderPageTurn>, controlRequests: Flow<Unit> = emptyFlow(), libraryKind: String? = null) {
    val context = LocalContext.current
    val preferences = remember { context.getSharedPreferences("comic_reader", 0) }
    val localProgress = remember { FixedLayoutLocalProgressStore(context, FixedLayoutReaderKind.Comic) }
    val archiveResult = remember(source) { runCatching { openComicPageSource(source) } }
    val archive = archiveResult.getOrNull()
    if (archive == null) {
        Box(Modifier.fillMaxSize().background(Color.Black), contentAlignment = Alignment.Center) {
            Text("This comic archive could not be opened.", color = Color.White)
        }
        return
    }
    DisposableEffect(archive) { onDispose(archive::close) }
    var mode by remember { mutableStateOf(runCatching { ComicMode.valueOf(preferences.getString("mode", ComicMode.Paged.name)!!) }.getOrDefault(ComicMode.Paged)) }
    var modeMenuExpanded by remember { mutableStateOf(false) }
    var rightToLeft by remember(progressKey, libraryKind) { mutableStateOf(comicReadsRightToLeft(preferences, progressKey, libraryKind)) }
    var pageFit by remember { mutableStateOf(runCatching { PageFit.valueOf(preferences.getString("page_fit", PageFit.Fit.name)!!) }.getOrDefault(PageFit.Fit)) }
    var fitMenuExpanded by remember { mutableStateOf(false) }
    var chromeVisible by remember { mutableStateOf(false) }
    val savedPage = remember(progressKey, archive) { localProgress.load(progressKey, archive.pageCount - 1) }
    var currentPage by remember(progressKey, archive) { mutableIntStateOf(savedPage) }
    var finished by remember { mutableStateOf(preferences.getBoolean("finished:$progressKey", false)) }
    LaunchedEffect(progressKey, archive.pageCount) { onReady(archive.pageCount, savedPage) }

    // A remote cannot tap: select/up/down ask for the controls, and every key
    // press while they show restarts the hide timer so focus is not pulled away.
    var keyActivity by remember { mutableIntStateOf(0) }
    LaunchedEffect(controlRequests) { controlRequests.collect { chromeVisible = true; keyActivity++ } }
    androidx.activity.compose.BackHandler(chromeVisible) { chromeVisible = false }
    LaunchedEffect(chromeVisible, modeMenuExpanded, fitMenuExpanded, keyActivity) {
        if (chromeVisible && !modeMenuExpanded && !fitMenuExpanded) {
            delay(4_000)
            chromeVisible = false
        }
    }

    Box(Modifier.fillMaxSize().background(Color.Black).onPreviewKeyEvent { if (it.type == KeyEventType.KeyDown) keyActivity++; false }) {
        if (archive.pageCount == 0) {
            Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) { Text("No supported images were found in this comic.") }
        } else if (mode == ComicMode.Paged) {
            PagedComic(archive, progressKey, currentPage, requestedPage, onRequestedPageConsumed, rightToLeft, pageFit, pageTurns, { chromeVisible = !chromeVisible }) {
                currentPage = it
                onPage(it)
                if (it == archive.pageCount - 1) {
                    finished = true
                    chromeVisible = true
                    preferences.edit().putBoolean("finished:$progressKey", true).apply()
                }
                localProgress.save(progressKey, it)
            }
        } else {
            WebtoonComic(archive, progressKey, currentPage, requestedPage, onRequestedPageConsumed, pageTurns, { chromeVisible = !chromeVisible }, onPage = {
                currentPage = it
                onPage(it)
                if (it == archive.pageCount - 1) {
                    finished = true
                    chromeVisible = true
                    preferences.edit().putBoolean("finished:$progressKey", true).apply()
                }
                localProgress.save(progressKey, it)
            })
        }
        MotionVisibility(chromeVisible, modifier = Modifier.align(Alignment.TopCenter)) {
            ReaderToolbar(
                title = title,
                mode = mode,
                rightToLeft = rightToLeft,
                pageFit = pageFit,
                modeMenuExpanded = modeMenuExpanded,
                fitMenuExpanded = fitMenuExpanded,
                onClose = onClose,
                onModeMenu = { modeMenuExpanded = it },
                onMode = {
                    mode = it
                    modeMenuExpanded = false
                    preferences.edit().putString("mode", it.name).apply()
                },
                onDirection = {
                    rightToLeft = it
                    preferences.edit().putBoolean(directionKey(progressKey), it).apply()
                },
                onFitMenu = { fitMenuExpanded = it },
                onFit = {
                    pageFit = it
                    fitMenuExpanded = false
                    preferences.edit().putString("page_fit", it.name).apply()
                },
            )
        }
        MotionVisibility(chromeVisible && archive.pageCount > 0, modifier = Modifier.align(Alignment.BottomCenter)) {
            Row(
                Modifier.navigationBarsPadding().padding(bottom = 10.dp).background(Color.Black.copy(alpha = 0.82f), RoundedCornerShape(20.dp)).padding(start = 14.dp, end = if (finished && onNextChapter != null) 6.dp else 14.dp, top = 5.dp, bottom = 5.dp),
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                Text(if (finished) "Finished · ${currentPage + 1} / ${archive.pageCount}" else "${currentPage + 1} / ${archive.pageCount}", color = Color.White, style = MaterialTheme.typography.labelLarge)
                if (finished && onNextChapter != null) Button(onClick = onNextChapter) { Text("Next chapter") }
            }
        }
    }
}

@Composable
private fun ReaderToolbar(
    title: String,
    mode: ComicMode,
    rightToLeft: Boolean,
    pageFit: PageFit,
    modeMenuExpanded: Boolean,
    fitMenuExpanded: Boolean,
    onClose: () -> Unit,
    onModeMenu: (Boolean) -> Unit,
    onMode: (ComicMode) -> Unit,
    onDirection: (Boolean) -> Unit,
    onFitMenu: (Boolean) -> Unit,
    onFit: (PageFit) -> Unit,
) {
    Row(
        Modifier.fillMaxWidth().background(Color.Black.copy(alpha = 0.82f)).statusBarsPadding().padding(horizontal = 4.dp, vertical = 4.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(2.dp),
    ) {
        IconButton(onClick = onClose) { Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Close reader", tint = Color.White) }
        Text(title, modifier = Modifier.weight(1f), color = Color.White, style = MaterialTheme.typography.titleSmall, maxLines = 1, overflow = TextOverflow.Ellipsis)
        if (mode == ComicMode.Paged) {
            TextButton(onClick = { onDirection(!rightToLeft) }) { Text(if (rightToLeft) "RTL" else "LTR", color = Color.White) }
            Box {
                TextButton(onClick = { onFitMenu(true) }) {
                    Text(pageFit.label, color = Color.White)
                    Icon(Icons.Default.ArrowDropDown, contentDescription = null, tint = Color.White)
                }
                DropdownMenu(expanded = fitMenuExpanded, onDismissRequest = { onFitMenu(false) }) {
                    PageFit.entries.forEach { option -> DropdownMenuItem(text = { Text(option.label) }, onClick = { onFit(option) }) }
                }
            }
        }
        Box {
            TextButton(onClick = { onModeMenu(true) }) {
                Text(if (mode == ComicMode.Paged) "Paged" else "Webtoon", color = Color.White)
                Icon(Icons.Default.ArrowDropDown, contentDescription = null, tint = Color.White)
            }
            DropdownMenu(expanded = modeMenuExpanded, onDismissRequest = { onModeMenu(false) }) {
                ComicMode.entries.forEach { option ->
                    DropdownMenuItem(text = { Text(if (option == ComicMode.Paged) "Paged" else "Webtoon") }, onClick = { onMode(option) })
                }
            }
        }
    }
}

@Composable
internal fun PagedComic(archive: ComicPageSource, cacheKeyPrefix: String, initialPage: Int, requestedPage: Int?, onRequestedPageConsumed: () -> Unit, rightToLeft: Boolean, pageFit: PageFit, pageTurns: Flow<ReaderPageTurn>, onTap: () -> Unit, pager: androidx.compose.foundation.pager.PagerState = rememberPagerState(initialPage = initialPage, pageCount = { archive.pageCount }), onPage: (Int) -> Unit) {
    val pendingRequest by rememberUpdatedState(requestedPage)
    val reportPage by rememberUpdatedState(onPage)
    LaunchedEffect(pager) {
        snapshotFlow { if (pendingRequest == null) pager.settledPage else null }
            .distinctUntilChanged().filterNotNull().collect { reportPage(it) }
    }
    LaunchedEffect(pager, pageTurns) { pageTurns.collect { turn ->
        val destination = (pager.currentPage + if (turn == ReaderPageTurn.Next) 1 else -1).coerceIn(0, archive.pageCount - 1)
        if (destination != pager.currentPage) supervisorScope { launch { pager.scrollToPage(destination) }.join() }
    } }
    LaunchedEffect(pager, requestedPage) { requestedPage?.let {
        val target = it.coerceIn(0, archive.pageCount - 1)
        retryRequestedScroll({ pager.scrollToPage(target) }, { pager.settledPage == target && !pager.isScrollInProgress })
        onRequestedPageConsumed()
    } }
    HorizontalPager(
        state = pager,
        modifier = Modifier.fillMaxSize().pointerInput(onTap) { detectTapGestures(onTap = { onTap() }) },
        reverseLayout = rightToLeft,
        beyondViewportPageCount = 1,
    ) { page ->
        if (pageFit == PageFit.Width) {
            Box(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
                ComicPage(archive, cacheKeyPrefix, page, Modifier.fillMaxWidth(), pageFit.contentScale)
            }
        } else {
            ComicPage(archive, cacheKeyPrefix, page, Modifier.fillMaxSize(), pageFit.contentScale)
        }
    }
}

@Composable
internal fun WebtoonComic(archive: ComicPageSource, cacheKeyPrefix: String, initialPage: Int, requestedPage: Int?, onRequestedPageConsumed: () -> Unit, pageTurns: Flow<ReaderPageTurn>, onTap: () -> Unit, onPage: (Int) -> Unit, list: androidx.compose.foundation.lazy.LazyListState = rememberLazyListState(initialFirstVisibleItemIndex = initialPage), pageContent: @Composable (Int, Modifier) -> Unit = { page, loading -> ComicPage(archive, cacheKeyPrefix, page, Modifier.fillMaxWidth(), ContentScale.FillWidth, loading) }) {
    val pendingRequest by rememberUpdatedState(requestedPage)
    val reportPage by rememberUpdatedState(onPage)
    LaunchedEffect(list) {
        snapshotFlow { if (pendingRequest == null) list.firstVisibleItemIndex else null }
            .distinctUntilChanged().filterNotNull().collect { reportPage(it) }
    }
    LaunchedEffect(list, requestedPage) {
        requestedPage?.let {
            val target = it.coerceIn(0, archive.pageCount - 1)
            retryRequestedScroll({ list.scrollToItem(target) }, { list.firstVisibleItemIndex == target && !list.isScrollInProgress })
            onRequestedPageConsumed()
        }
    }
    LaunchedEffect(list, pageTurns) { pageTurns.collect { turn ->
        val destination = (list.firstVisibleItemIndex + if (turn == ReaderPageTurn.Next) 1 else -1).coerceIn(0, archive.pageCount - 1)
        if (destination != list.firstVisibleItemIndex) {
            supervisorScope { launch { list.scrollToItem(destination) }.join() }
        }
    } }
    BoxWithConstraints(Modifier.fillMaxSize()) {
        val viewportHeight = maxHeight
        LazyColumn(Modifier.fillMaxSize().pointerInput(onTap) { detectTapGestures(onTap = { onTap() }) }, state = list) {
            items(archive.pageCount, key = { it }) { page ->
                // A short final image must still be scrollable to its own index.
                if (page == archive.pageCount - 1) {
                    Box(Modifier.fillMaxWidth().heightIn(min = viewportHeight)) {
                        pageContent(page, Modifier.fillParentMaxHeight())
                    }
                } else {
                    pageContent(page, Modifier.fillParentMaxHeight())
                }
            }
        }
    }
}

// Gesture mutations may cancel an attempt without cancelling its owning effect.
// Replacement/disposal cancels the owner and therefore every retry.
private suspend fun retryRequestedScroll(scroll: suspend () -> Unit, reached: () -> Boolean) {
    while (currentCoroutineContext().isActive) {
        var completed = false
        supervisorScope { launch { scroll(); completed = true }.join() }
        if (completed && reached()) return
        delay(50)
    }
}

private val PageFit.label: String get() = when (this) {
    PageFit.Fit -> "Fit"
    PageFit.Width -> "Width"
}

private val PageFit.contentScale: ContentScale get() = when (this) {
    PageFit.Fit -> ContentScale.Fit
    PageFit.Width -> ContentScale.FillWidth
}

@Composable
private fun ComicPage(archive: ComicPageSource, cacheKeyPrefix: String, page: Int, modifier: Modifier, scale: ContentScale, loadingModifier: Modifier = Modifier) {
    var retryToken by remember(archive, page) { mutableIntStateOf(0) }
    var imageLoaded by remember(archive, page, retryToken) { mutableStateOf(false) }
    val result by produceState<Result<ByteArray>?>(null, archive, page, retryToken) {
        value = withContext(Dispatchers.IO) { runCatching { archive.page(page) } }
    }
    val bytes = result?.getOrNull()
    if (result == null) {
        Box(modifier.then(loadingModifier), contentAlignment = Alignment.Center) {
            if (LocalReduceMotion.current) CircularProgressIndicator(progress = { 1f }) else CircularProgressIndicator()
        }
    } else if (bytes == null) {
        Box(modifier.then(loadingModifier), contentAlignment = Alignment.Center) {
            Column(horizontalAlignment = Alignment.CenterHorizontally) {
                Text("This page could not be loaded.", color = Color.White)
                TextButton(onClick = { retryToken++ }) { Text("Retry") }
            }
        }
    } else {
        val context = LocalContext.current
        // Reusing a stable ImageRequest (keyed by the decoded bytes) lets Coil's own bounded
        // memory cache serve repeat compositions of this page without re-decoding a bitmap;
        // the key is scoped to this book + page so different comics never collide, and the
        // disk cache stays off since these bytes are already local and already cached above.
        val request = remember(bytes) {
            ImageRequest.Builder(context)
                .data(bytes)
                .memoryCacheKey("comic:$cacheKeyPrefix:$page")
                .memoryCachePolicy(CachePolicy.ENABLED)
                .diskCachePolicy(CachePolicy.DISABLED)
                .build()
        }
        AsyncImage(
            model = request,
            contentDescription = "Page ${page + 1}",
            contentScale = scale,
            onSuccess = { imageLoaded = true },
            modifier = modifier.then(if (imageLoaded) Modifier else loadingModifier),
        )
    }
}
