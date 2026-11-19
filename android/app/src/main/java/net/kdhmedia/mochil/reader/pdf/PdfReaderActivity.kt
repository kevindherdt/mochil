package net.kdhmedia.mochil.reader.pdf

import android.content.ActivityNotFoundException
import android.content.Intent
import android.os.Bundle
import android.view.KeyEvent
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.activity.viewModels
import androidx.core.content.FileProvider
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.getValue
import java.io.File
import net.kdhmedia.mochil.ui.theme.MochilTheme
import net.kdhmedia.mochil.reader.ReaderHardwareNavigationPreferences
import net.kdhmedia.mochil.reader.ReaderPageTurn
import net.kdhmedia.mochil.reader.FixedLayoutLocalProgressStore
import net.kdhmedia.mochil.reader.FixedLayoutReaderKind
import net.kdhmedia.mochil.reader.fixedLayoutProgressIdentity
import net.kdhmedia.mochil.reader.fixedLayoutStorageKey
import net.kdhmedia.mochil.reader.opensReaderControls
import net.kdhmedia.mochil.reader.readerPageTurn
import kotlinx.coroutines.flow.MutableSharedFlow

class PdfReaderActivity : ComponentActivity() {
    private val progress: PdfProgressViewModel by viewModels()
    private val pageTurns = MutableSharedFlow<ReaderPageTurn>(extraBufferCapacity = 1)
    private val controlRequests = MutableSharedFlow<Unit>(extraBufferCapacity = 1)
    private val hardwarePreferences by lazy { ReaderHardwareNavigationPreferences(this) }
    private val localProgress by lazy { FixedLayoutLocalProgressStore(this, FixedLayoutReaderKind.Pdf) }
    private var progressKey = ""
    private var latestPage: Int? = null

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        val requested = intent.getStringExtra(EXTRA_PATH)?.let(::File)
        val root = File(filesDir, "publications").canonicalFile
        val file = requested?.canonicalFile?.takeIf { it.isFile && it.path.startsWith(root.path + File.separator) }
        if (file == null) {
            finish()
            return
        }
        enableEdgeToEdge()
        val legacyProgressKey = intent.getStringExtra(EXTRA_PROGRESS_KEY).orEmpty()
        val progressIdentity = fixedLayoutProgressIdentity(
            intent.getStringExtra(EXTRA_LIBRARY_ID).orEmpty(),
            intent.getStringExtra(EXTRA_BOOK_ID).orEmpty(),
            legacyProgressKey,
        )
        progressKey = progressIdentity?.let { fixedLayoutStorageKey(it.libraryId, it.bookId) } ?: legacyProgressKey
        localProgress.migrateLegacyKey(legacyProgressKey, progressKey)
        setContent {
            MochilTheme {
                val progressState by progress.state.collectAsStateWithLifecycle()
                PdfReaderScreen(
                    file = file,
                    title = intent.getStringExtra(EXTRA_TITLE).orEmpty(),
                    progressKey = progressKey,
                    requestedPage = progressState.requestedPage,
                    onRequestedPageConsumed = progress::consumedRequestedPage,
                    onReady = { pageCount, localPage -> latestPage = localPage; progressIdentity?.let { progress.open(it.libraryId, it.bookId, pageCount, localPage, FixedLayoutLocatorKind.Pdf) } },
                    onPage = { page -> latestPage = page; progress.pageChanged(page) },
                    onClose = ::finish,
                    onOpenExternal = { openExternal(file) },
                    pageTurns = pageTurns,
                    controlRequests = controlRequests,
                )
                progressState.prompt?.let { prompt ->
                    AlertDialog(
                        onDismissRequest = {},
                        title = { Text("Reading position changed") },
                        text = { Text("Another device is on page ${prompt.remotePage + 1}. This device is on page ${prompt.localPage + 1}.") },
                        confirmButton = { TextButton(onClick = progress::useRemote) { Text("Go to page ${prompt.remotePage + 1}") } },
                        dismissButton = { TextButton(onClick = progress::keepLocal) { Text("Keep page ${prompt.localPage + 1}") } },
                    )
                }
            }
        }
    }

    override fun onKeyDown(keyCode: Int, event: KeyEvent): Boolean {
        if (opensReaderControls(keyCode)) {
            if (event.repeatCount == 0) controlRequests.tryEmit(Unit)
            return true
        }
        val turn = readerPageTurn(keyCode, hardwarePreferences.volumeButtonsEnabled()) ?: return super.onKeyDown(keyCode, event)
        if (event.repeatCount == 0) pageTurns.tryEmit(turn)
        return true
    }

    override fun onResume() {
        super.onResume()
        progress.retryPending()
    }

    override fun onPause() {
        latestPage?.let { localProgress.saveNow(progressKey, it) }
        super.onPause()
    }

    override fun onKeyUp(keyCode: Int, event: KeyEvent): Boolean =
        if (readerPageTurn(keyCode, hardwarePreferences.volumeButtonsEnabled()) != null) true else super.onKeyUp(keyCode, event)

    private fun openExternal(file: File) {
        val uri = FileProvider.getUriForFile(this, "$packageName.files", file)
        val view = Intent(Intent.ACTION_VIEW).setDataAndType(uri, "application/pdf").addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION)
        try {
            startActivity(Intent.createChooser(view, "Open PDF with"))
        } catch (_: ActivityNotFoundException) {
            // The native reader remains open when no external PDF app exists.
        }
    }

    companion object {
        const val EXTRA_PATH = "publication_path"
        const val EXTRA_TITLE = "publication_title"
        const val EXTRA_PROGRESS_KEY = "publication_progress_key"
        const val EXTRA_LIBRARY_ID = "publication_library_id"
        const val EXTRA_BOOK_ID = "publication_book_id"
    }
}
