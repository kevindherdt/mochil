package net.kdhmedia.mochil.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.combinedClickable
import androidx.compose.foundation.focusable
import androidx.compose.foundation.focusGroup
import androidx.compose.foundation.gestures.detectTapGestures
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.focus.onFocusChanged
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.input.key.KeyEventType
import androidx.compose.ui.input.key.onPreviewKeyEvent
import androidx.compose.ui.input.key.type
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp

/**
 * The description, clamped to the lines the page can spare. It reserves those lines even when the text is shorter, so
 * focusing another episode does not move the rail below it. When the whole text does not fit, the block itself becomes
 * one focus stop: the remote's Enter on the description opens the whole synopsis, and "Read more" says so.
 */
@Composable
internal fun TvSynopsis(
    story: String,
    maxLines: Int,
    style: TextStyle,
    expandable: Boolean,
    onOpen: () -> Unit,
    modifier: Modifier = Modifier,
    readMoreFocus: FocusRequester? = null,
) {
    if (story.isBlank() || maxLines <= 0) return
    var focused by remember { mutableStateOf(false) }
    Column(
        modifier.fillMaxWidth()
            .then(
                if (!expandable) Modifier else Modifier
                    .onFocusChanged { focused = it.isFocused }
                    .tvFocusVisual(focused, RoundedCornerShape(10.dp), focusedScale = 1.01f)
                    .combinedClickable(
                        interactionSource = remember { MutableInteractionSource() },
                        indication = null,
                        role = Role.Button,
                        onClickLabel = "Read the full description",
                        onClick = onOpen,
                    )
                    .then(readMoreFocus?.let { Modifier.focusRequester(it) } ?: Modifier),
            )
            .padding(horizontal = 4.dp, vertical = 2.dp),
        verticalArrangement = Arrangement.spacedBy(2.dp),
    ) {
        Text(
            story,
            modifier = Modifier.fillMaxWidth(),
            style = style,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            minLines = maxLines,
            maxLines = maxLines,
            overflow = TextOverflow.Ellipsis,
        )
        if (expandable) Text(
            "Read more",
            style = MaterialTheme.typography.labelMedium,
            fontWeight = FontWeight.SemiBold,
            color = MaterialTheme.colorScheme.primary,
            maxLines = 1,
        )
    }
}

/** The whole description over the page, so a long synopsis can always be read to the end. */
@Composable
internal fun TvSynopsisOverlay(title: String, story: String, onDismiss: () -> Unit) {
    val closeFocus = remember { FocusRequester() }
    val storyFocus = remember { FocusRequester() }
    val scroll = rememberScrollState()
    val step = with(LocalDensity.current) { 140.dp.toPx() }
    var closeFocused by remember { mutableStateOf(false) }
    // Focus the text, not Close: the key press that opened this overlay must not also dismiss it.
    LaunchedEffect(Unit) { runCatching { storyFocus.requestFocus() } }
    Box(
        Modifier.fillMaxSize()
            .background(Color.Black.copy(alpha = 0.62f))
            // The overlay covers the page but lives in the same window, so its stop sits next to the page's own
            // focusables: an arrow it did not answer walked the remote into Play or an episode underneath while the text
            // was still open, which left Close unreachable. Every direction key is answered and consumed here.
            .onPreviewKeyEvent { event ->
                if (event.type != KeyEventType.KeyDown) return@onPreviewKeyEvent false
                val action = tvRemoteAction(event.nativeKeyEvent.keyCode)
                if (action !in listOf(TvRemoteAction.Up, TvRemoteAction.Down, TvRemoteAction.Left, TvRemoteAction.Right)) {
                    return@onPreviewKeyEvent false
                }
                when (tvSynopsisMove(action!!, closeFocused, scroll.value > 0f, scroll.value < scroll.maxValue)) {
                    TvSynopsisMove.ScrollBack -> scroll.dispatchRawDelta(-step)
                    TvSynopsisMove.ScrollForward -> scroll.dispatchRawDelta(step)
                    TvSynopsisMove.ToClose -> runCatching { closeFocus.requestFocus() }
                    TvSynopsisMove.ToStory -> runCatching { storyFocus.requestFocus() }
                    null -> Unit
                }
                true
            }
            .pointerInput(Unit) { detectTapGestures { onDismiss() } },
        contentAlignment = Alignment.Center,
    ) {
        Surface(shape = RoundedCornerShape(18.dp), color = MaterialTheme.colorScheme.surface, tonalElevation = 6.dp) {
            Column(
                Modifier.widthIn(max = 900.dp).fillMaxWidth(0.72f).fillMaxHeight(0.82f).padding(28.dp),
                verticalArrangement = Arrangement.spacedBy(14.dp),
            ) {
                Text(title, style = MaterialTheme.typography.headlineSmall, fontWeight = FontWeight.SemiBold, maxLines = 2, overflow = TextOverflow.Ellipsis)
                Column(Modifier.weight(1f).fillMaxWidth().verticalScroll(scroll).focusRequester(storyFocus).focusable().focusGroup().testTag("tv-synopsis-text")) {
                    Text(story, style = tvDetailStoryStyle(MaterialTheme.typography.bodyMedium))
                }
                Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.End) {
                    TvPill(
                        "Close",
                        selected = false,
                        onClick = onDismiss,
                        modifier = Modifier.focusRequester(closeFocus)
                            .onFocusChanged { closeFocused = it.isFocused }
                            .testTag("tv-synopsis-close"),
                        compact = true,
                    )
                }
            }
        }
    }
}
