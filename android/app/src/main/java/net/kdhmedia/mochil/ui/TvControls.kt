package net.kdhmedia.mochil.ui

import androidx.compose.foundation.ScrollState
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.clickable
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.DropdownMenu
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.animation.core.FastOutSlowInEasing
import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.animation.core.snap
import androidx.compose.animation.core.tween
import androidx.compose.ui.graphics.graphicsLayer
import net.kdhmedia.mochil.ui.theme.LocalReduceMotion
import androidx.compose.ui.focus.onFocusChanged
import androidx.compose.ui.graphics.Shape
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.input.key.KeyEventType
import androidx.compose.ui.input.key.onPreviewKeyEvent
import androidx.compose.ui.input.key.type
import androidx.compose.ui.semantics.selected
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import androidx.compose.ui.zIndex
import android.view.KeyEvent

internal enum class TvHoldResult { Pass, Trigger, Consume }

/** Shared spatial-focus motion for TV cards; large enough to read at distance without reflow. */
internal const val TV_CARD_FOCUS_SCALE = 1.035f

/** Avoid graphics/draw modifier nodes on the many idle TV controls. */
@Composable
internal fun Modifier.tvFocusVisual(
    focused: Boolean,
    shape: Shape,
    selected: Boolean = false,
    focusedScale: Float = TV_CARD_FOCUS_SCALE,
): Modifier {
    // The grow-and-settle is what makes moving across a shelf feel connected; it is skipped for reduced motion.
    val reduceMotion = LocalReduceMotion.current
    val scale by animateFloatAsState(
        if (focused) focusedScale else 1f,
        if (reduceMotion) snap() else tween(durationMillis = 150, easing = FastOutSlowInEasing),
        label = "tvFocusScale",
    )
    return this
    .then(if (scale != 1f) Modifier.graphicsLayer { scaleX = scale; scaleY = scale }.zIndex(if (focused) 1f else .5f) else Modifier)
    .then(
        when {
            // A faint outer halo before the solid ring: at three metres a 3dp line alone is easy
            // to lose against busy artwork. Static, so it survives reduce-motion.
            focused -> Modifier
                .border(8.dp, MaterialTheme.colorScheme.primary.copy(alpha = .22f), shape)
                .border(3.dp, MaterialTheme.colorScheme.primary, shape)
            selected -> Modifier.border(2.dp, MaterialTheme.colorScheme.primary, shape)
            else -> Modifier
        },
    )
}

internal class TvSelectHoldState {
    private var triggered = false

    fun handle(select: Boolean, action: Int, repeatCount: Int, longPress: Boolean): TvHoldResult {
        if (!select) return TvHoldResult.Pass
        // A fresh press starts over, so a release the popup took never leaves the next click swallowed.
        if (action == KeyEvent.ACTION_DOWN && !longPress && repeatCount == 0) triggered = false
        if (action == KeyEvent.ACTION_DOWN && (longPress || repeatCount > 0)) {
            if (triggered) return TvHoldResult.Consume
            triggered = true
            return TvHoldResult.Trigger
        }
        if (action == KeyEvent.ACTION_UP && triggered) {
            triggered = false
            return TvHoldResult.Consume
        }
        return TvHoldResult.Pass
    }
}

internal class TvContextMenuInputGuard {
    private var awaitingOpeningRelease = true

    fun consume(select: Boolean, action: Int): Boolean {
        if (!awaitingOpeningRelease || !select) return false
        if (action == KeyEvent.ACTION_UP) awaitingOpeningRelease = false
        return true
    }
}

@Composable
internal fun TvContextDropdownMenu(expanded: Boolean, onDismissRequest: () -> Unit, content: @Composable ColumnScope.() -> Unit) {
    val guard = remember(expanded) { TvContextMenuInputGuard() }
    DropdownMenu(
        expanded = expanded,
        onDismissRequest = onDismissRequest,
        modifier = Modifier.onPreviewKeyEvent { event ->
            val native = event.nativeKeyEvent
            guard.consume(tvRemoteAction(native.keyCode) == TvRemoteAction.Select, native.action)
        },
        content = content,
    )
}

/**
 * Scroll a text panel with the remote instead of walking focus out of it: Up/Down scroll while there is somewhere to go
 * and are passed through at either end, so the remote can still leave the panel afterwards. Apply it to a focusable node
 * (the same one the user is meant to land on), because a preview handler only sees keys while focus is inside it.
 */
@Composable
internal fun Modifier.tvScrollKeys(state: ScrollState, step: Dp = 140.dp): Modifier {
    val pixels = with(LocalDensity.current) { step.toPx() }
    return onPreviewKeyEvent { event ->
        if (event.type != KeyEventType.KeyDown) return@onPreviewKeyEvent false
        when (tvRemoteAction(event.nativeKeyEvent.keyCode)) {
            TvRemoteAction.Up -> if (state.value > 0f) { state.dispatchRawDelta(-pixels); true } else false
            TvRemoteAction.Down -> if (state.value < state.maxValue) { state.dispatchRawDelta(pixels); true } else false
            else -> false
        }
    }
}

@Composable
internal fun Modifier.tvRemoteLongPress(enabled: Boolean = true, onLongPress: () -> Unit): Modifier {
    val hold = remember { TvSelectHoldState() }
    return if (!enabled) this else onPreviewKeyEvent { event ->
        val native = event.nativeKeyEvent
        when (hold.handle(tvRemoteAction(native.keyCode) == TvRemoteAction.Select, native.action, native.repeatCount, native.isLongPress)) {
            TvHoldResult.Pass -> false
            TvHoldResult.Trigger -> {
                // The card keeps swallowing this hold's release: if it arrives before the menu has focus, it must not count as a click and start playback.
                onLongPress()
                true
            }
            TvHoldResult.Consume -> true
        }
    }
}

/** A compact TV control whose focus position remains obvious from viewing distance. */
@Composable
internal fun TvIconButton(
    icon: ImageVector,
    contentDescription: String,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    active: Boolean = false,
    selectedState: Boolean? = null,
) {
    var focused by remember { mutableStateOf(false) }
    val shape = RoundedCornerShape(12.dp)
    IconButton(
        onClick = onClick,
        modifier = modifier.size(52.dp)
            .semantics { selectedState?.let { selected = it } }
            .onFocusChanged { focused = it.isFocused }
            .tvFocusVisual(focused, shape, focusedScale = 1.06f),
    ) {
        Icon(icon, contentDescription, Modifier.size(30.dp), tint = if (active) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurface)
    }
}

/** A text-valued transport control for state that cannot be communicated by an icon alone. */
@Composable
internal fun TvTextButton(label: String, onClick: () -> Unit, modifier: Modifier = Modifier, active: Boolean = false) {
    var focused by remember { mutableStateOf(false) }
    val shape = RoundedCornerShape(12.dp)
    TextButton(
        onClick = onClick,
        modifier = modifier.size(width = 64.dp, height = 52.dp).semantics { selected = active }
            .onFocusChanged { focused = it.isFocused }
            .tvFocusVisual(focused, shape, focusedScale = 1.06f),
    ) {
        Text(label, color = if (focused || active) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurface)
    }
}

@Composable
internal fun Modifier.tvFocusBorder(): Modifier {
    var focused by remember { mutableStateOf(false) }
    val shape = RoundedCornerShape(12.dp)
    return this.onFocusChanged { focused = it.isFocused }.tvFocusVisual(focused, shape, focusedScale = 1f)
}

/** The one TV action button: a rounded pill, accent-filled when primary, frosted when not, and lit up under the remote. */
@Composable
internal fun TvButton(text: String, onClick: () -> Unit, modifier: Modifier = Modifier, primary: Boolean = false, enabled: Boolean = true) {
    var focused by androidx.compose.runtime.remember { androidx.compose.runtime.mutableStateOf(false) }
    val shape = androidx.compose.foundation.shape.CircleShape
    val accent = androidx.compose.material3.MaterialTheme.colorScheme.primary
    val fill = when {
        primary -> if (focused) accent else accent.copy(alpha = .86f)
        else -> androidx.compose.ui.graphics.Color.White.copy(alpha = if (focused) .22f else .10f)
    }
    val ink = if (primary) androidx.compose.material3.MaterialTheme.colorScheme.onPrimary else androidx.compose.ui.graphics.Color.White
    androidx.compose.foundation.layout.Box(
        modifier.height(48.dp)
            .onFocusChanged { focused = it.isFocused }
            .tvFocusVisual(focused, shape, focusedScale = 1.05f)
            .background(fill, shape)
            .clickable(enabled = enabled, interactionSource = androidx.compose.runtime.remember { androidx.compose.foundation.interaction.MutableInteractionSource() }, indication = null, role = androidx.compose.ui.semantics.Role.Button, onClick = onClick)
            .padding(horizontal = 24.dp),
        contentAlignment = androidx.compose.ui.Alignment.Center,
    ) {
        androidx.compose.material3.Text(text, style = androidx.compose.material3.MaterialTheme.typography.titleMedium, fontWeight = androidx.compose.ui.text.font.FontWeight.SemiBold, color = ink.copy(alpha = if (enabled) 1f else .45f), maxLines = 1)
    }
}
