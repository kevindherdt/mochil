package net.kdhmedia.mochil.wiki

import java.io.File
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.launch
import net.kdhmedia.mochil.auth.DeviceCredential

/** Session-owned attachment preparation; late results cannot launch a reader. */
internal class WikiPdfOpening(
    private val scope: CoroutineScope,
    private val prepare: suspend (DeviceCredential, String, WikiEntry) -> File,
    private val opened: (File, WikiEntry, String) -> Unit,
    private val status: (String?) -> Unit,
    private val failure: (String) -> Unit,
) {
    private var job: Job? = null
    private var generation = 0L

    fun cancel() {
        generation++
        job?.cancel()
        job = null
        status(null)
    }

    fun openLink(target: String, state: WikiState, credential: DeviceCredential?): Boolean {
        // Detect the actual extension before Markdown-note normalization.
        val path = target.substringBefore('|').substringBefore('#').trim()
        if (!path.endsWith(".pdf", true)) return false
        cancel()
        val entry = WikiFileIndex(state.entries, setOf("pdf")).resolve(state.selectedPath, target)
        if (entry == null || credential == null) {
            failure("This PDF could not be matched to an attachment in the connected vault.")
            return true
        }
        val current = generation
        status("Opening PDF…")
        job = scope.launch {
            try {
                val file = prepare(credential, state.libraryId, entry)
                if (generation == current) opened(file, entry, "local:wiki-pdf-${wikiPdfIdentity(credential, state.libraryId, entry)}")
            } catch (cancelled: CancellationException) { throw cancelled }
            catch (error: Exception) {
                if (generation == current) failure(error.message ?: "Could not open this PDF. Try again.")
            } finally {
                if (generation == current) { status(null); job = null }
            }
        }
        return true
    }
}
