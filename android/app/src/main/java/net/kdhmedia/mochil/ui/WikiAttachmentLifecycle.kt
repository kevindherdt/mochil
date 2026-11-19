package net.kdhmedia.mochil.ui

import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.rememberUpdatedState
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleEventObserver
import androidx.lifecycle.compose.LocalLifecycleOwner

/** Do not open a late attachment after leaving the wiki or backgrounding the app. */
@Composable
internal fun WikiAttachmentLifecycle(cancel: () -> Unit) {
    val currentCancel by rememberUpdatedState(cancel)
    val owner = LocalLifecycleOwner.current
    DisposableEffect(owner) {
        val observer = LifecycleEventObserver { _, event ->
            if (event == Lifecycle.Event.ON_STOP) currentCancel()
        }
        owner.lifecycle.addObserver(observer)
        onDispose { owner.lifecycle.removeObserver(observer); currentCancel() }
    }
}
