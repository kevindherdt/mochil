package net.kdhmedia.mochil.catalogue

import android.app.Application
import android.net.Uri
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import net.kdhmedia.mochil.audio.OfflineAudioStore

class DownloadLocationViewModel(application: Application) : AndroidViewModel(application) {
    private val store = DownloadLocationStore(application)
    private val mutableState = MutableStateFlow(store.current())
    val state: StateFlow<DownloadLocation> = mutableState.asStateFlow()

    init {
        recoverSelectedFolder(store.current())
    }

    fun select(uri: Uri?) {
        if (uri == null) return
        val selected = runCatching { store.select(uri) }
            .getOrElse {
                mutableState.value = store.current().copy(message = "Mochil could not keep access to that folder.", messageIsError = true)
                return
            }
        mutableState.value = selected
        recoverSelectedFolder(selected)
    }

    fun useAppStorage() { mutableState.value = store.useAppStorage() }
    fun refresh() { mutableState.value = store.current() }

    private fun recoverSelectedFolder(selected: DownloadLocation) {
        if (selected.treeUri.isNullOrBlank()) return
        viewModelScope.launch {
            val recovered = withContext(Dispatchers.IO) { OfflineAudioStore(getApplication()).recoverExternalDownloads() }
            if (recovered > 0 && mutableState.value.treeUri == selected.treeUri) {
                mutableState.value = selected.copy(
                    message = "Recovered $recovered music, podcast, or audiobook download${if (recovered == 1) "" else "s"} from this folder.",
                    messageIsError = false,
                )
            }
        }
    }
}
