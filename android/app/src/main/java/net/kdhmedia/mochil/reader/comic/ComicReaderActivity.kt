package net.kdhmedia.mochil.reader.comic

import android.os.Bundle
import android.view.KeyEvent
import java.io.File
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.activity.viewModels
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.getValue
import androidx.lifecycle.lifecycleScope
import kotlinx.coroutines.launch
import net.kdhmedia.mochil.auth.DeviceCredentialStore
import net.kdhmedia.mochil.catalogue.BookSummary
import net.kdhmedia.mochil.catalogue.LibraryNavigationStore
import net.kdhmedia.mochil.catalogue.PublicationOpener
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
import net.kdhmedia.mochil.reader.pdf.PdfProgressViewModel
import net.kdhmedia.mochil.reader.pdf.FixedLayoutLocatorKind

class ComicReaderActivity : ComponentActivity() {
    private val progress: PdfProgressViewModel by viewModels()
    private val pageTurns = MutableSharedFlow<ReaderPageTurn>(extraBufferCapacity = 1)
    private val controlRequests = MutableSharedFlow<Unit>(extraBufferCapacity = 1)
    private val hardwarePreferences by lazy { ReaderHardwareNavigationPreferences(this) }
    private val localProgress by lazy { FixedLayoutLocalProgressStore(this, FixedLayoutReaderKind.Comic) }
    private var progressKey = ""
    private var latestPage: Int? = null
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        val requested = intent.getStringExtra(EXTRA_PATH)?.let(::File)
        val root = File(filesDir, "publications").canonicalFile
        val file = requested?.canonicalFile?.takeIf { it.isFile && it.path.startsWith(root.path + File.separator) }
        val credential = DeviceCredentialStore(this).load()
        val streamUrl = intent.getStringExtra(EXTRA_STREAM_URL)?.takeIf { it.startsWith("/") && it.contains("{pageNumber}") }
        val source: ComicSource? = when {
            file != null -> ComicSource.Local(file)
            streamUrl != null && credential != null ->
                ComicSource.Streamed(credential.serverUrl, streamUrl, intent.getIntExtra(EXTRA_PAGE_COUNT, 0), credential.token)
            else -> null
        }
        if (source == null) {
            finish()
            return
        }
        enableEdgeToEdge()
        @Suppress("DEPRECATION")
        val nextBook = intent.getSerializableExtra(EXTRA_NEXT_BOOK) as? BookSummary
        val nextChapter: (() -> Unit)? = nextBook?.let { next ->
            {
                val credential = DeviceCredentialStore(this).load()
                if (credential != null) lifecycleScope.launch {
                    runCatching { PublicationOpener(this@ComicReaderActivity).prepare(credential, next) }
                        .onSuccess {
                            startActivity(it)
                            finish()
                        }
                }
            }
        }
        val legacyProgressKey = intent.getStringExtra(EXTRA_PROGRESS_KEY).orEmpty()
        val progressIdentity = fixedLayoutProgressIdentity(
            intent.getStringExtra(EXTRA_LIBRARY_ID).orEmpty(),
            intent.getStringExtra(EXTRA_BOOK_ID).orEmpty(),
            legacyProgressKey,
        )
        progressKey = progressIdentity?.let { fixedLayoutStorageKey(it.libraryId, it.bookId) } ?: legacyProgressKey
        localProgress.migrateLegacyKey(legacyProgressKey, progressKey)
        // The library kind comes from the last library list the app cached, so it is also known offline. Files
        // opened without a library id (local files, WebDAV) keep the old right-to-left default.
        val libraryKind = intent.getStringExtra(EXTRA_LIBRARY_ID)?.takeIf { it.isNotBlank() }?.let { id ->
            LibraryNavigationStore(this).cachedLibraries().firstOrNull { it.id == id }?.kind
        }
        setContent {
            MochilTheme {
                val progressState by progress.state.collectAsStateWithLifecycle()
                ComicReaderScreen(
                    source = source,
                    title = intent.getStringExtra(EXTRA_TITLE).orEmpty(),
                    progressKey = progressKey,
                    requestedPage = progressState.requestedPage,
                    onRequestedPageConsumed = progress::consumedRequestedPage,
                    onReady = { pageCount, localPage -> latestPage = localPage; progressIdentity?.let { progress.open(it.libraryId, it.bookId, pageCount, localPage, FixedLayoutLocatorKind.Comic) } },
                    onPage = { page -> latestPage = page; progress.pageChanged(page) },
                    onClose = ::finish,
                    onNextChapter = nextChapter,
                    pageTurns = pageTurns,
                    controlRequests = controlRequests,
                    libraryKind = libraryKind,
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

    companion object {
        const val EXTRA_PATH = "publication_path"
        const val EXTRA_STREAM_URL = "publication_stream_url"
        const val EXTRA_PAGE_COUNT = "publication_page_count"
        const val EXTRA_TITLE = "publication_title"
        const val EXTRA_PROGRESS_KEY = "publication_progress_key"
        const val EXTRA_LIBRARY_ID = "publication_library_id"
        const val EXTRA_BOOK_ID = "publication_book_id"
        const val EXTRA_NEXT_BOOK = "next_publication"
    }
}
