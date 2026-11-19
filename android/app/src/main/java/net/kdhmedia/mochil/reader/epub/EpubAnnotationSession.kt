package net.kdhmedia.mochil.reader.epub

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import net.kdhmedia.mochil.auth.DeviceCredential
import org.readium.r2.navigator.Decoration
import org.readium.r2.navigator.preferences.Theme
import org.readium.r2.shared.publication.Locator
import org.readium.r2.shared.publication.Publication

internal data class AnnotationRemoteBinding(
    val credential: DeviceCredential,
    val libraryId: String,
    val bookId: String,
)

internal class EpubAnnotationSession(
    private val controller: EpubAnnotationController,
) {
    var state by mutableStateOf(LocalAnnotationState())
        private set
    var notesExporting by mutableStateOf(false)
        private set

    private var remote by mutableStateOf<AnnotationRemoteBinding?>(null)

    val notesAvailable: Boolean
        get() = remote != null

    val deviceId: String
        get() = remote?.credential?.deviceId.orEmpty()

    suspend fun load(): LocalAnnotationState = controller.load().also { state = it }

    fun bindRemote(credential: DeviceCredential?, libraryId: String, bookId: String) {
        remote = credential
            ?.takeIf { libraryId.isNotBlank() && bookId.isNotBlank() }
            ?.let { AnnotationRemoteBinding(it, libraryId, bookId) }
    }

    suspend fun repairStableLocators(publication: Publication): LocalAnnotationState =
        controller.repairStableLocators(publication).also { state = it }

    suspend fun createHighlight(locator: Locator, spineIndex: Int): Boolean =
        controller.createHighlight(locator, deviceId, spineIndex).also { state = controller.state }

    suspend fun delete(id: String): Boolean =
        controller.delete(id, deviceId).also { state = controller.state }

    suspend fun editNote(id: String, note: String): Boolean =
        controller.editNote(id, note, deviceId).also { state = controller.state }

    suspend fun resolveConflict(id: String, keepLocal: Boolean): AnnotationConflictResolution =
        controller.resolveConflict(id, keepLocal).also { state = controller.state }

    suspend fun synchronize(publication: Publication): LocalAnnotationState? {
        val binding = remote ?: return null
        return controller.synchronize(
            publication,
            binding.credential,
            binding.libraryId,
            binding.bookId,
        ).also { state = it }
    }

    suspend fun exportToNotes(publication: Publication): AnnotationNotesExport? {
        val binding = remote ?: return null
        if (notesExporting) return null
        notesExporting = true
        return try {
            controller.exportToNotes(
                publication,
                binding.credential,
                binding.libraryId,
                binding.bookId,
            ).also { state = it.state }
        } finally {
            notesExporting = false
        }
    }

    fun decorations(theme: Theme): List<Decoration> = state.visible.mapNotNull { annotation ->
        annotation.readiumLocator()?.let { locator ->
            Decoration(annotation.id, locator, Decoration.Style.Highlight(highlightColor(theme)))
        }
    }

    private fun highlightColor(theme: Theme): Int = when (theme) {
        Theme.DARK -> 0xFFFFB38C.toInt()
        Theme.SEPIA -> 0xFFD47E45.toInt()
        Theme.LIGHT -> 0xFFE38252.toInt()
    }
}
