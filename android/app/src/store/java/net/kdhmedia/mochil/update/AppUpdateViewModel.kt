package net.kdhmedia.mochil.update

import android.app.Application
import androidx.lifecycle.AndroidViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow

/** Store distributions receive updates exclusively from their installed store. */
class AppUpdateViewModel(application: Application) : AndroidViewModel(application) {
    private val mutableState = MutableStateFlow(AppUpdateState())
    val state: StateFlow<AppUpdateState> = mutableState.asStateFlow()

    fun check() { mutableState.value = AppUpdateState(message = "Updates are managed by your app store.") }
    fun downloadAndInstall() = check()
}
