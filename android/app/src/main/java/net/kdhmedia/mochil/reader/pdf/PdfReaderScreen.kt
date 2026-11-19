package net.kdhmedia.mochil.reader.pdf

import android.graphics.Bitmap
import net.kdhmedia.mochil.ui.MotionVisibility
import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.gestures.detectTapGestures
import androidx.compose.foundation.gestures.rememberTransformableState
import androidx.compose.foundation.gestures.transformable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.pager.HorizontalPager
import androidx.compose.foundation.pager.rememberPagerState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.automirrored.filled.OpenInNew
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import net.kdhmedia.mochil.ui.theme.LocalReduceMotion
import androidx.compose.material3.Slider
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableFloatStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.runtime.snapshotFlow
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.asImageBitmap
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import java.io.File
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.distinctUntilChanged
import kotlinx.coroutines.flow.Flow
import androidx.compose.ui.input.key.KeyEventType
import androidx.compose.ui.input.key.onPreviewKeyEvent
import androidx.compose.ui.input.key.type
import kotlinx.coroutines.flow.emptyFlow
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import net.kdhmedia.mochil.reader.ReaderPageTurn
import net.kdhmedia.mochil.reader.FixedLayoutLocalProgressStore
import net.kdhmedia.mochil.reader.FixedLayoutReaderKind

@Composable
fun PdfReaderScreen(
    file: File,
    title: String,
    progressKey: String,
    requestedPage: Int?,
    onRequestedPageConsumed: () -> Unit,
    onReady: (Int, Int) -> Unit,
    onPage: (Int) -> Unit,
    onClose: () -> Unit,
    onOpenExternal: () -> Unit,
    pageTurns: Flow<ReaderPageTurn>,
    controlRequests: Flow<Unit> = emptyFlow(),
) {
    val context = LocalContext.current
    val localProgress = remember { FixedLayoutLocalProgressStore(context, FixedLayoutReaderKind.Pdf) }
    val documentResult = remember(file.path) { runCatching { PdfDocument(file) } }
    val document = documentResult.getOrNull()
    if (document == null || document.pageCount == 0) {
        Box(Modifier.fillMaxSize().background(Color(0xFF171513)), contentAlignment = Alignment.Center) {
            Text("This PDF could not be opened.", color = Color.White)
        }
        return
    }
    DisposableEffect(document) { onDispose(document::close) }
    val savedPage = localProgress.load(progressKey, document.pageCount - 1)
    val pager = rememberPagerState(initialPage = savedPage, pageCount = { document.pageCount })
    val scope = rememberCoroutineScope()
    var chromeVisible by remember { mutableStateOf(false) }
    var draggedPage by remember { mutableFloatStateOf(-1f) }

    LaunchedEffect(document, savedPage) { onReady(document.pageCount, savedPage) }
    LaunchedEffect(requestedPage) {
        requestedPage?.takeIf { it in 0 until document.pageCount }?.let { page ->
            if (pager.currentPage != page) pager.scrollToPage(page)
            onRequestedPageConsumed()
        }
    }
    LaunchedEffect(pager, progressKey) {
        snapshotFlow { pager.settledPage }.distinctUntilChanged().collect { page ->
            localProgress.save(progressKey, page)
            onPage(page)
        }
    }
    LaunchedEffect(pager, pageTurns) {
        pageTurns.collect { turn ->
            val destination = (pager.currentPage + if (turn == ReaderPageTurn.Next) 1 else -1).coerceIn(0, document.pageCount - 1)
            if (destination != pager.currentPage) pager.scrollToPage(destination)
        }
    }
    var keyActivity by remember { mutableIntStateOf(0) }
    LaunchedEffect(controlRequests) { controlRequests.collect { chromeVisible = true; keyActivity++ } }
    androidx.activity.compose.BackHandler(chromeVisible) { chromeVisible = false }
    LaunchedEffect(chromeVisible, keyActivity) {
        if (chromeVisible) {
            delay(4_000)
            chromeVisible = false
        }
    }

    Box(Modifier.fillMaxSize().background(Color(0xFF171513)).onPreviewKeyEvent { if (it.type == KeyEventType.KeyDown) keyActivity++; false }) {
        HorizontalPager(
            state = pager,
            beyondViewportPageCount = 1,
            modifier = Modifier.fillMaxSize().pointerInput(Unit) { detectTapGestures(onTap = { chromeVisible = !chromeVisible }) },
        ) { page -> PdfPage(document, page) }
        MotionVisibility(chromeVisible, modifier = Modifier.align(Alignment.TopCenter)) {
            Row(
                Modifier.fillMaxWidth().background(Color(0xE6171513)).statusBarsPadding().padding(horizontal = 4.dp, vertical = 4.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                IconButton(onClick = onClose) { Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Close PDF", tint = Color.White) }
                Text(title, Modifier.weight(1f), color = Color.White, style = MaterialTheme.typography.titleSmall, maxLines = 1, overflow = TextOverflow.Ellipsis)
                IconButton(onClick = onOpenExternal) { Icon(Icons.AutoMirrored.Filled.OpenInNew, contentDescription = "Open PDF in another app", tint = Color.White) }
            }
        }
        MotionVisibility(chromeVisible, modifier = Modifier.align(Alignment.BottomCenter)) {
            Column(
                Modifier.fillMaxWidth().navigationBarsPadding().padding(horizontal = 16.dp, vertical = 10.dp).background(Color(0xE6171513), RoundedCornerShape(14.dp)).padding(horizontal = 14.dp, vertical = 8.dp),
                horizontalAlignment = Alignment.CenterHorizontally,
            ) {
                val shownPage = if (draggedPage >= 0f) draggedPage.toInt() else pager.currentPage
                Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                    Text("${shownPage + 1} / ${document.pageCount}", color = Color.White, style = MaterialTheme.typography.labelLarge)
                    Text("Pinch to zoom", color = Color.White.copy(alpha = .72f), style = MaterialTheme.typography.labelMedium)
                }
                if (document.pageCount > 1) Slider(
                    value = shownPage.toFloat(),
                    onValueChange = { draggedPage = it },
                    onValueChangeFinished = {
                        val destination = draggedPage.toInt().coerceIn(0, document.pageCount - 1)
                        draggedPage = -1f
                        scope.launch { pager.scrollToPage(destination) }
                    },
                    valueRange = 0f..(document.pageCount - 1).toFloat(),
                    steps = (document.pageCount - 2).coerceIn(0, 100),
                )
            }
        }
    }
}

@Composable
private fun PdfPage(document: PdfDocument, page: Int) {
    val targetWidth = androidx.compose.ui.platform.LocalWindowInfo.current.containerSize.width
    var bitmap by remember(document, page, targetWidth) { mutableStateOf<Bitmap?>(null) }
    var scale by remember(document, page) { mutableFloatStateOf(1f) }
    var offset by remember(document, page) { mutableStateOf(Offset.Zero) }
    val transformState = rememberTransformableState { _, zoom, pan, _ ->
        val nextScale = (scale * zoom).coerceIn(1f, 4f)
        scale = nextScale
        offset = if (nextScale == 1f) Offset.Zero else offset + pan
    }
    LaunchedEffect(document, page, targetWidth) {
        bitmap = withContext(Dispatchers.IO) { runCatching { document.render(page, targetWidth) }.getOrNull() }
    }
    DisposableEffect(document, page, targetWidth) { onDispose { bitmap?.recycle() } }
    Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
        bitmap?.let {
            Image(
                it.asImageBitmap(),
                contentDescription = "Page ${page + 1}",
                modifier = Modifier.fillMaxSize()
                    .graphicsLayer(scaleX = scale, scaleY = scale, translationX = offset.x, translationY = offset.y)
                    .transformable(state = transformState, canPan = { scale > 1f }),
            )
        }
            ?: if (LocalReduceMotion.current) {
                CircularProgressIndicator(progress = { 1f }, color = Color(0xFFF09A72))
            } else {
                CircularProgressIndicator(color = Color(0xFFF09A72))
            }
    }
}
